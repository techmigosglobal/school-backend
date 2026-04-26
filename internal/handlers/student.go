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
	schoolID := c.Query("school_id")
	sectionID := c.Query("section_id")
	status := c.Query("status")

	var students []models.Student
	var total int64

	query := database.DB.Model(&models.Student{})
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	if sectionID != "" {
		query = query.Where("current_section_id = ?", sectionID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	query = query.Preload("Guardians").Preload("MedicalRecord").Preload("CurrentSection").Offset((page-1)*pageSize).Limit(pageSize)
	query.Find(&students)

	c.JSON(http.StatusOK, models.PaginatedResponse{
		Success:    true,
		Data:       students,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: int(total) / pageSize,
	})
}

func (h *StudentHandler) GetStudent(c *gin.Context) {
	id := c.Param("id")
	var student models.Student
	if err := database.DB.Preload("Guardians").Preload("MedicalRecord").Preload("CurrentSection").Preload("CurrentSection.Grade").Preload("Documents").First(&student, "id = ?", id).Error; err != nil {
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

	student := models.Student{
		SchoolID:         req.SchoolID,
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

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: student})
}

func (h *StudentHandler) UpdateStudent(c *gin.Context) {
	id := c.Param("id")
	var student models.Student
	if err := database.DB.First(&student, "id = ?", id).Error; err != nil {
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

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: student})
}

func (h *StudentHandler) DeleteStudent(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Student{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete student"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Student deleted successfully"})
}

func (h *StudentHandler) GetStudentEnrollments(c *gin.Context) {
	studentID := c.Param("id")
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

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: enrollment})
}

func (h *StudentHandler) GetStudentAttendance(c *gin.Context) {
	studentID := c.Param("id")
	month := c.Query("month")
	year := c.Query("year")

	var attendance []models.StudentAttendance
	query := database.DB.Where("student_id = ?", studentID).Preload("Session")
	if month != "" {
		query = query.Where("strftime('%m', marked_at) = ?", month)
	}
	if year != "" {
		query = query.Where("strftime('%Y', marked_at) = ?", year)
	}
	query.Find(&attendance)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: attendance})
}

func (h *StudentHandler) GetStudentFees(c *gin.Context) {
	studentID := c.Param("id")
	var invoices []models.FeeInvoice
	database.DB.Where("student_id = ?", studentID).Preload("Items").Preload("Payments").Find(&invoices)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: invoices})
}

func (h *StudentHandler) GetStudentMarks(c *gin.Context) {
	studentID := c.Param("id")
	examID := c.Query("exam_id")

	var marks []models.StudentMark
	query := database.DB.Where("student_id = ?", studentID).Preload("ExamSchedule").Preload("ExamSchedule.Subject")
	if examID != "" {
		query = query.Preload("ExamSchedule.Exam", "exam_id = ?", examID)
	}
	query.Find(&marks)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: marks})
}

func (h *StudentHandler) GetStudentTransport(c *gin.Context) {
	studentID := c.Param("id")
	var transports []models.StudentTransport
	database.DB.Where("student_id = ?", studentID).Preload("Route").Preload("Stop").Find(&transports)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: transports})
}