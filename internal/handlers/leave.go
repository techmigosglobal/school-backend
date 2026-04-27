package handlers

import (
	"net/http"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type LeaveHandler struct{}

func NewLeaveHandler() *LeaveHandler {
	return &LeaveHandler{}
}

func (h *LeaveHandler) GetLeaveTypes(c *gin.Context) {
	schoolID := scopedSchoolID(c)
	var leaveTypes []models.LeaveType
	query := database.DB.Preload("School")
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	query.Find(&leaveTypes)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: leaveTypes})
}

func (h *LeaveHandler) CreateLeaveType(c *gin.Context) {
	var req struct {
		SchoolID         string `json:"school_id" binding:"required"`
		LeaveName        string `json:"leave_name" binding:"required"`
		MaxDaysPerYear   int    `json:"max_days_per_year"`
		CarryForwardDays int    `json:"carry_forward_days"`
		IsPaid           bool   `json:"is_paid"`
		ApplicableTo     string `json:"applicable_to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	leaveType := models.LeaveType{
		SchoolID:         scopedSchoolID(c),
		LeaveName:        req.LeaveName,
		MaxDaysPerYear:   req.MaxDaysPerYear,
		CarryForwardDays: req.CarryForwardDays,
		IsPaid:           req.IsPaid,
		ApplicableTo:     req.ApplicableTo,
	}

	if err := database.DB.Create(&leaveType).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create leave type"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: leaveType})
}

func (h *LeaveHandler) GetLeaveApplications(c *gin.Context) {
	staffID := c.Query("staff_id")
	status := c.Query("status")

	var applications []models.LeaveApplication
	query := database.DB.Preload("Staff").Preload("LeaveType").Preload("Approver")
	if staffID != "" {
		query = query.Where("staff_id = ?", staffID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Find(&applications)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: applications})
}

func (h *LeaveHandler) CreateLeaveApplication(c *gin.Context) {
	var req models.CreateLeaveApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fromDate, _ := time.Parse("2006-01-02", req.FromDate)
	toDate, _ := time.Parse("2006-01-02", req.ToDate)

	var totalDays float64
	if fromDate.Equal(toDate) {
		totalDays = 1
	} else {
		totalDays = toDate.Sub(fromDate).Hours()/24 + 1
	}
	if req.HalfDay {
		totalDays = 0.5
	}

	application := models.LeaveApplication{
		StaffID:     req.StaffID,
		LeaveTypeID: req.LeaveTypeID,
		FromDate:    fromDate,
		ToDate:      toDate,
		HalfDay:     req.HalfDay,
		TotalDays:   totalDays,
		Reason:      req.Reason,
		Status:      "pending",
		AppliedAt:   time.Now(),
	}

	if err := database.DB.Create(&application).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create leave application"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: application})
}

func (h *LeaveHandler) ApproveLeaveApplication(c *gin.Context) {
	id := c.Param("id")
	var application models.LeaveApplication
	if err := database.DB.First(&application, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	application.Status = req.Status
	if req.Status == "rejected" {
		application.RejectionReason = req.Reason
	}

	if err := database.DB.Save(&application).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update application"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: application})
}

func (h *LeaveHandler) GetLeaveBalances(c *gin.Context) {
	staffID := c.Query("staff_id")
	yearID := c.Query("academic_year_id")

	var balances []models.LeaveBalance
	query := database.DB.Preload("LeaveType").Preload("Staff")
	if staffID != "" {
		query = query.Where("staff_id = ?", staffID)
	}
	if yearID != "" {
		query = query.Where("academic_year_id = ?", yearID)
	}
	query.Find(&balances)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: balances})
}

func (h *LeaveHandler) InitializeLeaveBalances(c *gin.Context) {
	var req struct {
		StaffID        string `json:"staff_id" binding:"required"`
		AcademicYearID string `json:"academic_year_id" binding:"required"`
		LeaveTypeID    string `json:"leave_type_id" binding:"required"`
		TotalEntitled  int    `json:"total_entitled" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	balance := models.LeaveBalance{
		StaffID:        req.StaffID,
		AcademicYearID: req.AcademicYearID,
		LeaveTypeID:    req.LeaveTypeID,
		TotalEntitled:  req.TotalEntitled,
		UsedDays:       0,
		PendingDays:    0,
		RemainingDays:  float64(req.TotalEntitled),
	}

	if err := database.DB.Create(&balance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize balance"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: balance})
}
