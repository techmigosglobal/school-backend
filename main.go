package main

import (
	"log"
	"os"
	"time"

	"school-backend/internal/config"
	"school-backend/internal/database"
	"school-backend/internal/handlers"
	"school-backend/internal/middleware"
	"school-backend/internal/models"
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

	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	redisClient, err := platform.NewRedisClient(cfg)
	if err != nil {
		if cfg.Environment == "production" {
			log.Fatalf("Failed to initialize redis: %v", err)
		}
		log.Printf("Redis unavailable, running without redis-backed features: %v", err)
	} else {
		log.Println("Redis connection success")
		services.Cache = services.NewCacheService(redisClient, cfg.Environment)
		services.Rate = services.NewRateLimitService(redisClient, cfg.Environment)
		services.Sessions = services.NewSessionStore(redisClient, cfg.Environment)
		services.Queue = services.NewJobQueue(redisClient, cfg.Environment)
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
	parentLinkHandler := handlers.NewParentLinkHandler()
	userHandler := handlers.NewUserHandler()
	auditLogHandler := handlers.NewAuditLogHandler()
	dashboardHandler := handlers.NewDashboardHandler()

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", middleware.RateLimitMiddleware("auth_login", cfg.RateLimitMaxLogin, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", middleware.AuthMiddleware(), authHandler.Logout)
			if !cfg.DisablePublicRegistration {
				auth.POST("/register-school-admin", middleware.RateLimitMiddleware("auth_register_admin", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), authHandler.RegisterSchoolAdmin)
				auth.POST("/register", middleware.RateLimitMiddleware("auth_register", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), authHandler.Register)
			}
			auth.GET("/profile", middleware.AuthMiddleware(), authHandler.GetProfile)
		}

		dashboard := api.Group("/dashboard")
		dashboard.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			dashboard.GET("/admin", middleware.RBACMiddleware("Admin"), middleware.PermissionMiddleware("dashboard", "read"), dashboardHandler.Admin)
			dashboard.GET("/principal", middleware.RBACMiddleware("Principal"), middleware.PermissionMiddleware("dashboard", "read"), dashboardHandler.Principal)
			dashboard.GET("/teacher", middleware.RBACMiddleware("Teacher"), middleware.PermissionMiddleware("dashboard", "read"), dashboardHandler.Teacher)
			dashboard.GET("/parent", middleware.RBACMiddleware("Parent"), middleware.PermissionMiddleware("dashboard", "read"), dashboardHandler.Parent)
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
			leave.POST("/applications", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.RateLimitMiddleware("leave_write", cfg.RateLimitMaxAPI, time.Duration(cfg.RateLimitWindowSeconds)*time.Second), leaveHandler.CreateLeaveApplication)
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

		parents := api.Group("/parents")
		parents.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			parents.POST("/:parent_user_id/students", middleware.RBACMiddleware("Admin", "Principal"), parentLinkHandler.AssignParentStudents)
			parents.GET("/:parent_user_id/students", middleware.RBACMiddleware("Admin", "Principal"), parentLinkHandler.GetParentStudents)
		}

		me := api.Group("/me")
		me.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			me.GET("/students", middleware.RBACMiddleware("Parent"), parentLinkHandler.GetMyStudents)
		}

		users := api.Group("/users")
		users.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			users.GET("", middleware.RBACMiddleware("Admin", "Principal"), userHandler.GetUsers)
			users.POST("", middleware.RBACMiddleware("Admin", "Principal"), userHandler.CreateUser)
			users.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal"), userHandler.UpdateUser)
			users.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), userHandler.DeleteUser)
		}

		guardians := api.Group("/guardians")
		guardians.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			h := handlers.NewCRUDHandler[models.Guardian]("guardians", "guardians", []string{"student_id", "full_name"}, false, "Student")
			guardians.GET("", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("guardians", "read"), h.List)
			guardians.GET("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("guardians", "read"), h.Get)
			guardians.POST("", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("guardians", "create"), h.Create)
			guardians.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("guardians", "update"), h.Update)
			guardians.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("guardians", "delete"), h.Delete)
		}

		medicalRecords := api.Group("/medical-records")
		medicalRecords.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			h := handlers.NewCRUDHandler[models.MedicalRecord]("medical_records", "medical_records", []string{"student_id"}, false, "Student")
			medicalRecords.GET("", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("medical_records", "read"), h.List)
			medicalRecords.GET("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("medical_records", "read"), h.Get)
			medicalRecords.POST("", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("medical_records", "create"), h.Create)
			medicalRecords.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("medical_records", "update"), h.Update)
			medicalRecords.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("medical_records", "delete"), h.Delete)
		}

		studentDocuments := api.Group("/student-documents")
		studentDocuments.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			h := handlers.NewCRUDHandler[models.StudentDocument]("student_documents", "student_documents", []string{"student_id", "doc_type", "file_url"}, false, "Student")
			studentDocuments.GET("", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("student_documents", "read"), h.List)
			studentDocuments.GET("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("student_documents", "read"), h.Get)
			studentDocuments.POST("", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("student_documents", "create"), h.Create)
			studentDocuments.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("student_documents", "update"), h.Update)
			studentDocuments.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("student_documents", "delete"), h.Delete)
		}

		staffDocuments := api.Group("/staff-documents")
		staffDocuments.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			h := handlers.NewCRUDHandler[models.StaffDocument]("staff_documents", "staff_documents", []string{"staff_id", "doc_type", "file_url"}, false, "Staff")
			staffDocuments.GET("", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("staff_documents", "read"), h.List)
			staffDocuments.GET("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("staff_documents", "read"), h.Get)
			staffDocuments.POST("", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("staff_documents", "create"), h.Create)
			staffDocuments.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("staff_documents", "update"), h.Update)
			staffDocuments.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("staff_documents", "delete"), h.Delete)
		}

		staffSubjects := api.Group("/staff-subjects")
		staffSubjects.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			h := handlers.NewCRUDHandler[models.StaffSubject]("staff_subjects", "staff_subjects", []string{"staff_id", "subject_id", "grade_id"}, false, "Staff", "Subject", "Grade")
			staffSubjects.GET("", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.PermissionMiddleware("staff_subjects", "read"), h.List)
			staffSubjects.GET("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.PermissionMiddleware("staff_subjects", "read"), h.Get)
			staffSubjects.POST("", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("staff_subjects", "create"), h.Create)
			staffSubjects.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("staff_subjects", "update"), h.Update)
			staffSubjects.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("staff_subjects", "delete"), h.Delete)
		}

		staffQualifications := api.Group("/staff-qualifications")
		staffQualifications.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			h := handlers.NewCRUDHandler[models.StaffQualification]("staff_qualifications", "staff_qualifications", []string{"staff_id", "degree"}, false, "Staff")
			staffQualifications.GET("", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.PermissionMiddleware("staff_qualifications", "read"), h.List)
			staffQualifications.GET("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.PermissionMiddleware("staff_qualifications", "read"), h.Get)
			staffQualifications.POST("", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("staff_qualifications", "create"), h.Create)
			staffQualifications.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("staff_qualifications", "update"), h.Update)
			staffQualifications.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("staff_qualifications", "delete"), h.Delete)
		}

		transport := api.Group("/transport")
		transport.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware(), middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("transport", "read"))
		{
			vehicles := handlers.NewCRUDHandler[models.Vehicle]("vehicles", "vehicles", []string{"vehicle_number", "vehicle_type"}, true)
			routes := handlers.NewCRUDHandler[models.Route]("routes", "routes", []string{"route_name"}, true, "Vehicle", "Stops")
			stops := handlers.NewCRUDHandler[models.RouteStop]("route_stops", "route_stops", []string{"route_id", "stop_name"}, false, "Route")
			studentTransport := handlers.NewCRUDHandler[models.StudentTransport]("student_transport", "student_transports", []string{"student_id", "academic_year_id", "route_id", "stop_id"}, false, "Student", "Route", "Stop")
			transport.GET("/vehicles", vehicles.List)
			transport.GET("/vehicles/:id", vehicles.Get)
			transport.POST("/vehicles", middleware.PermissionMiddleware("transport", "create"), vehicles.Create)
			transport.PUT("/vehicles/:id", middleware.PermissionMiddleware("transport", "update"), vehicles.Update)
			transport.DELETE("/vehicles/:id", middleware.PermissionMiddleware("transport", "delete"), vehicles.Delete)
			transport.GET("/routes", routes.List)
			transport.GET("/routes/:id", routes.Get)
			transport.POST("/routes", middleware.PermissionMiddleware("transport", "create"), routes.Create)
			transport.PUT("/routes/:id", middleware.PermissionMiddleware("transport", "update"), routes.Update)
			transport.DELETE("/routes/:id", middleware.PermissionMiddleware("transport", "delete"), routes.Delete)
			transport.GET("/stops", stops.List)
			transport.GET("/stops/:id", stops.Get)
			transport.POST("/stops", middleware.PermissionMiddleware("transport", "create"), stops.Create)
			transport.PUT("/stops/:id", middleware.PermissionMiddleware("transport", "update"), stops.Update)
			transport.DELETE("/stops/:id", middleware.PermissionMiddleware("transport", "delete"), stops.Delete)
			transport.GET("/student-assignments", studentTransport.List)
			transport.GET("/student-assignments/:id", studentTransport.Get)
			transport.POST("/student-assignments", middleware.PermissionMiddleware("transport", "create"), studentTransport.Create)
			transport.PUT("/student-assignments/:id", middleware.PermissionMiddleware("transport", "update"), studentTransport.Update)
			transport.DELETE("/student-assignments/:id", middleware.PermissionMiddleware("transport", "delete"), studentTransport.Delete)
		}

		library := api.Group("/library")
		library.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware(), middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.PermissionMiddleware("library", "read"))
		{
			categories := handlers.NewCRUDHandler[models.BookCategory]("book_categories", "book_categories", []string{"category_name"}, true)
			books := handlers.NewCRUDHandler[models.Book]("books", "books", []string{"category_id", "title"}, true, "Category")
			issues := handlers.NewCRUDHandler[models.BookIssue]("book_issues", "book_issues", []string{"book_id", "borrower_type", "borrower_id"}, false, "Book")
			library.GET("/categories", categories.List)
			library.GET("/categories/:id", categories.Get)
			library.POST("/categories", middleware.PermissionMiddleware("library", "create"), categories.Create)
			library.PUT("/categories/:id", middleware.PermissionMiddleware("library", "update"), categories.Update)
			library.DELETE("/categories/:id", middleware.PermissionMiddleware("library", "delete"), categories.Delete)
			library.GET("/books", books.List)
			library.GET("/books/:id", books.Get)
			library.POST("/books", middleware.PermissionMiddleware("library", "create"), books.Create)
			library.PUT("/books/:id", middleware.PermissionMiddleware("library", "update"), books.Update)
			library.DELETE("/books/:id", middleware.PermissionMiddleware("library", "delete"), books.Delete)
			library.GET("/issues", issues.List)
			library.GET("/issues/:id", issues.Get)
			library.POST("/issues", middleware.PermissionMiddleware("library", "create"), issues.Create)
			library.PUT("/issues/:id", middleware.PermissionMiddleware("library", "update"), issues.Update)
			library.DELETE("/issues/:id", middleware.PermissionMiddleware("library", "delete"), issues.Delete)
		}

		payroll := api.Group("/payroll")
		payroll.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			h := handlers.NewCRUDHandler[models.Payroll]("payroll", "payrolls", []string{"staff_id", "academic_year_id", "month", "year"}, false, "Staff", "AcademicYear")
			payroll.GET("", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("payroll", "read"), h.List)
			payroll.GET("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("payroll", "read"), h.Get)
			payroll.POST("", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("payroll", "create"), h.Create)
			payroll.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("payroll", "update"), h.Update)
			payroll.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("payroll", "delete"), h.Delete)
		}

		ptm := api.Group("/parent-teacher-meetings")
		ptm.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			h := handlers.NewCRUDHandler[models.ParentTeacherMeeting]("parent_teacher_meetings", "parent_teacher_meetings", []string{"event_id", "section_id", "teacher_id", "guardian_id", "student_id"}, false, "Event", "Section", "Teacher", "Guardian", "Student")
			ptm.GET("", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("parent_teacher_meetings", "read"), h.List)
			ptm.GET("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("parent_teacher_meetings", "read"), h.Get)
			ptm.POST("", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("parent_teacher_meetings", "create"), h.Create)
			ptm.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.PermissionMiddleware("parent_teacher_meetings", "update"), h.Update)
			ptm.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("parent_teacher_meetings", "delete"), h.Delete)
		}

		auditLogs := api.Group("/audit-logs")
		auditLogs.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			auditLogs.GET("", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("audit_logs", "read"), auditLogHandler.List)
		}

		homework := api.Group("/homework")
		homework.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			h := handlers.NewCRUDHandler[models.Homework]("homework", "homework", []string{"title"}, true)
			homework.GET("", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("homework", "read"), h.List)
			homework.GET("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("homework", "read"), h.Get)
			homework.POST("", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.PermissionMiddleware("homework", "create"), h.Create)
			homework.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.PermissionMiddleware("homework", "update"), h.Update)
			homework.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.PermissionMiddleware("homework", "delete"), h.Delete)
		}

		diary := api.Group("/diary-entries")
		diary.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			h := handlers.NewCRUDHandler[models.DiaryEntry]("diary_entries", "diary_entries", []string{"title"}, true)
			diary.GET("", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("diary_entries", "read"), h.List)
			diary.GET("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("diary_entries", "read"), h.Get)
			diary.POST("", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.PermissionMiddleware("diary_entries", "create"), h.Create)
			diary.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.PermissionMiddleware("diary_entries", "update"), h.Update)
			diary.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher"), middleware.PermissionMiddleware("diary_entries", "delete"), h.Delete)
		}

		conversations := api.Group("/message-conversations")
		conversations.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			h := handlers.NewCRUDHandler[models.MessageConversation]("message_conversations", "message_conversations", []string{"teacher_id", "parent_id"}, true)
			conversations.GET("", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("message_conversations", "read"), h.List)
			conversations.GET("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("message_conversations", "read"), h.Get)
			conversations.POST("", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("message_conversations", "create"), h.Create)
			conversations.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("message_conversations", "update"), h.Update)
			conversations.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("message_conversations", "delete"), h.Delete)
		}

		messages := api.Group("/messages")
		messages.Use(middleware.AuthMiddleware(), middleware.SchoolScopeMiddleware())
		{
			h := handlers.NewCRUDHandler[models.Message]("messages", "messages", []string{"conversation_id", "sender_id", "sender_role", "body"}, false)
			messages.GET("", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("messages", "read"), h.List)
			messages.GET("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("messages", "read"), h.Get)
			messages.POST("", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("messages", "create"), h.Create)
			messages.PUT("/:id", middleware.RBACMiddleware("Admin", "Principal", "Teacher", "Parent"), middleware.PermissionMiddleware("messages", "update"), h.Update)
			messages.DELETE("/:id", middleware.RBACMiddleware("Admin", "Principal"), middleware.PermissionMiddleware("messages", "delete"), h.Delete)
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
