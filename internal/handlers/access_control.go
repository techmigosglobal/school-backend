package handlers

import (
	"net/http"
	"strings"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func currentRole(c *gin.Context) string {
	return strings.ToLower(strings.TrimSpace(c.GetString("role_name")))
}

func currentStaffID(c *gin.Context) string {
	if !strings.EqualFold(strings.TrimSpace(c.GetString("linked_type")), "staff") {
		return ""
	}
	if linkedID := strings.TrimSpace(c.GetString("linked_id")); linkedID != "" {
		return linkedID
	}

	var user models.User
	if err := database.DB.First(&user, "id = ? AND school_id = ?", c.GetString("user_id"), scopedSchoolID(c)).Error; err == nil {
		if user.LinkedID != nil && strings.TrimSpace(*user.LinkedID) != "" {
			return strings.TrimSpace(*user.LinkedID)
		}
	}

	var staff models.Staff
	if err := database.DB.First(&staff, "school_id = ? AND email = ?", scopedSchoolID(c), c.GetString("email")).Error; err != nil {
		return ""
	}
	return staff.ID
}

func teacherAssignedSectionSubquery(c *gin.Context) *gorm.DB {
	staffID := currentStaffID(c)
	return database.DB.Model(&models.Section{}).
		Select("sections.id").
		Where("sections.class_teacher_id = ?", staffID).
		Or("sections.id IN (?)", database.DB.Model(&models.TimetableSlot{}).Select("section_id").Where("staff_id = ?", staffID))
}

func applyStudentVisibility(c *gin.Context, query *gorm.DB) *gorm.DB {
	schoolID := scopedSchoolID(c)
	switch currentRole(c) {
	case "parent":
		return query.Joins("JOIN parent_student_links ON parent_student_links.student_id = students.id").
			Where("parent_student_links.school_id = ? AND parent_student_links.parent_user_id = ?", schoolID, c.GetString("user_id"))
	case "teacher":
		return query.Where("students.current_section_id IN (?)", teacherAssignedSectionSubquery(c))
	default:
		return query
	}
}

func canAccessStudentID(c *gin.Context, studentID string) bool {
	if strings.TrimSpace(studentID) == "" {
		return false
	}
	schoolID := scopedSchoolID(c)
	var count int64
	query := database.DB.Model(&models.Student{}).Where("students.id = ? AND students.school_id = ?", studentID, schoolID)
	query = applyStudentVisibility(c, query)
	query.Count(&count)
	return count > 0
}

func studentEnrollmentInSection(studentID, enrollmentID, sectionID string) bool {
	if studentID == "" || enrollmentID == "" || sectionID == "" {
		return false
	}
	var count int64
	database.DB.Model(&models.Enrollment{}).
		Where("id = ? AND student_id = ? AND section_id = ? AND status != ?", enrollmentID, studentID, sectionID, "inactive").
		Count(&count)
	return count > 0
}

func teacherCanAccessSection(c *gin.Context, sectionID string) bool {
	if currentRole(c) != "teacher" {
		return true
	}
	staffID := currentStaffID(c)
	if staffID == "" || sectionID == "" {
		return false
	}
	var count int64
	database.DB.Model(&models.Section{}).
		Where("sections.id = ? AND sections.class_teacher_id = ?", sectionID, staffID).
		Or("sections.id = ? AND EXISTS (SELECT 1 FROM timetable_slots WHERE timetable_slots.section_id = sections.id AND timetable_slots.staff_id = ?)", sectionID, staffID).
		Count(&count)
	return count > 0
}

func teacherCanAccessSectionSubject(c *gin.Context, sectionID, subjectID string) bool {
	if currentRole(c) != "teacher" {
		return true
	}
	staffID := currentStaffID(c)
	if staffID == "" || sectionID == "" {
		return false
	}
	var count int64
	query := database.DB.Model(&models.Section{}).
		Where("sections.id = ? AND sections.class_teacher_id = ?", sectionID, staffID)
	if subjectID != "" {
		query = query.Or("sections.id = ? AND EXISTS (SELECT 1 FROM timetable_slots WHERE timetable_slots.section_id = sections.id AND timetable_slots.staff_id = ? AND timetable_slots.subject_id = ?)", sectionID, staffID, subjectID)
	} else {
		query = query.Or("sections.id = ? AND EXISTS (SELECT 1 FROM timetable_slots WHERE timetable_slots.section_id = sections.id AND timetable_slots.staff_id = ?)", sectionID, staffID)
	}
	query.Count(&count)
	return count > 0
}

func linkedStudentSubquery(c *gin.Context) *gorm.DB {
	return database.DB.Model(&models.ParentStudentLink{}).
		Select("student_id").
		Where("school_id = ? AND parent_user_id = ?", scopedSchoolID(c), c.GetString("user_id"))
}

func forbid(c *gin.Context, message string) {
	fail(c, http.StatusForbidden, message)
}
