package handlers

import (
	"net/http"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type StudentHandler struct{}

func NewStudentHandler() *StudentHandler {
	return &StudentHandler{}
}

func (h *StudentHandler) GetStudents(c *gin.Context) {
	page, pageSize := parsePagination(c)
	schoolID := scopedSchoolID(c)
	sectionID := c.Query("section_id")
	status := c.Query("status")

	var students []models.Student
	var total int64

	query := database.DB.Model(&models.Student{})
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	query = applyStudentVisibility(c, query)
	if currentRole(c) == "teacher" && sectionID != "" && !teacherCanAccessSection(c, sectionID) {
		forbid(c, "section access denied")
		return
	}
	if sectionID != "" {
		query = query.Where("current_section_id = ?", sectionID)
	}
	if status != "" {
		// Caller explicitly requested a specific status (e.g. ?status=inactive
		// for admin views). Pass it through as-is.
		query = query.Where("status = ?", status)
	} else {
		// Default: exclude soft-deleted (inactive) students from all listings.
		// Use ?status=inactive to explicitly query deactivated records.
		query = query.Where("status != ?", "inactive")
	}

	query.Count(&total)
	query = query.Preload("Guardians").Preload("MedicalRecord").Preload("CurrentSection").Offset((page - 1) * pageSize).Limit(pageSize)
	query.Find(&students)

	c.JSON(http.StatusOK, paginationResult(page, pageSize, total, students))
}

func (h *StudentHandler) GetStudent(c *gin.Context) {
	id := c.Param("id")
	if !canAccessStudentID(c, id) {
		fail(c, http.StatusForbidden, "student access denied")
		return
	}
	var student models.Student
	if err := database.DB.Preload("Guardians").Preload("MedicalRecord").Preload("CurrentSection").Preload("CurrentSection.Grade").Preload("Documents").First(&student, "id = ? AND school_id = ?", id, scopedSchoolID(c)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: student})
}

func (h *StudentHandler) CreateStudent(c *gin.Context) {
	var req models.CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dob, _ := time.Parse("2006-01-02", req.DateOfBirth)
	admDate := time.Now()
	if req.AdmissionDate != "" {
		admDate, _ = time.Parse("2006-01-02", req.AdmissionDate)
	}
	if req.CurrentSectionID != "" {
		var sectionCount int64
		database.DB.Model(&models.Section{}).
			Joins("JOIN grades ON grades.id = sections.grade_id").
			Where("sections.id = ? AND grades.school_id = ?", req.CurrentSectionID, scopedSchoolID(c)).
			Count(&sectionCount)
		if sectionCount == 0 {
			fail(c, http.StatusBadRequest, "section does not belong to school")
			return
		}
	}

	student := models.Student{
		SchoolID:         scopedSchoolID(c),
		StudentCode:      req.StudentCode,
		AdmissionNumber:  req.AdmissionNumber,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		DateOfBirth:      dob,
		Gender:           req.Gender,
		AdmissionDate:    admDate,
		CurrentSectionID: &req.CurrentSectionID,
		Status:           "active",
	}

	if err := database.DB.Create(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create student"})
		return
	}

	id := student.ID
	auditAction(c, "students", "create", "students", &id)
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: student})
}

func (h *StudentHandler) UpdateStudent(c *gin.Context) {
	id := c.Param("id")
	var student models.Student
	if err := database.DB.First(&student, "id = ? AND school_id = ?", id, scopedSchoolID(c)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}

	var req models.CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	student.FirstName = req.FirstName
	student.LastName = req.LastName
	student.Gender = req.Gender
	if req.CurrentSectionID != "" {
		student.CurrentSectionID = &req.CurrentSectionID
	}

	if err := database.DB.Save(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update student"})
		return
	}

	auditAction(c, "students", "update", "students", &id)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: student})
}

func (h *StudentHandler) DeleteStudent(c *gin.Context) {
	id := c.Param("id")

	// Soft-delete: mark the student inactive rather than removing the row.
	// This preserves attendance, fee invoice, exam, and audit records that
	// reference this student's ID.
	result := database.DB.Model(&models.Student{}).
		Where("id = ? AND school_id = ?", id, scopedSchoolID(c)).
		Update("status", "inactive")

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to deactivate student",
		})
		return
	}
	if result.RowsAffected == 0 {
		// Either the student does not exist or belongs to another school.
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Student not found",
		})
		return
	}

	auditAction(c, "students", "delete", "students", &id)
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Student deactivated successfully",
	})
}

func (h *StudentHandler) GetStudentEnrollments(c *gin.Context) {
	studentID := c.Param("id")
	if !canAccessStudentID(c, studentID) {
		fail(c, http.StatusForbidden, "student access denied")
		return
	}
	var enrollments []models.Enrollment
	database.DB.Preload("Section").Preload("Section.Grade").Preload("AcademicYear").Where("student_id = ?", studentID).Find(&enrollments)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: enrollments})
}

func (h *StudentHandler) CreateEnrollment(c *gin.Context) {
	var req models.CreateEnrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !canAccessStudentID(c, req.StudentID) {
		forbid(c, "student access denied")
		return
	}
	var sectionCount int64
	database.DB.Model(&models.Section{}).
		Joins("JOIN grades ON grades.id = sections.grade_id").
		Where("sections.id = ? AND grades.school_id = ?", req.SectionID, scopedSchoolID(c)).
		Count(&sectionCount)
	if sectionCount == 0 {
		fail(c, http.StatusBadRequest, "section does not belong to school")
		return
	}

	enrollDate := time.Now()
	if req.EnrollmentDate != "" {
		enrollDate, _ = time.Parse("2006-01-02", req.EnrollmentDate)
	}

	enrollment := models.Enrollment{
		StudentID:      req.StudentID,
		SectionID:      req.SectionID,
		AcademicYearID: req.AcademicYearID,
		RollNumber:     req.RollNumber,
		EnrollmentDate: enrollDate,
		Status:         "enrolled",
	}

	if err := database.DB.Create(&enrollment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create enrollment"})
		return
	}

	id := enrollment.ID
	auditAction(c, "enrollments", "create", "enrollments", &id)
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: enrollment})
}

func (h *StudentHandler) GetStudentAttendance(c *gin.Context) {
	studentID := c.Param("id")
	if !canAccessStudentID(c, studentID) {
		fail(c, http.StatusForbidden, "student access denied")
		return
	}
	month := c.Query("month")
	year := c.Query("year")

	var attendance []models.StudentAttendance
	query := database.DB.Where("student_id = ?", studentID).Preload("Session")
	if month != "" {
		start, end, ok := monthYearRange(month, year)
		if ok {
			query = query.Where("marked_at >= ? AND marked_at < ?", start, end)
		}
	} else if year != "" {
		start, _, ok := monthYearRange("01", year)
		if ok {
			query = query.Where("marked_at >= ? AND marked_at < ?", start, start.AddDate(1, 0, 0))
		}
	}
	query.Find(&attendance)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: attendance})
}

func (h *StudentHandler) GetStudentFees(c *gin.Context) {
	studentID := c.Param("id")
	if !canAccessStudentID(c, studentID) {
		fail(c, http.StatusForbidden, "student access denied")
		return
	}
	var invoices []models.FeeInvoice
	database.DB.Where("student_id = ?", studentID).Preload("Items").Preload("Payments").Find(&invoices)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: invoices})
}

func (h *StudentHandler) GetStudentMarks(c *gin.Context) {
	studentID := c.Param("id")
	if !canAccessStudentID(c, studentID) {
		fail(c, http.StatusForbidden, "student access denied")
		return
	}
	examID := c.Query("exam_id")

	var marks []models.StudentMark
	query := database.DB.Where("student_id = ?", studentID).Preload("ExamSchedule").Preload("ExamSchedule.Subject")
	if examID != "" {
		query = query.Joins("JOIN exam_schedules ON exam_schedules.id = student_marks.exam_schedule_id").
			Where("exam_schedules.exam_id = ?", examID).
			Preload("ExamSchedule.Exam")
	}
	query.Find(&marks)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: marks})
}

func (h *StudentHandler) GetStudentTransport(c *gin.Context) {
	studentID := c.Param("id")
	if !canAccessStudentID(c, studentID) {
		fail(c, http.StatusForbidden, "student access denied")
		return
	}
	var transports []models.StudentTransport
	database.DB.Where("student_id = ?", studentID).Preload("Route").Preload("Stop").Find(&transports)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: transports})
}

func isParent(c *gin.Context) bool {
	return currentRole(c) == "parent"
}

func (h *StudentHandler) teacherStaffID(c *gin.Context) string {
	return currentStaffID(c)
}
