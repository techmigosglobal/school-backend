package database

import (
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"school-backend/internal/config"
	"school-backend/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Initialize(cfg *config.Config) error {
	var err error

	logLevel := logger.Silent
	if cfg.Environment == "development" {
		logLevel = logger.Info
	}

	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logLevel)}
	normalizedDatabaseURL := normalizeDatabaseURL(cfg.DatabaseURL)
	if shouldUsePostgres(normalizedDatabaseURL) {
		DB, err = gorm.Open(postgres.Open(normalizedDatabaseURL), gormCfg)
	} else {
		DB, err = gorm.Open(sqlite.Open(cfg.DatabaseDSN), gormCfg)
	}
	if err != nil {
		return err
	}

	log.Println("Database connected successfully")

	if err := autoMigrate(); err != nil {
		return err
	}

	log.Println("Database migrations completed")

	if err := seedData(); err != nil {
		log.Printf("Warning: Seed data error (may already exist): %v", err)
	}

	return nil
}

func shouldUsePostgres(databaseURL string) bool {
	url := normalizeDatabaseURL(databaseURL)
	if url == "" {
		return false
	}
	// Common formats: postgres://... or postgresql://...
	return strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://")
}

func normalizeDatabaseURL(databaseURL string) string {
	url := strings.TrimSpace(databaseURL)
	if url == "" {
		return ""
	}

	// Tolerate accidental "DATABASE_URL=..." value pasted into env.
	url = strings.TrimPrefix(url, "DATABASE_URL=")
	url = strings.TrimSpace(url)

	// Tolerate malformed scheme "postgres:user:pass@host/db".
	if strings.HasPrefix(url, "postgres:") && !strings.HasPrefix(url, "postgres://") {
		url = "postgres://" + strings.TrimPrefix(url, "postgres:")
	}
	if strings.HasPrefix(url, "postgresql:") && !strings.HasPrefix(url, "postgresql://") {
		url = "postgresql://" + strings.TrimPrefix(url, "postgresql:")
	}

	return url
}

func autoMigrate() error {
	return DB.AutoMigrate(
		&models.School{},
		&models.AcademicYear{},
		&models.Term{},
		&models.Holiday{},
		&models.WorkingDayConfig{},
		&models.Department{},
		&models.Subject{},
		&models.Grade{},
		&models.GradeSubject{},
		&models.Section{},
		&models.Room{},
		&models.Staff{},
		&models.StaffQualification{},
		&models.StaffSubject{},
		&models.StaffDocument{},
		&models.Student{},
		&models.Guardian{},
		&models.MedicalRecord{},
		&models.StudentDocument{},
		&models.Enrollment{},
		&models.TransferRecord{},
		&models.PromotionRule{},
		&models.TimetableSlot{},
		&models.Substitution{},
		&models.AttendanceSession{},
		&models.StudentAttendance{},
		&models.StaffAttendance{},
		&models.AttendanceSummary{},
		&models.ExamType{},
		&models.Exam{},
		&models.ExamSchedule{},
		&models.StudentMark{},
		&models.GradingScale{},
		&models.ReportCard{},
		&models.FeeCategory{},
		&models.FeeStructure{},
		&models.FeeConcession{},
		&models.FeeInvoice{},
		&models.FeeInvoiceItem{},
		&models.Payment{},
		&models.BookCategory{},
		&models.Book{},
		&models.BookIssue{},
		&models.Vehicle{},
		&models.Route{},
		&models.RouteStop{},
		&models.StudentTransport{},
		&models.Announcement{},
		&models.EventCalendar{},
		&models.ParentTeacherMeeting{},
		&models.NotificationLog{},
		&models.LeaveType{},
		&models.LeaveBalance{},
		&models.LeaveApplication{},
		&models.Payroll{},
		&models.Role{},
		&models.Permission{},
		&models.User{},
		&models.UserSession{},
		&models.OTPVerification{},
		&models.AuditLog{},
	)
}

func seedData() error {
	var count int64
	DB.Model(&models.School{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("Seeding initial data...")

	schoolID := "550e8400-e29b-41d4-a716-446655440000"
	school := models.School{
		BaseModel: models.BaseModel{ID: schoolID},
		Name:            "Demo International School",
		SchoolType:      "cbse",
		AffiliationBoard: "CBSE",
		Email:           "info@demoschool.edu",
		Phone:           "+91-9876543210",
		City:            "Mumbai",
		State:           "Maharashtra",
		Timezone:        "Asia/Kolkata",
		Currency:        "INR",
	}
	DB.Create(&school)

	yearID := "660e8400-e29b-41d4-a716-446655440000"
	academicYear := models.AcademicYear{
		BaseModel: models.BaseModel{ID: yearID},
		SchoolID:  schoolID,
		YearLabel: "2025-2026",
		StartDate: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		IsCurrent: true,
		Status:    "active",
	}
	DB.Create(&academicYear)

	term1ID := "770e8400-e29b-41d4-a716-446655440001"
	term1 := models.Term{
		BaseModel:      models.BaseModel{ID: term1ID},
		AcademicYearID: yearID,
		TermNumber:     1,
		TermName:       "First Term",
		StartDate:      time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC),
		IsCurrent:      true,
	}
	DB.Create(&term1)

	term2ID := "770e8400-e29b-41d4-a716-446655440002"
	term2 := models.Term{
		BaseModel:      models.BaseModel{ID: term2ID},
		AcademicYearID: yearID,
		TermNumber:     2,
		TermName:       "Second Term",
		StartDate:      time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		IsCurrent:      false,
	}
	DB.Create(&term2)

	deptID := "880e8400-e29b-41d4-a716-446655440000"
	dept := models.Department{
		BaseModel:       models.BaseModel{ID: deptID},
		SchoolID:        schoolID,
		DepartmentName:  "Science",
		Description:     "Science Department",
	}
	DB.Create(&dept)

	grade1ID := "990e8400-e29b-41d4-a716-446655440001"
	grade1 := models.Grade{
		BaseModel:    models.BaseModel{ID: grade1ID},
		SchoolID:     schoolID,
		GradeNumber:  1,
		GradeName:    "Grade 1",
	}
	DB.Create(&grade1)

	grade2ID := "990e8400-e29b-41d4-a716-446655440002"
	grade2 := models.Grade{
		BaseModel:    models.BaseModel{ID: grade2ID},
		SchoolID:     schoolID,
		GradeNumber:  2,
		GradeName:    "Grade 2",
	}
	DB.Create(&grade2)

	grade10ID := "990e8400-e29b-41d4-a716-446655440010"
	grade10 := models.Grade{
		BaseModel:    models.BaseModel{ID: grade10ID},
		SchoolID:     schoolID,
		GradeNumber:  10,
		GradeName:    "Grade 10",
	}
	DB.Create(&grade10)

	subjID := "aa0e8400-e29b-41d4-a716-446655440001"
	subj := models.Subject{
		BaseModel:     models.BaseModel{ID: subjID},
		SchoolID:      schoolID,
		DepartmentID:  deptID,
		SubjectName:   "Mathematics",
		SubjectCode:   "MATH",
		SubjectType:   "core",
		CreditHours:   4,
	}
	DB.Create(&subj)

	subj2ID := "aa0e8400-e29b-41d4-a716-446655440002"
	subj2 := models.Subject{
		BaseModel:     models.BaseModel{ID: subj2ID},
		SchoolID:      schoolID,
		DepartmentID:  deptID,
		SubjectName:   "Science",
		SubjectCode:   "SCI",
		SubjectType:   "core",
		CreditHours:   4,
	}
	DB.Create(&subj2)

	sectionID := "bb0e8400-e29b-41d4-a716-446655440001"
	section := models.Section{
		BaseModel:      models.BaseModel{ID: sectionID},
		GradeID:        grade10ID,
		AcademicYearID: yearID,
		SectionName:    "A",
		Capacity:       40,
	}
	DB.Create(&section)

	roomID := "cc0e8400-e29b-41d4-a716-446655440001"
	room := models.Room{
		BaseModel:   models.BaseModel{ID: roomID},
		SchoolID:    schoolID,
		RoomNumber:  "101",
		RoomType:    "classroom",
		Block:       "A",
		Floor:       1,
		Capacity:    40,
	}
	DB.Create(&room)

	staffID := "dd0e8400-e29b-41d4-a716-446655440001"
	staff := models.Staff{
		BaseModel:       models.BaseModel{ID: staffID},
		SchoolID:        schoolID,
		StaffCode:       "STF001",
		FirstName:       "John",
		LastName:        "Doe",
		Email:           "john.doe@demoschool.edu",
		Phone:           "+91-9876543211",
		DateOfBirth:     time.Date(1985, 6, 15, 0, 0, 0, 0, time.UTC),
		Gender:          "male",
		DepartmentID:    &deptID,
		Designation:     "Senior Teacher",
		EmploymentType:  "permanent",
		JoinDate:        time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC),
		BasicSalary:     50000,
		Status:          "active",
	}
	DB.Create(&staff)

	studentID := "ee0e8400-e29b-41d4-a716-446655440001"
	student := models.Student{
		BaseModel:         models.BaseModel{ID: studentID},
		SchoolID:          schoolID,
		StudentCode:       "STU001",
		AdmissionNumber:   "ADM2025001",
		FirstName:         "Alice",
		LastName:          "Smith",
		DateOfBirth:       time.Date(2010, 3, 20, 0, 0, 0, 0, time.UTC),
		Gender:            "female",
		CasteCategory:     "general",
		Nationality:       "Indian",
		AdmissionDate:     time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
		CurrentSectionID:  &sectionID,
		Status:            "active",
	}
	DB.Create(&student)

	guardianID := "ff0e8400-e29b-41d4-a716-446655440002"
	guardian := models.Guardian{
		BaseModel:    models.BaseModel{ID: guardianID},
		StudentID:    studentID,
		FullName:     "Robert Smith",
		Relationship: "father",
		Phone:        "+91-9876543212",
		Email:        "robert.smith@email.com",
		Occupation:   "Engineer",
		AnnualIncome: 800000,
		IsPrimary:    true,
		CanPickup:    true,
	}
	DB.Create(&guardian)

	enrollmentID := "ff0e8400-e29b-41d4-a716-446655440001"
	enrollment := models.Enrollment{
		BaseModel:      models.BaseModel{ID: enrollmentID},
		StudentID:      studentID,
		SectionID:      sectionID,
		AcademicYearID: yearID,
		RollNumber:     "1",
		EnrollmentDate: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
		Status:         "enrolled",
	}
	DB.Create(&enrollment)

	roleAdminID := "110e8400-e29b-41d4-a716-446655440001"
	roleAdmin := models.Role{
		BaseModel:     models.BaseModel{ID: roleAdminID},
		SchoolID:      schoolID,
		RoleName:      "Admin",
		Description:   "School Administrator",
		IsSystemRole:  true,
	}
	DB.Create(&roleAdmin)

	roleTeacherID := "110e8400-e29b-41d4-a716-446655440002"
	roleTeacher := models.Role{
		BaseModel:     models.BaseModel{ID: roleTeacherID},
		SchoolID:      schoolID,
		RoleName:      "Teacher",
		Description:   "Teaching Staff",
		IsSystemRole:  true,
	}
	DB.Create(&roleTeacher)

	roleParentID := "110e8400-e29b-41d4-a716-446655440003"
	roleParent := models.Role{
		BaseModel:     models.BaseModel{ID: roleParentID},
		SchoolID:      schoolID,
		RoleName:      "Parent",
		Description:   "Parent/Guardian",
		IsSystemRole:  true,
	}
	DB.Create(&roleParent)

	permAdmin := models.Permission{
		BaseModel: models.BaseModel{ID: "110e8400-e29b-41d4-a716-446655440010"},
		RoleID:    roleAdminID,
		Module:    "dashboard",
		CanRead:   true,
		CanCreate: true,
		CanUpdate: true,
		CanDelete: true,
		CanExport: true,
	}
	DB.Create(&permAdmin)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	userAdminID := "120e8400-e29b-41d4-a716-446655440001"
	userAdmin := models.User{
		BaseModel:     models.BaseModel{ID: userAdminID},
		SchoolID:      schoolID,
		Email:         "admin@demoschool.edu",
		Phone:         "+91-9876543219",
		PasswordHash:  string(hashedPassword),
		RoleID:        roleAdminID,
		LinkedType:    "staff",
		LinkedID:      &staffID,
		IsActive:      true,
		IsVerified:    true,
	}
	DB.Create(&userAdmin)

	feeCatID := "130e8400-e29b-41d4-a716-446655440001"
	feeCat := models.FeeCategory{
		BaseModel:     models.BaseModel{ID: feeCatID},
		SchoolID:      schoolID,
		CategoryName:  "Tuition Fee",
		Frequency:     "monthly",
		IsRefundable:  false,
	}
	DB.Create(&feeCat)

	feeStructID := "130e8400-e29b-41d4-a716-446655440002"
	feeStructure := models.FeeStructure{
		BaseModel:      models.BaseModel{ID: feeStructID},
		SchoolID:       schoolID,
		AcademicYearID: yearID,
		GradeID:        grade10ID,
		FeeCategoryID:  feeCatID,
		Amount:         5000,
		DueDay:         10,
		LateFinePerDay: 50,
	}
	DB.Create(&feeStructure)

	leaveTypeID := "140e8400-e29b-41d4-a716-446655440001"
	leaveType := models.LeaveType{
		BaseModel:        models.BaseModel{ID: leaveTypeID},
		SchoolID:         schoolID,
		LeaveName:        "Casual Leave",
		MaxDaysPerYear:   12,
		CarryForwardDays: 0,
		IsPaid:           false,
		ApplicableTo:     "all",
	}
	DB.Create(&leaveType)

	log.Println("Seed data created successfully")
	return nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
