package handlers

import (
	"net/http"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type SchoolHandler struct{}

func NewSchoolHandler() *SchoolHandler {
	return &SchoolHandler{}
}

func (h *SchoolHandler) GetSchools(c *gin.Context) {
	var schools []models.School
	database.DB.Find(&schools)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: schools})
}

func (h *SchoolHandler) GetSchool(c *gin.Context) {
	id := c.Param("id")
	var school models.School
	if err := database.DB.First(&school, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "School not found"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: school})
}

func (h *SchoolHandler) CreateSchool(c *gin.Context) {
	var req models.CreateSchoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	school := models.School{
		Name:             req.Name,
		SchoolType:       req.SchoolType,
		AffiliationBoard: req.AffiliationBoard,
		Email:            req.Email,
		Phone:            req.Phone,
		City:             req.City,
		State:            req.State,
		Timezone:         req.Timezone,
		Currency:         req.Currency,
	}

	if err := database.DB.Create(&school).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create school"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: school})
}

func (h *SchoolHandler) GetAcademicYears(c *gin.Context) {
	schoolID := c.Query("school_id")
	var years []models.AcademicYear
	query := database.DB.Preload("Terms")
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	query.Find(&years)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: years})
}

func (h *SchoolHandler) GetAcademicYear(c *gin.Context) {
	id := c.Param("id")
	var year models.AcademicYear
	if err := database.DB.Preload("Terms").Preload("Holidays").First(&year, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Academic year not found"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: year})
}

func (h *SchoolHandler) CreateAcademicYear(c *gin.Context) {
	var req models.CreateAcademicYearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)

	year := models.AcademicYear{
		SchoolID:  req.SchoolID,
		YearLabel: req.YearLabel,
		StartDate: startDate,
		EndDate:   endDate,
		IsCurrent: req.IsCurrent,
		Status:    "active",
	}

	if err := database.DB.Create(&year).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create academic year"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: year})
}

func (h *SchoolHandler) GetGrades(c *gin.Context) {
	schoolID := c.Query("school_id")
	var grades []models.Grade
	query := database.DB.Preload("Sections")
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	query.Find(&grades)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: grades})
}

func (h *SchoolHandler) GetGrade(c *gin.Context) {
	id := c.Param("id")
	var grade models.Grade
	if err := database.DB.Preload("Sections").Preload("GradeSubjects").First(&grade, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Grade not found"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: grade})
}

func (h *SchoolHandler) CreateGrade(c *gin.Context) {
	var req models.CreateGradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	grade := models.Grade{
		SchoolID:    req.SchoolID,
		GradeNumber: req.GradeNumber,
		GradeName:   req.GradeName,
	}

	if err := database.DB.Create(&grade).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create grade"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: grade})
}

func (h *SchoolHandler) GetSections(c *gin.Context) {
	gradeID := c.Query("grade_id")
	yearID := c.Query("academic_year_id")
	var sections []models.Section
	query := database.DB.Preload("Grade").Preload("ClassTeacher").Preload("Room")
	if gradeID != "" {
		query = query.Where("grade_id = ?", gradeID)
	}
	if yearID != "" {
		query = query.Where("academic_year_id = ?", yearID)
	}
	query.Find(&sections)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: sections})
}

func (h *SchoolHandler) GetSection(c *gin.Context) {
	id := c.Param("id")
	var section models.Section
	if err := database.DB.Preload("Grade").Preload("ClassTeacher").Preload("Room").Preload("AcademicYear").First(&section, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Section not found"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: section})
}

func (h *SchoolHandler) CreateSection(c *gin.Context) {
	var req models.CreateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	section := models.Section{
		GradeID:        req.GradeID,
		AcademicYearID: req.AcademicYearID,
		SectionName:    req.SectionName,
		Capacity:       req.Capacity,
	}

	if req.ClassTeacherID != "" {
		section.ClassTeacherID = &req.ClassTeacherID
	}
	if req.RoomID != "" {
		section.RoomID = &req.RoomID
	}

	if err := database.DB.Create(&section).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create section"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: section})
}

func (h *SchoolHandler) GetDepartments(c *gin.Context) {
	schoolID := c.Query("school_id")
	var depts []models.Department
	query := database.DB.Preload("HODStaff")
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	query.Find(&depts)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: depts})
}

func (h *SchoolHandler) CreateDepartment(c *gin.Context) {
	var req struct {
		SchoolID       string `json:"school_id" binding:"required"`
		DepartmentName string `json:"department_name" binding:"required"`
		Description    string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dept := models.Department{
		SchoolID:       req.SchoolID,
		DepartmentName: req.DepartmentName,
		Description:    req.Description,
	}

	if err := database.DB.Create(&dept).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create department"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: dept})
}

func (h *SchoolHandler) GetSubjects(c *gin.Context) {
	schoolID := c.Query("school_id")
	deptID := c.Query("department_id")
	var subjects []models.Subject
	query := database.DB.Preload("Department")
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	if deptID != "" {
		query = query.Where("department_id = ?", deptID)
	}
	query.Find(&subjects)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: subjects})
}

func (h *SchoolHandler) CreateSubject(c *gin.Context) {
	var req models.CreateSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subject := models.Subject{
		SchoolID:     req.SchoolID,
		DepartmentID: req.DepartmentID,
		SubjectName:  req.SubjectName,
		SubjectCode:  req.SubjectCode,
		SubjectType:  req.SubjectType,
		CreditHours:  req.CreditHours,
	}

	if err := database.DB.Create(&subject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subject"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: subject})
}

func (h *SchoolHandler) GetRooms(c *gin.Context) {
	schoolID := c.Query("school_id")
	var rooms []models.Room
	query := database.DB.Preload("School")
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	query.Find(&rooms)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: rooms})
}

func (h *SchoolHandler) CreateRoom(c *gin.Context) {
	var req struct {
		SchoolID   string `json:"school_id" binding:"required"`
		RoomNumber string `json:"room_number" binding:"required"`
		RoomType   string `json:"room_type" binding:"required"`
		Block      string `json:"block"`
		Floor      int    `json:"floor"`
		Capacity   int    `json:"capacity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	room := models.Room{
		SchoolID:   req.SchoolID,
		RoomNumber: req.RoomNumber,
		RoomType:   req.RoomType,
		Block:      req.Block,
		Floor:      req.Floor,
		Capacity:   req.Capacity,
	}

	if err := database.DB.Create(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create room"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: room})
}

func (h *SchoolHandler) GetTerms(c *gin.Context) {
	yearID := c.Param("year_id")
	var terms []models.Term
	database.DB.Where("academic_year_id = ?", yearID).Find(&terms)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: terms})
}

func (h *SchoolHandler) PaginationMeta(page, pageSize int, total int64) map[string]int {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return map[string]int{
		"page":        page,
		"page_size":   pageSize,
		"total":       int(total),
		"total_pages": totalPages,
	}
}
