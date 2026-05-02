package handlers

import (
	"net/http"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type FeeHandler struct{}

func NewFeeHandler() *FeeHandler {
	return &FeeHandler{}
}

func (h *FeeHandler) GetFeeCategories(c *gin.Context) {
	schoolID := scopedSchoolID(c)
	var categories []models.FeeCategory
	query := database.DB.Preload("School")
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	query.Find(&categories)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: categories})
}

func (h *FeeHandler) CreateFeeCategory(c *gin.Context) {
	var req struct {
		SchoolID     string `json:"school_id" binding:"required"`
		CategoryName string `json:"category_name" binding:"required"`
		Frequency    string `json:"frequency" binding:"required"`
		IsRefundable bool   `json:"is_refundable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cat := models.FeeCategory{
		SchoolID:     scopedSchoolID(c),
		CategoryName: req.CategoryName,
		Frequency:    req.Frequency,
		IsRefundable: req.IsRefundable,
	}

	if err := database.DB.Create(&cat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create fee category"})
		return
	}

	id := cat.ID
	auditAction(c, "fees", "create", "fee_categories", &id)
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: cat})
}

func (h *FeeHandler) GetFeeStructures(c *gin.Context) {
	schoolID := scopedSchoolID(c)
	yearID := c.Query("academic_year_id")
	gradeID := c.Query("grade_id")

	var structures []models.FeeStructure
	query := database.DB.Preload("FeeCategory").Preload("Grade").Preload("AcademicYear")
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	if yearID != "" {
		query = query.Where("academic_year_id = ?", yearID)
	}
	if gradeID != "" {
		query = query.Where("grade_id = ?", gradeID)
	}
	query.Find(&structures)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: structures})
}

func (h *FeeHandler) CreateFeeStructure(c *gin.Context) {
	var req models.CreateFeeStructureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	structure := models.FeeStructure{
		SchoolID:       scopedSchoolID(c),
		AcademicYearID: req.AcademicYearID,
		GradeID:        req.GradeID,
		FeeCategoryID:  req.FeeCategoryID,
		Amount:         req.Amount,
		DueDay:         req.DueDay,
		LateFinePerDay: req.LateFinePerDay,
	}

	if err := database.DB.Create(&structure).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create fee structure"})
		return
	}

	id := structure.ID
	auditAction(c, "fees", "create", "fee_structures", &id)
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: structure})
}

func (h *FeeHandler) GetInvoices(c *gin.Context) {
	schoolID := scopedSchoolID(c)
	studentID := c.Query("student_id")
	status := c.Query("status")
	page, pageSize := parsePagination(c)

	// FeeInvoice has no school_id column; school boundary is enforced by
	// joining through students so that only invoices belonging to the
	// caller's school are returned.
	var invoices []models.FeeInvoice
	var total int64

	query := database.DB.Model(&models.FeeInvoice{}).
		Joins("JOIN students ON students.id = fee_invoices.student_id").
		Where("students.school_id = ?", schoolID)
	if currentRole(c) == "parent" {
		query = query.Where("fee_invoices.student_id IN (?)", linkedStudentSubquery(c))
	}

	if studentID != "" {
		query = query.Where("fee_invoices.student_id = ?", studentID)
	}
	if status != "" {
		query = query.Where("fee_invoices.status = ?", status)
	}

	query.Count(&total)

	if err := query.
		Preload("Student").
		Preload("Items").
		Preload("Payments").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&invoices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to fetch invoices",
		})
		return
	}

	c.JSON(http.StatusOK, paginationResult(page, pageSize, total, invoices))
}

func (h *FeeHandler) CreateInvoice(c *gin.Context) {
	var req struct {
		StudentID      string  `json:"student_id" binding:"required"`
		AcademicYearID string  `json:"academic_year_id" binding:"required"`
		InvoiceNumber  string  `json:"invoice_number" binding:"required"`
		InvoiceDate    string  `json:"invoice_date" binding:"required"`
		DueDate        string  `json:"due_date" binding:"required"`
		TotalAmount    float64 `json:"total_amount" binding:"required"`
		DiscountAmount float64 `json:"discount_amount"`
		NetAmount      float64 `json:"net_amount" binding:"required"`
		Items          []struct {
			FeeCategoryID string  `json:"fee_category_id" binding:"required"`
			Amount        float64 `json:"amount" binding:"required"`
			Description   string  `json:"description"`
		}
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !canAccessStudentID(c, req.StudentID) {
		forbid(c, "student access denied")
		return
	}

	invoiceDate, _ := time.Parse("2006-01-02", req.InvoiceDate)
	dueDate, _ := time.Parse("2006-01-02", req.DueDate)

	invoice := models.FeeInvoice{
		StudentID:      req.StudentID,
		AcademicYearID: req.AcademicYearID,
		InvoiceNumber:  req.InvoiceNumber,
		InvoiceDate:    invoiceDate,
		DueDate:        dueDate,
		TotalAmount:    req.TotalAmount,
		DiscountAmount: req.DiscountAmount,
		NetAmount:      req.NetAmount,
		PaidAmount:     0,
		Balance:        req.NetAmount,
		Status:         "pending",
	}

	if err := database.DB.Create(&invoice).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create invoice"})
		return
	}

	for _, item := range req.Items {
		invoiceItem := models.FeeInvoiceItem{
			InvoiceID:     invoice.ID,
			FeeCategoryID: item.FeeCategoryID,
			Amount:        item.Amount,
			Description:   item.Description,
		}
		database.DB.Create(&invoiceItem)
	}

	id := invoice.ID
	auditAction(c, "fees", "create", "fee_invoices", &id)
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: invoice})
}

func (h *FeeHandler) RecordPayment(c *gin.Context) {
	var req struct {
		InvoiceID     string  `json:"invoice_id" binding:"required"`
		ReceiptNumber string  `json:"receipt_number" binding:"required"`
		AmountPaid    float64 `json:"amount_paid" binding:"required"`
		PaymentDate   string  `json:"payment_date" binding:"required"`
		PaymentMode   string  `json:"payment_mode" binding:"required"`
		TransactionID string  `json:"transaction_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	paymentDate, _ := time.Parse("2006-01-02", req.PaymentDate)
	var invoice models.FeeInvoice
	if err := database.DB.Joins("JOIN students ON students.id = fee_invoices.student_id").
		Where("fee_invoices.id = ? AND students.school_id = ?", req.InvoiceID, scopedSchoolID(c)).
		First(&invoice).Error; err != nil {
		forbid(c, "invoice access denied")
		return
	}

	payment := models.Payment{
		InvoiceID:     req.InvoiceID,
		ReceiptNumber: req.ReceiptNumber,
		AmountPaid:    req.AmountPaid,
		PaymentDate:   paymentDate,
		PaymentMode:   req.PaymentMode,
		TransactionID: req.TransactionID,
	}

	if err := database.DB.Create(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record payment"})
		return
	}

	invoice.PaidAmount += req.AmountPaid
	invoice.Balance -= req.AmountPaid
	if invoice.Balance <= 0 {
		invoice.Status = "paid"
		invoice.Balance = 0
	} else {
		invoice.Status = "partial"
	}
	database.DB.Save(&invoice)

	id := payment.ID
	auditAction(c, "fees", "create", "payments", &id)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: payment})
}

func (h *FeeHandler) GetConcessions(c *gin.Context) {
	studentID := c.Query("student_id")
	var concessions []models.FeeConcession
	query := database.DB.Preload("FeeCategory").Preload("Student")
	if currentRole(c) == "parent" {
		query = query.Where("student_id IN (?)", linkedStudentSubquery(c))
	}
	if studentID != "" {
		query = query.Where("student_id = ?", studentID)
	}
	query.Find(&concessions)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: concessions})
}
