package main

import (
	"log"
	"os"

	"school-backend/internal/config"
	"school-backend/internal/database"
	"school-backend/internal/handlers"
	"school-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	middleware.SetJWTSecret(cfg.JWTSecret)

	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(middleware.CORSMiddleware())

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":    "School Desk Backend API",
			"version":    "1.0.0",
			"status":     "running",
			"endpoints":  "/api/v1/*path",
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	authHandler := handlers.NewAuthHandler()
	schoolHandler := handlers.NewSchoolHandler()
	staffHandler := handlers.NewStaffHandler()
	studentHandler := handlers.NewStudentHandler()
	attendanceHandler := handlers.NewAttendanceHandler()
	examHandler := handlers.NewExamHandler()
	feeHandler := handlers.NewFeeHandler()
	leaveHandler := handlers.NewLeaveHandler()
	timetableHandler := handlers.NewTimetableHandler()
	announcementHandler := handlers.NewAnnouncementHandler()

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.GET("/profile", middleware.AuthMiddleware(), authHandler.GetProfile)
		}

		schools := api.Group("/schools")
		{
			schools.GET("", schoolHandler.GetSchools)
			schools.GET("/:id", schoolHandler.GetSchool)
			schools.POST("", schoolHandler.CreateSchool)
		}

		academicYears := api.Group("/academic-years")
		{
			academicYears.GET("", schoolHandler.GetAcademicYears)
			academicYears.GET("/:id", schoolHandler.GetAcademicYear)
			academicYears.POST("", schoolHandler.CreateAcademicYear)
			academicYears.GET("/:year_id/terms", schoolHandler.GetTerms)
		}

		grades := api.Group("/grades")
		{
			grades.GET("", schoolHandler.GetGrades)
			grades.GET("/:id", schoolHandler.GetGrade)
			grades.POST("", schoolHandler.CreateGrade)
		}

		sections := api.Group("/sections")
		{
			sections.GET("", schoolHandler.GetSections)
			sections.GET("/:id", schoolHandler.GetSection)
			sections.POST("", schoolHandler.CreateSection)
		}

		departments := api.Group("/departments")
		{
			departments.GET("", schoolHandler.GetDepartments)
			departments.POST("", schoolHandler.CreateDepartment)
		}

		subjects := api.Group("/subjects")
		{
			subjects.GET("", schoolHandler.GetSubjects)
			subjects.POST("", schoolHandler.CreateSubject)
		}

		rooms := api.Group("/rooms")
		{
			rooms.GET("", schoolHandler.GetRooms)
			rooms.POST("", schoolHandler.CreateRoom)
		}

		staff := api.Group("/staff")
		staff.Use(middleware.AuthMiddleware())
		{
			staff.GET("", staffHandler.GetStaff)
			staff.GET("/:id", staffHandler.GetStaffMember)
			staff.POST("", staffHandler.CreateStaff)
			staff.PUT("/:id", staffHandler.UpdateStaff)
			staff.DELETE("/:id", staffHandler.DeleteStaff)
			staff.GET("/:id/leave-balances", staffHandler.GetStaffLeaveBalance)
			staff.GET("/:id/attendance", staffHandler.GetStaffAttendance)
		}

		students := api.Group("/students")
		students.Use(middleware.AuthMiddleware())
		{
			students.GET("", studentHandler.GetStudents)
			students.GET("/:id", studentHandler.GetStudent)
			students.POST("", studentHandler.CreateStudent)
			students.PUT("/:id", studentHandler.UpdateStudent)
			students.DELETE("/:id", studentHandler.DeleteStudent)
			students.GET("/:id/enrollments", studentHandler.GetStudentEnrollments)
			students.POST("/enrollments", studentHandler.CreateEnrollment)
			students.GET("/:id/attendance", studentHandler.GetStudentAttendance)
			students.GET("/:id/fees", studentHandler.GetStudentFees)
			students.GET("/:id/marks", studentHandler.GetStudentMarks)
			students.GET("/:id/transport", studentHandler.GetStudentTransport)
		}

		attendance := api.Group("/attendance")
		attendance.Use(middleware.AuthMiddleware())
		{
			attendance.GET("/sessions", attendanceHandler.GetAttendanceSessions)
			attendance.POST("/sessions", attendanceHandler.CreateAttendanceSession)
			attendance.POST("/sessions/:session_id/mark", attendanceHandler.MarkStudentAttendance)
			attendance.GET("/summary", attendanceHandler.GetStudentAttendanceSummary)
			attendance.POST("/staff", attendanceHandler.MarkStaffAttendance)
		}

		exams := api.Group("/exams")
		exams.Use(middleware.AuthMiddleware())
		{
			exams.GET("/types", examHandler.GetExamTypes)
			exams.POST("/types", examHandler.CreateExamType)
			exams.GET("", examHandler.GetExams)
			exams.GET("/:id", examHandler.GetExam)
			exams.POST("", examHandler.CreateExam)
			exams.POST("/schedules", examHandler.CreateExamSchedule)
			exams.POST("/schedules/:schedule_id/marks", examHandler.EnterMarks)
			exams.GET("/report-cards", examHandler.GetReportCards)
			exams.GET("/grading-scale", examHandler.GetGradingScale)
		}

		fees := api.Group("/fees")
		fees.Use(middleware.AuthMiddleware())
		{
			fees.GET("/categories", feeHandler.GetFeeCategories)
			fees.POST("/categories", feeHandler.CreateFeeCategory)
			fees.GET("/structures", feeHandler.GetFeeStructures)
			fees.POST("/structures", feeHandler.CreateFeeStructure)
			fees.GET("/invoices", feeHandler.GetInvoices)
			fees.POST("/invoices", feeHandler.CreateInvoice)
			fees.POST("/payments", feeHandler.RecordPayment)
			fees.GET("/concessions", feeHandler.GetConcessions)
		}

		leave := api.Group("/leave")
		leave.Use(middleware.AuthMiddleware())
		{
			leave.GET("/types", leaveHandler.GetLeaveTypes)
			leave.POST("/types", leaveHandler.CreateLeaveType)
			leave.GET("/applications", leaveHandler.GetLeaveApplications)
			leave.POST("/applications", leaveHandler.CreateLeaveApplication)
			leave.PUT("/applications/:id/approve", leaveHandler.ApproveLeaveApplication)
			leave.GET("/balances", leaveHandler.GetLeaveBalances)
			leave.POST("/balances/initialize", leaveHandler.InitializeLeaveBalances)
		}

		timetable := api.Group("/timetable")
		timetable.Use(middleware.AuthMiddleware())
		{
			timetable.GET("/slots", timetableHandler.GetTimetableSlots)
			timetable.POST("/slots", timetableHandler.CreateTimetableSlot)
			timetable.PUT("/slots/:id", timetableHandler.UpdateTimetableSlot)
			timetable.DELETE("/slots/:id", timetableHandler.DeleteTimetableSlot)
			timetable.GET("/substitutions", timetableHandler.GetSubstitutions)
			timetable.POST("/substitutions", timetableHandler.CreateSubstitution)
			timetable.GET("/section/:section_id", timetableHandler.GetTimetableBySection)
		}

		announcements := api.Group("/announcements")
		announcements.Use(middleware.AuthMiddleware())
		{
			announcements.GET("", announcementHandler.GetAnnouncements)
			announcements.POST("", announcementHandler.CreateAnnouncement)
		}

		events := api.Group("/events")
		events.Use(middleware.AuthMiddleware())
		{
			events.GET("", announcementHandler.GetEvents)
			events.POST("", announcementHandler.CreateEvent)
		}

		notifications := api.Group("/notifications")
		notifications.Use(middleware.AuthMiddleware())
		{
			notifications.GET("", announcementHandler.GetNotifications)
			notifications.PUT("/:id/read", announcementHandler.MarkNotificationRead)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Port
	}

	log.Printf("School Desk Backend starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
