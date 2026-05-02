package handlers

import (
	"net/http"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type StaffHandler struct{}

func NewStaffHandler() *StaffHandler {
	return &StaffHandler{}
}

func (h *StaffHandler) GetStaff(c *gin.Context) {
	page, pageSize := parsePagination(c)
	schoolID := scopedSchoolID(c)
	deptID := c.Query("department_id")
	status := c.Query("status")

	var staff []models.Staff
	var total int64

	query := database.DB.Model(&models.Staff{})
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	if deptID != "" {
		query = query.Where("department_id = ?", deptID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	query = query.Preload("Department").Preload("Qualifications").Preload("Subjects").Offset((page - 1) * pageSize).Limit(pageSize)
	query.Find(&staff)

	c.JSON(http.StatusOK, paginationResult(page, pageSize, total, staff))
}

func (h *StaffHandler) GetStaffMember(c *gin.Context) {
	id := c.Param("id")
	var staff models.Staff
	if err := database.DB.Preload("Department").Preload("Qualifications").Preload("Subjects").Preload("Subjects.Subject").Preload("Documents").First(&staff, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Staff not found"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: staff})
}

func (h *StaffHandler) CreateStaff(c *gin.Context) {
	var req models.CreateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dob, _ := time.Parse("2006-01-02", req.DateOfBirth)
	joinDate, _ := time.Parse("2006-01-02", req.JoinDate)

	staff := models.Staff{
		SchoolID:       scopedSchoolID(c),
		StaffCode:      req.StaffCode,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
		Phone:          req.Phone,
		DateOfBirth:    dob,
		Gender:         req.Gender,
		DepartmentID:   &req.DepartmentID,
		Designation:    req.Designation,
		EmploymentType: req.EmploymentType,
		JoinDate:       joinDate,
		BasicSalary:    req.BasicSalary,
		Status:         "active",
	}

	if err := database.DB.Create(&staff).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create staff"})
		return
	}

	id := staff.ID
	auditAction(c, "staff", "create", "staff", &id)
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: staff})
}

func (h *StaffHandler) UpdateStaff(c *gin.Context) {
	id := c.Param("id")
	var staff models.Staff
	if err := database.DB.First(&staff, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Staff not found"})
		return
	}

	var req models.CreateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	staff.FirstName = req.FirstName
	staff.LastName = req.LastName
	staff.Email = req.Email
	staff.Phone = req.Phone
	staff.Designation = req.Designation
	staff.EmploymentType = req.EmploymentType
	staff.BasicSalary = req.BasicSalary

	if req.DateOfBirth != "" {
		staff.DateOfBirth, _ = time.Parse("2006-01-02", req.DateOfBirth)
	}
	if req.DepartmentID != "" {
		staff.DepartmentID = &req.DepartmentID
	}

	if err := database.DB.Save(&staff).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update staff"})
		return
	}

	auditAction(c, "staff", "update", "staff", &id)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: staff})
}

func (h *StaffHandler) DeleteStaff(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Staff{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete staff"})
		return
	}
	auditAction(c, "staff", "delete", "staff", &id)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Staff deleted successfully"})
}

func (h *StaffHandler) GetStaffLeaveBalance(c *gin.Context) {
	staffID := c.Param("id")
	var balances []models.LeaveBalance
	database.DB.Preload("LeaveType").Where("staff_id = ?", staffID).Find(&balances)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: balances})
}

func (h *StaffHandler) GetStaffAttendance(c *gin.Context) {
	staffID := c.Param("id")
	month := c.Query("month")
	year := c.Query("year")

	var attendance []models.StaffAttendance
	query := database.DB.Where("staff_id = ?", staffID)
	if month != "" {
		start, end, ok := monthYearRange(month, year)
		if ok {
			query = query.Where("date >= ? AND date < ?", start, end)
		}
	} else if year != "" {
		start, _, ok := monthYearRange("01", year)
		if ok {
			yearEnd := start.AddDate(1, 0, 0)
			query = query.Where("date >= ? AND date < ?", start, yearEnd)
		}
	}
	query.Find(&attendance)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: attendance})
}
