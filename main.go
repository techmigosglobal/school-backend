package main

import (
	"log"
	"os"
	"time"

	"school-backend/internal/config"
	"school-backend/internal/database"
	"school-backend/internal/handlers"
	"school-backend/internal/middleware"
	"school-backend/internal/platform"
	"school-backend/internal/services"
	"school-backend/internal/worker"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}
	middleware.SetJWTSecret(cfg.JWTSecret)
	middleware.SetAllowedOrigins(cfg.AllowedOrigins)

	redisClient, err := platform.NewRedisClient(cfg)
	if err != nil {
		if cfg.Environment == "production" {
			log.Fatalf("Failed to initialize redis: %v", err)
		}
		log.Printf("Redis unavailable, running without redis-backed features: %v", err)
	} else {
		services.Cache = services.NewCacheService(redisClient, cfg.Environment)
		services.Rate = services.NewRateLimitService(redisClient, cfg.Environment)
		services.Sessions = services.NewSessionStore(redisClient, cfg.Environment)
		services.Queue = services.NewJobQueue(redisClient, cfg.Environment)
	}

	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if cfg.AppMode == "worker" {
		if err := worker.RunNotificationWorker(); err != nil {
			log.Fatalf("Worker failed: %v", err)
		}
		return
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(middleware.RequestIDMiddleware(), middleware.CORSMiddleware())

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":   "School Desk Backend API",
			"version":   "1.0.0",
			"status":    "running",
			"endpoints": "/api/v1/*path",
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
			auth.POST("/login", middleware.RateLimitMiddleware("auth_login", cfg.RateLimitMaxLogin, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", middleware.AuthMiddleware(), authHandler.Logout)
			if !cfg.DisablePublicRegistration {
				auth.POST("/register", middleware.RateLimitMiddleware("auth_register", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), authHandler.Register)
			}
			auth.GET("/profile", middleware.AuthMiddleware(), authHandler.GetProfile)
		}

		schools := api.Group("/schools")
		schools.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			schools.GET("", middleware.CacheMiddleware("schools_list", time.Duration(cfg.CacheTTLSeconds)*time.Second), schoolHandler.GetSchools)
			schools.GET("/:id", middleware.CacheMiddleware("schools_detail", time.Duration(cfg.CacheTTLSeconds)*time.Second), schoolHandler.GetSchool)
			schools.POST("", middleware.RBACMiddleware("Admin", "Principal"), schoolHandler.CreateSchool)
		}

		academicYears := api.Group("/academic-years")
		academicYears.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			academicYears.GET("", middleware.CacheMiddleware("academic_years_list", time.Duration(cfg.CacheTTLSeconds)*time.Second), schoolHandler.GetAcademicYears)
			academicYears.GET("/:id", middleware.CacheMiddleware("academic_years_detail", time.Duration(cfg.CacheTTLSeconds)*time.Second), schoolHandler.GetAcademicYear)
			academicYears.POST("", middleware.RBACMiddleware("Admin", "Principal"), schoolHandler.CreateAcademicYear)
			academicYears.GET("/:id/terms", schoolHandler.GetTerms)
		}

		grades := api.Group("/grades")
		grades.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			grades.GET("", schoolHandler.GetGrades)
			grades.GET("/:id", schoolHandler.GetGrade)
			grades.POST("", middleware.RBACMiddleware("Admin", "Principal"), schoolHandler.CreateGrade)
		}

		sections := api.Group("/sections")
		sections.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			sections.GET("", schoolHandler.GetSections)
			sections.GET("/:id", schoolHandler.GetSection)
			sections.POST("", middleware.RBACMiddleware("Admin", "Principal"), schoolHandler.CreateSection)
		}

		departments := api.Group("/departments")
		departments.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			departments.GET("", schoolHandler.GetDepartments)
			departments.POST("", middleware.RBACMiddleware("Admin", "Principal"), schoolHandler.CreateDepartment)
		}

		subjects := api.Group("/subjects")
		subjects.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			subjects.GET("", schoolHandler.GetSubjects)
			subjects.POST("", middleware.RBACMiddleware("Admin", "Principal"), schoolHandler.CreateSubject)
		}

		rooms := api.Group("/rooms")
		rooms.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			rooms.GET("", schoolHandler.GetRooms)
			rooms.POST("", middleware.RBACMiddleware("Admin", "Principal"), schoolHandler.CreateRoom)
		}

		staff := api.Group("/staff")
		staff.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			staff.GET("", staffHandler.GetStaff)
			staff.GET("/:id", staffHandler.GetStaffMember)
			staff.POST("", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("staff_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), staffHandler.CreateStaff)
			staff.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("staff_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), staffHandler.UpdateStaff)
			staff.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("staff_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), staffHandler.DeleteStaff)
			staff.GET("/:id/leave-balances", staffHandler.GetStaffLeaveBalance)
			staff.GET("/:id/attendance", staffHandler.GetStaffAttendance)
		}

		students := api.Group("/students")
		students.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			students.GET("", studentHandler.GetStudents)
			students.GET("/:id", studentHandler.GetStudent)
			students.POST("", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("student_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), studentHandler.CreateStudent)
			students.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("student_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), studentHandler.UpdateStudent)
			students.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("student_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), studentHandler.DeleteStudent)
			students.GET("/:id/enrollments", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), studentHandler.GetStudentEnrollments)
			students.POST("/enrollments", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("student_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), studentHandler.CreateEnrollment)
			students.GET("/:id/attendance", studentHandler.GetStudentAttendance)
			students.GET("/:id/fees", studentHandler.GetStudentFees)
			students.GET("/:id/marks", studentHandler.GetStudentMarks)
			students.GET("/:id/transport", studentHandler.GetStudentTransport)
		}

		attendance := api.Group("/attendance")
		attendance.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			attendance.GET("/sessions", attendanceHandler.GetAttendanceSessions)
			attendance.POST("/sessions", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.RateLimitMiddleware("attendance_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), attendanceHandler.CreateAttendanceSession)
			attendance.POST("/sessions/:session_id/mark", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.RateLimitMiddleware("attendance_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), attendanceHandler.MarkStudentAttendance)
			attendance.GET("/summary", attendanceHandler.GetStudentAttendanceSummary)
			attendance.POST("/staff", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("attendance_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), attendanceHandler.MarkStaffAttendance)
		}

		exams := api.Group("/exams")
		exams.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			exams.GET("/types", examHandler.GetExamTypes)
			exams.POST("/types", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("exam_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), examHandler.CreateExamType)
			exams.GET("", examHandler.GetExams)
			exams.GET("/:id", examHandler.GetExam)
			exams.POST("", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("exam_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), examHandler.CreateExam)
			exams.POST("/schedules", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("exam_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), examHandler.CreateExamSchedule)
			exams.POST("/schedules/:schedule_id/marks", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.RateLimitMiddleware("exam_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), examHandler.EnterMarks)
			exams.GET("/report-cards", examHandler.GetReportCards)
			exams.GET("/grading-scale", examHandler.GetGradingScale)
		}

		fees := api.Group("/fees")
		fees.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			fees.GET("/categories", feeHandler.GetFeeCategories)
			fees.POST("/categories", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("fee_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), feeHandler.CreateFeeCategory)
			fees.GET("/structures", feeHandler.GetFeeStructures)
			fees.POST("/structures", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("fee_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), feeHandler.CreateFeeStructure)
			fees.GET("/invoices", feeHandler.GetInvoices)
			fees.POST("/invoices", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("fee_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), feeHandler.CreateInvoice)
			fees.POST("/payments", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("fee_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), feeHandler.RecordPayment)
			fees.GET("/concessions", feeHandler.GetConcessions)
		}

		leave := api.Group("/leave")
		leave.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			leave.GET("/types", leaveHandler.GetLeaveTypes)
			leave.POST("/types", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("leave_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), leaveHandler.CreateLeaveType)
			leave.GET("/applications", leaveHandler.GetLeaveApplications)
			leave.POST("/applications", middleware.RateLimitMiddleware("leave_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), leaveHandler.CreateLeaveApplication)
			leave.PUT("/applications/:id/approve", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("leave_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), leaveHandler.ApproveLeaveApplication)
			leave.GET("/balances", leaveHandler.GetLeaveBalances)
			leave.POST("/balances/initialize", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("leave_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), leaveHandler.InitializeLeaveBalances)
		}

		timetable := api.Group("/timetable")
		timetable.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			timetable.GET("/slots", timetableHandler.GetTimetableSlots)
			timetable.POST("/slots", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("timetable_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), timetableHandler.CreateTimetableSlot)
			timetable.PUT("/slots/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("timetable_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), timetableHandler.UpdateTimetableSlot)
			timetable.DELETE("/slots/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("timetable_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), timetableHandler.DeleteTimetableSlot)
			timetable.GET("/substitutions", timetableHandler.GetSubstitutions)
			timetable.POST("/substitutions", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("timetable_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), timetableHandler.CreateSubstitution)
			timetable.GET("/section/:section_id", timetableHandler.GetTimetableBySection)
		}

		announcements := api.Group("/announcements")
		announcements.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			announcements.GET("", announcementHandler.GetAnnouncements)
			announcements.POST("", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.RateLimitMiddleware("announcement_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), announcementHandler.CreateAnnouncement)
		}

		events := api.Group("/events")
		events.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			events.GET("", announcementHandler.GetEvents)
			events.POST("", middleware.RBACMiddleware("Admin", "Principal"), middleware.RateLimitMiddleware("event_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), announcementHandler.CreateEvent)
		}

		notifications := api.Group("/notifications")
		notifications.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
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
