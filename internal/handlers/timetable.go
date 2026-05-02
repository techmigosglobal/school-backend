package handlers

import (
	"net/http"
	"strconv"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type TimetableHandler struct{}

func NewTimetableHandler() *TimetableHandler {
	return &TimetableHandler{}
}

func (h *TimetableHandler) GetTimetableSlots(c *gin.Context) {
	sectionID := c.Query("section_id")
	yearID := c.Query("academic_year_id")
	dayOfWeek := c.Query("day_of_week")
	staffID := c.Query("staff_id")

	var slots []models.TimetableSlot
	query := database.DB.Preload("Subject").Preload("Staff").Preload("Room")
	if sectionID != "" {
		query = query.Where("section_id = ?", sectionID)
	}
	if yearID != "" {
		query = query.Where("academic_year_id = ?", yearID)
	}
	if dayOfWeek != "" {
		query = query.Where("day_of_week = ?", dayOfWeek)
	}
	if staffID != "" {
		query = query.Where("staff_id = ?", staffID)
	}
	query.Order("day_of_week, period_number").Find(&slots)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: slots})
}

func (h *TimetableHandler) CreateTimetableSlot(c *gin.Context) {
	var req models.CreateTimetableSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.DayOfWeek < 1 || req.DayOfWeek > 7 {
		fail(c, http.StatusBadRequest, "day_of_week must be between 1 and 7")
		return
	}
	if req.PeriodNumber < 1 {
		fail(c, http.StatusBadRequest, "period_number must be greater than zero")
		return
	}

	slot := models.TimetableSlot{
		SectionID:      req.SectionID,
		AcademicYearID: req.AcademicYearID,
		TermID:         req.TermID,
		DayOfWeek:      req.DayOfWeek,
		PeriodNumber:   req.PeriodNumber,
		SubjectID:      req.SubjectID,
		StaffID:        req.StaffID,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		SlotType:       "regular",
	}

	if req.RoomID != "" {
		slot.RoomID = &req.RoomID
	}

	if err := database.DB.Create(&slot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create timetable slot"})
		return
	}

	id := slot.ID
	auditAction(c, "timetable", "create", "timetable_slots", &id)
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: slot})
}

func (h *TimetableHandler) UpdateTimetableSlot(c *gin.Context) {
	id := c.Param("id")
	var slot models.TimetableSlot
	if err := database.DB.First(&slot, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Timetable slot not found"})
		return
	}

	var req models.CreateTimetableSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slot.SubjectID = req.SubjectID
	slot.StaffID = req.StaffID
	slot.PeriodNumber = req.PeriodNumber
	slot.StartTime = req.StartTime
	slot.EndTime = req.EndTime
	if req.RoomID != "" {
		slot.RoomID = &req.RoomID
	}

	if err := database.DB.Save(&slot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update timetable slot"})
		return
	}

	auditAction(c, "timetable", "update", "timetable_slots", &id)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: slot})
}

func (h *TimetableHandler) DeleteTimetableSlot(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.TimetableSlot{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete timetable slot"})
		return
	}
	auditAction(c, "timetable", "delete", "timetable_slots", &id)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Timetable slot deleted successfully"})
}

func (h *TimetableHandler) GetSubstitutions(c *gin.Context) {
	date := c.Query("date")
	originalStaffID := c.Query("original_staff_id")

	var subs []models.Substitution
	query := database.DB.Preload("TimetableSlot").Preload("OriginalStaff").Preload("SubstituteStaff")
	if date != "" {
		parsed, err := time.Parse("2006-01-02", date)
		if err != nil {
			fail(c, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
			return
		}
		query = query.Where("date >= ? AND date < ?", parsed, parsed.AddDate(0, 0, 1))
	}
	if originalStaffID != "" {
		query = query.Where("original_staff_id = ?", originalStaffID)
	}
	query.Find(&subs)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: subs})
}

func (h *TimetableHandler) CreateSubstitution(c *gin.Context) {
	var req struct {
		TimetableSlotID   string `json:"timetable_slot_id" binding:"required"`
		Date              string `json:"date" binding:"required"`
		OriginalStaffID   string `json:"original_staff_id" binding:"required"`
		SubstituteStaffID string `json:"substitute_staff_id" binding:"required"`
		Reason            string `json:"reason"`
		ApprovedBy        string `json:"approved_by"`
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
	if req.OriginalStaffID == req.SubstituteStaffID {
		fail(c, http.StatusBadRequest, "substitute_staff_id must be different from original_staff_id")
		return
	}

	sub := models.Substitution{
		TimetableSlotID:   req.TimetableSlotID,
		Date:              date,
		OriginalStaffID:   req.OriginalStaffID,
		SubstituteStaffID: req.SubstituteStaffID,
		Reason:            req.Reason,
	}

	if req.ApprovedBy != "" {
		sub.ApprovedBy = &req.ApprovedBy
	}

	if err := database.DB.Create(&sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create substitution"})
		return
	}

	id := sub.ID
	auditAction(c, "timetable", "create", "substitutions", &id)
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: sub})
}

func (h *TimetableHandler) GetTimetableBySection(c *gin.Context) {
	sectionID := c.Param("section_id")
	yearID := c.Query("academic_year_id")

	var slots []models.TimetableSlot
	query := database.DB.Where("section_id = ?", sectionID).Preload("Subject").Preload("Staff")
	if yearID != "" {
		query = query.Where("academic_year_id = ?", yearID)
	}
	if err := query.Order("day_of_week, period_number").Find(&slots).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to load timetable")
		return
	}

	timetable := make(map[string]map[string]models.TimetableSlot)
	for _, slot := range slots {
		day := strconv.Itoa(slot.DayOfWeek)
		period := strconv.Itoa(slot.PeriodNumber)
		if timetable[day] == nil {
			timetable[day] = make(map[string]models.TimetableSlot)
		}
		timetable[day][period] = slot
	}

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: timetable})
}
