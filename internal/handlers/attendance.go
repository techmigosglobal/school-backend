package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AttendanceHandler struct{}

func NewAttendanceHandler() *AttendanceHandler {
	return &AttendanceHandler{}
}

func (h *AttendanceHandler) GetAttendanceSessions(c *gin.Context) {
	sectionID := c.Query("section_id")
	date := c.Query("date")

	var sessions []models.AttendanceSession
	query := database.DB.Model(&models.AttendanceSession{}).
		Joins("JOIN sections ON sections.id = attendance_sessions.section_id").
		Joins("JOIN grades ON grades.id = sections.grade_id").
		Where("grades.school_id = ?", scopedSchoolID(c)).
		Preload("Subject").Preload("Staff")
	switch currentRole(c) {
	case "teacher":
		query = query.Where("attendance_sessions.section_id IN (?)", teacherAssignedSectionSubquery(c))
	case "parent":
		query = query.Joins("JOIN student_attendances ON student_attendances.session_id = attendance_sessions.id").
			Where("student_attendances.student_id IN (?)", linkedStudentSubquery(c)).
			Group("attendance_sessions.id")
	}
	if sectionID != "" {
		query = query.Where("attendance_sessions.section_id = ?", sectionID)
	}
	if date != "" {
		parsed, err := time.Parse("2006-01-02", date)
		if err != nil {
			fail(c, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
			return
		}
		query = query.Where("date >= ? AND date < ?", parsed, parsed.AddDate(0, 0, 1))
	}
	if err := query.Find(&sessions).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to load attendance sessions")
		return
	}

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

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}
	if req.PeriodNumber < 1 {
		fail(c, http.StatusBadRequest, "period_number must be greater than zero")
		return
	}
	if currentRole(c) == "teacher" {
		if currentStaffID(c) != req.StaffID || !teacherCanAccessSectionSubject(c, req.SectionID, req.SubjectID) {
			forbid(c, "attendance session access denied")
			return
		}
	}
	if scopedSchoolID(c) != "" {
		var sectionCount int64
		database.DB.Model(&models.Section{}).
			Joins("JOIN grades ON grades.id = sections.grade_id").
			Where("sections.id = ? AND grades.school_id = ?", req.SectionID, scopedSchoolID(c)).
			Count(&sectionCount)
		if sectionCount == 0 {
			fail(c, http.StatusBadRequest, "section does not belong to school")
			return
		}
	}

	session := models.AttendanceSession{
		SectionID:     req.SectionID,
		SubjectID:     req.SubjectID,
		StaffID:       req.StaffID,
		Date:          date,
		PeriodNumber:  req.PeriodNumber,
		TotalStudents: 0,
		PresentCount:  0,
		IsFinalized:   false,
	}

	if req.TimetableSlotID != "" {
		session.TimetableSlotID = &req.TimetableSlotID
	}

	if err := database.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	id := session.ID
	auditAction(c, "attendance", "create", "attendance_sessions", &id)
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
	if len(req.Attendances) == 0 {
		fail(c, http.StatusBadRequest, "attendances must contain at least one record")
		return
	}

	var session models.AttendanceSession
	if err := database.DB.First(&session, "id = ?", sessionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "Attendance session not found")
			return
		}
		fail(c, http.StatusInternalServerError, "Failed to load attendance session")
		return
	}
	if !teacherCanAccessSection(c, session.SectionID) {
		forbid(c, "attendance session access denied")
		return
	}
	if currentRole(c) == "teacher" && currentStaffID(c) != session.StaffID && !teacherCanAccessSectionSubject(c, session.SectionID, session.SubjectID) {
		forbid(c, "attendance session access denied")
		return
	}

	now := time.Now().UTC()
	markedBy := c.GetString("user_id")
	presentCount := 0
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := tx.Model(&models.StudentAttendance{}).Where("session_id = ?", sessionID).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return errDuplicateAttendance
		}
		seenStudents := map[string]bool{}
		for _, att := range req.Attendances {
			status := strings.TrimSpace(att.Status)
			if !validAttendanceStatus(status) {
				return errInvalidAttendanceStatus
			}
			if seenStudents[att.StudentID] {
				return errDuplicateAttendance
			}
			seenStudents[att.StudentID] = true
			if !studentEnrollmentInSection(att.StudentID, att.EnrollmentID, session.SectionID) {
				return errAttendanceAccessDenied
			}
			if strings.EqualFold(status, "present") || strings.EqualFold(status, "late") {
				presentCount++
			}
			attendance := models.StudentAttendance{
				SessionID:    sessionID,
				StudentID:    att.StudentID,
				EnrollmentID: att.EnrollmentID,
				Status:       status,
				Reason:       att.Reason,
				MarkedAt:     now,
				MarkedBy:     &markedBy,
			}
			if err := tx.Create(&attendance).Error; err != nil {
				return err
			}
		}
		session.TotalStudents = len(req.Attendances)
		session.PresentCount = presentCount
		return tx.Save(&session).Error
	})
	if err != nil {
		if err == errInvalidAttendanceStatus {
			fail(c, http.StatusBadRequest, "Invalid attendance status")
			return
		}
		if err == errDuplicateAttendance {
			fail(c, http.StatusConflict, "Attendance already marked for this session")
			return
		}
		if err == errAttendanceAccessDenied {
			forbid(c, "student does not belong to attendance session")
			return
		}
		fail(c, http.StatusInternalServerError, "Failed to mark attendance")
		return
	}

	auditAction(c, "attendance", "update", "student_attendances", &sessionID)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Attendance marked successfully"})
}

var errInvalidAttendanceStatus = errors.New("invalid attendance status")
var errDuplicateAttendance = errors.New("duplicate attendance")
var errAttendanceAccessDenied = errors.New("attendance access denied")

func validAttendanceStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "present", "absent", "late", "half-day", "leave":
		return true
	default:
		return false
	}
}

func combineDateAndClock(date time.Time, clock string) (time.Time, error) {
	parsed, err := time.Parse("15:04:05", clock)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(
		date.Year(), date.Month(), date.Day(),
		parsed.Hour(), parsed.Minute(), parsed.Second(), 0,
		time.UTC,
	), nil
}

func (h *AttendanceHandler) GetStudentAttendanceSummary(c *gin.Context) {
	studentID := c.Query("student_id")
	yearID := c.Query("academic_year_id")
	termID := c.Query("term_id")

	var summary models.AttendanceSummary
	query := database.DB.Where("student_id = ?", studentID)
	if !canAccessStudentID(c, studentID) {
		forbid(c, "student access denied")
		return
	}
	if yearID != "" {
		query = query.Where("academic_year_id = ?", yearID)
	}
	if termID != "" {
		query = query.Where("term_id = ?", termID)
	}
	if err := query.First(&summary).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "Attendance summary not found")
			return
		}
		fail(c, http.StatusInternalServerError, "Failed to load attendance summary")
		return
	}

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

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	attendance := models.StaffAttendance{
		StaffID:     req.StaffID,
		Date:        date,
		Status:      req.Status,
		BiometricID: req.BiometricID,
	}

	// Parse check_in if provided
	if req.CheckIn != "" {
		checkIn, err := combineDateAndClock(date, req.CheckIn)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid check_in format. Use HH:MM:SS"})
			return
		}
		attendance.CheckIn = &checkIn
	}

	// Parse check_out if provided
	if req.CheckOut != "" {
		checkOut, err := combineDateAndClock(date, req.CheckOut)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid check_out format. Use HH:MM:SS"})
			return
		}
		attendance.CheckOut = &checkOut
	}

	if err := database.DB.Create(&attendance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark attendance"})
		return
	}

	id := attendance.ID
	auditAction(c, "attendance", "create", "staff_attendances", &id)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: attendance})
}
