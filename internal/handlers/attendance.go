package handlers

import (
	"net/http"
	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type AttendanceHandler struct{}

func NewAttendanceHandler() *AttendanceHandler {
	return &AttendanceHandler{}
}

func (h *AttendanceHandler) GetAttendanceSessions(c *gin.Context) {
	sectionID := c.Query("section_id")
	date := c.Query("date")

	var sessions []models.AttendanceSession
	query := database.DB.Preload("Subject").Preload("Staff")
	if sectionID != "" {
		query = query.Where("section_id = ?", sectionID)
	}
	if date != "" {
		query = query.Where("date = ?", date)
	}
	query.Find(&sessions)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: sessions})
}

func (h *AttendanceHandler) CreateAttendanceSession(c *gin.Context) {
	var req struct {
		SectionID       string `json:"section_id" binding:"required"`
		SubjectID       string `json:"subject_id" binding:"required"`
		StaffID         string `json:"staff_id" binding:"required"`
		Date            string `json:"date" binding:"required"`
		PeriodNumber    int    `json:"period_number"`
		TimetableSlotID string `json:"timetable_slot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session := models.AttendanceSession{
		SectionID:       req.SectionID,
		SubjectID:       req.SubjectID,
		StaffID:         req.StaffID,
		PeriodNumber:    req.PeriodNumber,
		TimetableSlotID: &req.TimetableSlotID,
		TotalStudents:   0,
		PresentCount:    0,
		IsFinalized:     false,
	}

	if req.TimetableSlotID != "" {
		session.TimetableSlotID = &req.TimetableSlotID
	}

	if err := database.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: session})
}

func (h *AttendanceHandler) MarkStudentAttendance(c *gin.Context) {
	sessionID := c.Param("session_id")
	var req struct {
		Attendances []struct {
			StudentID    string `json:"student_id" binding:"required"`
			EnrollmentID string `json:"enrollment_id" binding:"required"`
			Status       string `json:"status" binding:"required"`
			Reason       string `json:"reason"`
		} `json:"attendances" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, att := range req.Attendances {
		attendance := models.StudentAttendance{
			SessionID:    sessionID,
			StudentID:    att.StudentID,
			EnrollmentID: att.EnrollmentID,
			Status:       att.Status,
			Reason:       att.Reason,
		}
		database.DB.Create(&attendance)
	}

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Attendance marked successfully"})
}

func (h *AttendanceHandler) GetStudentAttendanceSummary(c *gin.Context) {
	studentID := c.Query("student_id")
	yearID := c.Query("academic_year_id")
	termID := c.Query("term_id")

	var summary models.AttendanceSummary
	query := database.DB.Where("student_id = ?", studentID)
	if yearID != "" {
		query = query.Where("academic_year_id = ?", yearID)
	}
	if termID != "" {
		query = query.Where("term_id = ?", termID)
	}
	query.First(&summary)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: summary})
}

func (h *AttendanceHandler) MarkStaffAttendance(c *gin.Context) {
	var req struct {
		StaffID     string `json:"staff_id" binding:"required"`
		Date        string `json:"date" binding:"required"`
		Status      string `json:"status" binding:"required"`
		CheckIn     string `json:"check_in"`
		CheckOut    string `json:"check_out"`
		BiometricID string `json:"biometric_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	attendance := models.StaffAttendance{
		StaffID:     req.StaffID,
		Status:      req.Status,
		BiometricID: req.BiometricID,
	}

	if err := database.DB.Create(&attendance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark attendance"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: attendance})
}