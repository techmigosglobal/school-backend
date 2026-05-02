package database

import (
	"errors"
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

	gormCfg := &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
	}
	normalizedDatabaseURL := normalizeDatabaseURL(cfg.DatabaseURL)
	if cfg.UsePostgresOnly && !shouldUsePostgres(normalizedDatabaseURL) {
		return errors.New("production requires postgres DATABASE_URL")
	}
	if shouldUsePostgres(normalizedDatabaseURL) {
		DB, err = gorm.Open(postgres.Open(normalizedDatabaseURL), gormCfg)
	} else {
		DB, err = gorm.Open(sqlite.Open(cfg.DatabaseDSN), gormCfg)
	}
	if err != nil {
		return err
	}

	log.Println("Database connected successfully")

	if cfg.MigrateOnStart {
		if err := autoMigrate(); err != nil {
			return err
		}

		log.Println("Migration success")
	}

	if err := ensureBootstrapPrincipal(cfg); err != nil {
		log.Printf("Warning: bootstrap principal setup failed: %v", err)
	}

	if err := ensureDefaultRolePermissions(); err != nil {
		log.Printf("Warning: role permission backfill failed: %v", err)
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
	// Phase 1: foundational tables without heavy cross-dependencies.
	if err := DB.AutoMigrate(
		&models.School{},
		&models.AcademicYear{},
		&models.Term{},
		&models.Holiday{},
		&models.WorkingDayConfig{},
		&models.Department{},
		&models.Subject{},
		&models.Grade{},
		&models.GradeSubject{},
		&models.Room{},
		&models.Role{},
		&models.Permission{},
	); err != nil {
		return err
	}

	// Phase 2: staff/student/auth core.
	if err := DB.AutoMigrate(
		&models.Staff{},
		&models.StaffQualification{},
		&models.StaffSubject{},
		&models.StaffDocument{},
		&models.Section{},
		&models.Student{},
		&models.Guardian{},
		&models.MedicalRecord{},
		&models.StudentDocument{},
		&models.Enrollment{},
		&models.ParentStudentLink{},
		&models.TransferRecord{},
		&models.PromotionRule{},
		&models.User{},
		&models.UserSession{},
		&models.OTPVerification{},
		&models.AuditLog{},
	); err != nil {
		return err
	}

	// Phase 3: operational domains.
	if err := DB.AutoMigrate(
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
		&models.Homework{},
		&models.DiaryEntry{},
		&models.MessageConversation{},
		&models.Message{},
		&models.NotificationLog{},
		&models.LeaveType{},
		&models.LeaveBalance{},
		&models.LeaveApplication{},
		&models.Payroll{},
	); err != nil {
		return err
	}

	return nil
}

func ensureBootstrapPrincipal(cfg *config.Config) error {
	var userCount int64
	if err := DB.Model(&models.User{}).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount > 0 {
		return nil
	}

	bootstrapEmail := strings.ToLower(strings.TrimSpace(cfg.BootstrapPrincipalEmail))
	hashedPassword, err := HashPassword(cfg.BootstrapPrincipalPassword)
	if err != nil {
		return err
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		school := models.School{
			Name:             "SchoolDesk",
			SchoolType:       "K-12",
			AffiliationBoard: "CBSE",
			Email:            "info@schooldesk.local",
			Phone:            "",
			City:             "",
			State:            "",
			Timezone:         "Asia/Kolkata",
			Currency:         "INR",
		}
		if err := tx.Create(&school).Error; err != nil {
			return err
		}

		roleIDs := map[string]string{}
		for _, roleName := range []string{"Principal", "Admin", "Teacher", "Parent"} {
			role := models.Role{
				SchoolID:     school.ID,
				RoleName:     roleName,
				Description:  roleName + " role",
				IsSystemRole: true,
			}
			if err := tx.Create(&role).Error; err != nil {
				return err
			}
			roleIDs[strings.ToLower(roleName)] = role.ID
		}
		seedRolePermissionsWithDB(tx, roleIDs["admin"], roleIDs["principal"], roleIDs["teacher"], roleIDs["parent"])

		staff := models.Staff{
			SchoolID:       school.ID,
			StaffCode:      "PRINC",
			FirstName:      "Principal",
			LastName:       "User",
			Email:          bootstrapEmail,
			Designation:    "Principal",
			EmploymentType: "permanent",
			JoinDate:       time.Now().UTC(),
			Status:         "active",
		}
		if err := tx.Create(&staff).Error; err != nil {
			return err
		}
		staffID := staff.ID
		principal := models.User{
			SchoolID:     school.ID,
			Username:     "PRINC",
			Email:        bootstrapEmail,
			PasswordHash: hashedPassword,
			RoleID:       roleIDs["principal"],
			LinkedType:   "staff",
			LinkedID:     &staffID,
			IsActive:     true,
			IsVerified:   true,
		}
		return tx.Create(&principal).Error
	})
}

func seedRolePermissions(adminRoleID, principalRoleID, teacherRoleID, parentRoleID string) {
	seedRolePermissionsWithDB(DB, adminRoleID, principalRoleID, teacherRoleID, parentRoleID)
}

func seedRolePermissionsWithDB(db *gorm.DB, adminRoleID, principalRoleID, teacherRoleID, parentRoleID string) {
	modules := permissionModules()
	createPermission := func(roleID, module string, read, create, update, delete, export bool) {
		upsertPermissionWithDB(db, roleID, module, read, create, update, delete, export)
	}
	for _, module := range modules {
		createPermission(adminRoleID, module, true, true, true, true, true)
		createPermission(principalRoleID, module, true, true, true, module != "audit_logs", true)

		teacherRead := inList(module, "dashboard", "guardians", "medical_records", "student_documents", "staff_subjects", "staff_qualifications", "library", "parent_teacher_meetings", "homework", "diary_entries", "message_conversations", "messages")
		teacherManage := inList(module, "homework", "diary_entries", "message_conversations", "messages", "parent_teacher_meetings")
		createPermission(teacherRoleID, module, teacherRead, teacherManage, teacherManage, false, false)

		parentRead := inList(module, "dashboard", "guardians", "medical_records", "student_documents", "parent_teacher_meetings", "homework", "diary_entries", "message_conversations", "messages")
		parentCreate := inList(module, "parent_teacher_meetings", "message_conversations", "messages")
		parentUpdate := inList(module, "message_conversations", "messages")
		createPermission(parentRoleID, module, parentRead, parentCreate, parentUpdate, false, false)
	}
}

func permissionModules() []string {
	return []string{
		"dashboard",
		"guardians",
		"medical_records",
		"student_documents",
		"staff_documents",
		"staff_subjects",
		"staff_qualifications",
		"transport",
		"library",
		"payroll",
		"parent_teacher_meetings",
		"homework",
		"diary_entries",
		"message_conversations",
		"messages",
		"audit_logs",
	}
}

func upsertPermission(roleID, module string, read, create, update, delete, export bool) {
	upsertPermissionWithDB(DB, roleID, module, read, create, update, delete, export)
}

func upsertPermissionWithDB(db *gorm.DB, roleID, module string, read, create, update, delete, export bool) {
	permission := models.Permission{}
	values := models.Permission{
		RoleID:    roleID,
		Module:    module,
		CanRead:   read,
		CanCreate: create,
		CanUpdate: update,
		CanDelete: delete,
		CanExport: export,
	}
	db.Where("role_id = ? AND module = ?", roleID, module).
		Assign(values).
		FirstOrCreate(&permission)
}

func inList(value string, items ...string) bool {
	for _, item := range items {
		if value == item {
			return true
		}
	}
	return false
}

func ensureDefaultRolePermissions() error {
	var roles []models.Role
	if err := DB.Where("LOWER(role_name) IN ?", []string{"admin", "principal", "teacher", "parent"}).Find(&roles).Error; err != nil {
		return err
	}
	roleIDs := map[string]string{}
	for _, role := range roles {
		roleIDs[strings.ToLower(strings.TrimSpace(role.RoleName))] = role.ID
	}
	adminRoleID, okAdmin := roleIDs["admin"]
	principalRoleID, okPrincipal := roleIDs["principal"]
	teacherRoleID, okTeacher := roleIDs["teacher"]
	parentRoleID, okParent := roleIDs["parent"]
	if !okAdmin || !okPrincipal || !okTeacher || !okParent {
		return nil
	}
	seedRolePermissions(adminRoleID, principalRoleID, teacherRoleID, parentRoleID)
	return removeDuplicatePermissions()
}

func removeDuplicatePermissions() error {
	var rows []models.Permission
	if err := DB.Order("role_id, module, created_at, id").Find(&rows).Error; err != nil {
		return err
	}
	seen := map[string]string{}
	duplicates := make([]string, 0)
	for _, row := range rows {
		key := row.RoleID + "\x00" + row.Module
		if _, ok := seen[key]; ok {
			duplicates = append(duplicates, row.ID)
			continue
		}
		seen[key] = row.ID
	}
	if len(duplicates) == 0 {
		return nil
	}
	return DB.Where("id IN ?", duplicates).Delete(&models.Permission{}).Error
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
