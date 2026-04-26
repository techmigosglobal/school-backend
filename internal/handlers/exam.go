package handlers

import (
	"net/http"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type ExamHandler struct{}

func NewExamHandler() *ExamHandler {
	return &ExamHandler{}
}

func (h *ExamHandler) GetExamTypes(c *gin.Context) {
	schoolID := c.Query("school_id")
	var examTypes []models.ExamType
	query := database.DB.Preload("School")
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	query.Find(&examTypes)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: examTypes})
}

func (h *ExamHandler) CreateExamType(c *gin.Context) {
	var req struct {
		SchoolID         string  `json:"school_id" binding:"required"`
		Name             string  `json:"name" binding:"required"`
		WeightagePercent float64 `json:"weightage_percent"`
		IsBoardExam      bool    `json:"is_board_exam"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	examType := models.ExamType{
		SchoolID:         req.SchoolID,
		Name:             req.Name,
		WeightagePercent: req.WeightagePercent,
		IsBoardExam:      req.IsBoardExam,
	}

	if err := database.DB.Create(&examType).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create exam type"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: examType})
}

func (h *ExamHandler) GetExams(c *gin.Context) {
	schoolID := c.Query("school_id")
	yearID := c.Query("academic_year_id")
	termID := c.Query("term_id")

	var exams []models.Exam
	query := database.DB.Preload("ExamType").Preload("AcademicYear").Preload("Term")
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	if yearID != "" {
		query = query.Where("academic_year_id = ?", yearID)
	}
	if termID != "" {
		query = query.Where("term_id = ?", termID)
	}
	query.Find(&exams)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: exams})
}

func (h *ExamHandler) GetExam(c *gin.Context) {
	id := c.Param("id")
	var exam models.Exam
	if err := database.DB.Preload("ExamType").Preload("AcademicYear").Preload("Term").Preload("Schedules").Preload("Schedules.Subject").First(&exam, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Exam not found"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: exam})
}

func (h *ExamHandler) CreateExam(c *gin.Context) {
	var req models.CreateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)

	exam := models.Exam{
		SchoolID:       req.SchoolID,
		AcademicYearID: req.AcademicYearID,
		TermID:         req.TermID,
		ExamTypeID:     req.ExamTypeID,
		ExamName:       req.ExamName,
		StartDate:      startDate,
		EndDate:        endDate,
		IsPublished:    false,
	}

	if err := database.DB.Create(&exam).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create exam"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: exam})
}

func (h *ExamHandler) CreateExamSchedule(c *gin.Context) {
	var req struct {
		ExamID     string `json:"exam_id" binding:"required"`
		GradeID    string `json:"grade_id" binding:"required"`
		SectionID  string `json:"section_id" binding:"required"`
		SubjectID  string `json:"subject_id" binding:"required"`
		ExamDate   string `json:"exam_date" binding:"required"`
		StartTime  string `json:"start_time"`
		EndTime    string `json:"end_time"`
		MaxMarks   int    `json:"max_marks" binding:"required"`
		PassMarks  int    `json:"pass_marks" binding:"required"`
		RoomID     string `json:"room_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	examDate, _ := time.Parse("2006-01-02", req.ExamDate)

	schedule := models.ExamSchedule{
		ExamID:     req.ExamID,
		GradeID:    req.GradeID,
		SectionID:  req.SectionID,
		SubjectID:  req.SubjectID,
		ExamDate:   examDate,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		MaxMarks:   req.MaxMarks,
		PassMarks:  req.PassMarks,
	}

	if req.RoomID != "" {
		schedule.RoomID = &req.RoomID
	}

	if err := database.DB.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create exam schedule"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: schedule})
}

func (h *ExamHandler) EnterMarks(c *gin.Context) {
	scheduleID := c.Param("schedule_id")
	var req struct {
		Marks []struct {
			StudentID    string  `json:"student_id" binding:"required"`
			EnrollmentID string  `json:"enrollment_id" binding:"required"`
			MarksObtained float64 `json:"marks_obtained"`
			GradeLabel   string  `json:"grade_label"`
			IsAbsent     bool    `json:"is_absent"`
			IsExempted   bool    `json:"is_exempted"`
		} `json:"marks" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, m := range req.Marks {
		mark := models.StudentMark{
			ExamScheduleID: scheduleID,
			StudentID:      m.StudentID,
			EnrollmentID:   m.EnrollmentID,
			MarksObtained:  m.MarksObtained,
			GradeLabel:     m.GradeLabel,
			IsAbsent:       m.IsAbsent,
			IsExempted:     m.IsExempted,
		}
		database.DB.Create(&mark)
	}

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Marks entered successfully"})
}

func (h *ExamHandler) GetReportCards(c *gin.Context) {
	studentID := c.Query("student_id")
	examID := c.Query("exam_id")

	var reportCards []models.ReportCard
	query := database.DB.Preload("Student").Preload("Exam")
	if studentID != "" {
		query = query.Where("student_id = ?", studentID)
	}
	if examID != "" {
		query = query.Where("exam_id = ?", examID)
	}
	query.Find(&reportCards)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: reportCards})
}

func (h *ExamHandler) GetGradingScale(c *gin.Context) {
	schoolID := c.Query("school_id")
	var scales []models.GradingScale
	query := database.DB.Preload("School")
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	query.Find(&scales)

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: scales})
}