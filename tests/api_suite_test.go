package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"school-backend/internal/config"
	"school-backend/internal/database"
	"school-backend/internal/models"

	_ "github.com/glebarez/go-sqlite"
)

const (
	baseURL = "http://127.0.0.1:19081/api/v1"
	pass    = "TestPass@2026"
)

type report struct {
	GeneratedAt string        `json:"generated_at"`
	Total       int           `json:"total_tests"`
	Passed      int           `json:"passed"`
	Failed      int           `json:"failed"`
	SuccessPct  float64       `json:"success_percent"`
	Tests       []reportEntry `json:"tests"`
}

type reportEntry struct {
	Name         string `json:"test_name"`
	Endpoint     string `json:"endpoint"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	StatusCode   int    `json:"status_code"`
	ResponseTime string `json:"response_time"`
	ResponseBody string `json:"response_body"`
	Error        string `json:"error_message,omitempty"`
}

type suite struct {
	client *http.Client
	tokens map[string]string
	ids    map[string]string
	rows   []reportEntry
}

func TestCompleteAPISuite(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(t.TempDir(), "schooldesk-api-suite.db")
	if err := prepareDatabase(dbPath); err != nil {
		t.Fatalf("prepare db: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", ".")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"PORT=19081",
		"JWT_SECRET=12345678901234567890123456789012",
		"DATABASE_DSN="+dbPath,
		"MIGRATE_ON_START=true",
		"SEED_ON_START=false",
		"GIN_MODE=release",
	)
	var logs bytes.Buffer
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start backend: %v", err)
	}
	defer func() {
		cancel()
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		_ = cmd.Wait()
	}()
	if err := waitForHealth(); err != nil {
		t.Fatalf("backend did not become healthy: %v\n%s", err, logs.String())
	}

	s := &suite{
		client: &http.Client{Timeout: 5 * time.Second},
		tokens: map[string]string{},
		ids: map[string]string{
			"school":          "school-suite",
			"year":            "year-suite",
			"term":            "term-suite-1",
			"grade":           "grade-suite-10",
			"section":         "section-suite-a",
			"outside_section": "section-suite-b",
			"subject":         "subject-suite-math",
			"staff":           "staff-suite-teacher",
			"student":         "student-suite-linked",
			"other_student":   "student-suite-other",
			"outside_student": "student-suite-outside",
			"enrollment":      "enrollment-suite-linked",
			"fee_category":    "fee-category-suite",
		},
	}

	s.login("Admin", "admin@suite.test", pass)
	s.login("Principal", "principal@suite.test", pass)
	s.login("Teacher", "teacher@suite.test", pass)
	s.login("Parent", "parent@suite.test", pass)

	s.expect("Admin dashboard metrics", "GET", "/dashboard/admin", "Admin", "Admin", nil, http.StatusOK)
	s.expect("Principal dashboard metrics", "GET", "/dashboard/principal", "Principal", "Principal", nil, http.StatusOK)
	s.expect("Teacher dashboard metrics", "GET", "/dashboard/teacher", "Teacher", "Teacher", nil, http.StatusOK)
	s.expect("Parent dashboard metrics", "GET", "/dashboard/parent", "Parent", "Parent", nil, http.StatusOK)
	s.expect("Parent forbidden admin dashboard", "GET", "/dashboard/admin", "Parent", "Parent", nil, http.StatusForbidden)

	s.expect("Unauthorized students list", "GET", "/students", "", "Anonymous", nil, http.StatusUnauthorized)
	s.expect("Parent forbidden create student", "POST", "/students", "Parent", "Parent", map[string]any{"first_name": "No", "last_name": "Access", "date_of_birth": "2015-01-01", "gender": "male"}, http.StatusForbidden)

	s.expectDataID("Admin create student", "POST", "/students", "Admin", "Admin", map[string]any{
		"first_name": "Runtime", "last_name": "Student", "date_of_birth": "2015-01-01", "gender": "female",
		"admission_number": "RT-001", "student_code": "RT-001", "current_section_id": s.ids["section"],
	}, http.StatusCreated, "new_student")
	s.expectDataID("Admin assign enrollment", "POST", "/students/enrollments", "Admin", "Admin", map[string]any{
		"student_id": s.ids["new_student"], "section_id": s.ids["section"], "academic_year_id": s.ids["year"], "roll_number": "77", "enrollment_date": "2026-05-01",
	}, http.StatusCreated, "new_enrollment")
	s.expect("Admin link parent to new student", "POST", "/parents/user-parent-suite/students", "Admin", "Admin", map[string]any{
		"admission_numbers": []string{"RT-001"},
	}, http.StatusOK)
	s.expectDataID("Admin create invoice", "POST", "/fees/invoices", "Admin", "Admin", map[string]any{
		"student_id": s.ids["new_student"], "academic_year_id": s.ids["year"], "invoice_number": "RT-INV-001", "invoice_date": "2026-05-01", "due_date": "2026-05-10",
		"total_amount": 1000, "discount_amount": 0, "net_amount": 1000,
		"items": []map[string]any{{"fee_category_id": s.ids["fee_category"], "amount": 1000, "description": "Tuition"}},
	}, http.StatusCreated, "invoice")
	s.expectDataID("Admin create exam type", "POST", "/exams/types", "Admin", "Admin", map[string]any{
		"school_id": s.ids["school"], "name": "Runtime Type", "weightage_percent": 10,
	}, http.StatusCreated, "exam_type")
	s.expectDataID("Admin create exam", "POST", "/exams", "Admin", "Admin", map[string]any{
		"school_id": s.ids["school"], "academic_year_id": s.ids["year"], "term_id": s.ids["term"], "exam_type_id": s.ids["exam_type"],
		"exam_name": "Runtime Exam", "start_date": "2026-05-15", "end_date": "2026-05-16",
	}, http.StatusCreated, "exam")
	s.expectDataID("Admin create exam schedule", "POST", "/exams/schedules", "Admin", "Admin", map[string]any{
		"exam_id": s.ids["exam"], "grade_id": s.ids["grade"], "section_id": s.ids["section"], "subject_id": s.ids["subject"],
		"exam_date": "2026-05-15", "start_time": "09:00", "end_time": "10:00", "max_marks": 100, "pass_marks": 35,
	}, http.StatusCreated, "schedule")
	s.expectDataID("Admin create outside exam schedule", "POST", "/exams/schedules", "Admin", "Admin", map[string]any{
		"exam_id": s.ids["exam"], "grade_id": s.ids["grade"], "section_id": s.ids["outside_section"], "subject_id": s.ids["subject"],
		"exam_date": "2026-05-15", "start_time": "11:00", "end_time": "12:00", "max_marks": 100, "pass_marks": 35,
	}, http.StatusCreated, "outside_schedule")

	s.expect("Teacher list assigned section students", "GET", "/students?section_id="+s.ids["section"], "Teacher", "Teacher", nil, http.StatusOK)
	s.expect("Teacher forbidden outside section students", "GET", "/students?section_id="+s.ids["outside_section"], "Teacher", "Teacher", nil, http.StatusForbidden)
	s.expectDataID("Teacher create attendance session", "POST", "/attendance/sessions", "Teacher", "Teacher", map[string]any{
		"section_id": s.ids["section"], "subject_id": s.ids["subject"], "staff_id": s.ids["staff"], "date": "2026-05-01", "period_number": 1,
	}, http.StatusCreated, "attendance_session")
	s.expectNoWrite("Attendance rejects invalid student", "POST", "/attendance/sessions/"+s.ids["attendance_session"]+"/mark", "Teacher", "Teacher", map[string]any{
		"attendances": []map[string]any{{"student_id": s.ids["outside_student"], "enrollment_id": "enrollment-suite-outside", "status": "present"}},
	}, http.StatusForbidden, "student_attendances")
	s.expect("Teacher mark attendance", "POST", "/attendance/sessions/"+s.ids["attendance_session"]+"/mark", "Teacher", "Teacher", map[string]any{
		"attendances": []map[string]any{{"student_id": s.ids["new_student"], "enrollment_id": s.ids["new_enrollment"], "status": "present"}},
	}, http.StatusOK)
	s.expectDataID("Teacher assign homework", "POST", "/homework", "Teacher", "Teacher", map[string]any{
		"title": "Runtime Homework", "subject": "Mathematics", "class": "Grade 10-A", "section_id": s.ids["section"],
		"teacher_id": s.ids["staff"], "student_id": s.ids["new_student"], "description": "API suite homework",
		"due_date": "2026-05-03T00:00:00Z", "status": "pending", "created_by": s.ids["staff"],
	}, http.StatusCreated, "homework")
	s.expectDataID("Admin create other child homework", "POST", "/homework", "Admin", "Admin", map[string]any{
		"title": "Other Homework", "subject": "Mathematics", "class": "Grade 10-A", "section_id": s.ids["section"],
		"teacher_id": s.ids["staff"], "student_id": s.ids["other_student"], "description": "Not linked to parent",
		"due_date": "2026-05-03T00:00:00Z", "status": "pending", "created_by": s.ids["staff"],
	}, http.StatusCreated, "other_homework")
	s.expectDataID("Teacher write diary", "POST", "/diary-entries", "Teacher", "Teacher", map[string]any{
		"date": "2026-05-01T00:00:00Z", "class": "Grade 10-A", "section_id": s.ids["section"], "subject": "Mathematics",
		"title": "Runtime Diary", "classwork": "API suite", "homework": "Complete assignment", "type": "regular",
		"teacher_id": s.ids["staff"], "student_id": s.ids["new_student"], "created_by": "Suite Teacher",
	}, http.StatusCreated, "diary")
	s.expectNoWrite("Teacher forbidden homework outside section", "POST", "/homework", "Teacher", "Teacher", map[string]any{
		"title": "Forbidden Homework", "subject": "Mathematics", "class": "Grade 10-B", "section_id": s.ids["outside_section"],
		"teacher_id": s.ids["staff"], "student_id": s.ids["outside_student"], "description": "Should fail",
		"due_date": "2026-05-03T00:00:00Z", "status": "pending", "created_by": s.ids["staff"],
	}, http.StatusForbidden, "homework")
	s.expect("Teacher enter marks", "POST", "/exams/schedules/"+s.ids["schedule"]+"/marks", "Teacher", "Teacher", map[string]any{
		"marks": []map[string]any{{"student_id": s.ids["new_student"], "enrollment_id": s.ids["new_enrollment"], "marks_obtained": 92, "grade_label": "A+"}},
	}, http.StatusOK)
	s.expectNoWrite("Teacher forbidden marks outside section", "POST", "/exams/schedules/"+s.ids["outside_schedule"]+"/marks", "Teacher", "Teacher", map[string]any{
		"marks": []map[string]any{{"student_id": s.ids["outside_student"], "enrollment_id": "enrollment-suite-outside", "marks_obtained": 88, "grade_label": "A"}},
	}, http.StatusForbidden, "student_marks")

	s.expect("Parent fetch linked child", "GET", "/me/students", "Parent", "Parent", nil, http.StatusOK)
	s.expect("Parent view attendance", "GET", "/students/"+s.ids["new_student"]+"/attendance", "Parent", "Parent", nil, http.StatusOK)
	s.expect("Parent blocked from other child fees", "GET", "/students/"+s.ids["other_student"]+"/fees", "Parent", "Parent", nil, http.StatusForbidden)
	s.expect("Parent blocked from other child attendance", "GET", "/students/"+s.ids["other_student"]+"/attendance", "Parent", "Parent", nil, http.StatusForbidden)
	s.expect("Parent blocked from other child marks", "GET", "/students/"+s.ids["other_student"]+"/marks", "Parent", "Parent", nil, http.StatusForbidden)
	s.expect("Parent blocked from other child transport", "GET", "/students/"+s.ids["other_student"]+"/transport", "Parent", "Parent", nil, http.StatusForbidden)
	s.expect("Teacher blocked from outside class fees", "GET", "/students/"+s.ids["outside_student"]+"/fees", "Teacher", "Teacher", nil, http.StatusForbidden)
	s.expect("Teacher blocked from outside class attendance", "GET", "/students/"+s.ids["outside_student"]+"/attendance", "Teacher", "Teacher", nil, http.StatusForbidden)
	s.expect("Teacher blocked from outside class marks", "GET", "/students/"+s.ids["outside_student"]+"/marks", "Teacher", "Teacher", nil, http.StatusForbidden)
	s.expect("Teacher blocked from outside class transport", "GET", "/students/"+s.ids["outside_student"]+"/transport", "Teacher", "Teacher", nil, http.StatusForbidden)
	s.expect("Parent forbidden other homework detail", "GET", "/homework/"+s.ids["other_homework"], "Parent", "Parent", nil, http.StatusForbidden)
	s.expect("Parent view homework", "GET", "/homework", "Parent", "Parent", nil, http.StatusOK)
	s.expectNoWrite("Parent forbidden conversation for other child", "POST", "/message-conversations", "Parent", "Parent", map[string]any{
		"reference_type": "homework", "reference_id": s.ids["homework"], "teacher_id": s.ids["staff"], "parent_id": "user-parent-suite",
		"student_id": s.ids["other_student"], "title": "Forbidden Conversation", "last_message": "", "last_message_time": "2026-05-01T00:00:00Z",
	}, http.StatusForbidden, "message_conversations")
	s.expectDataID("Parent create message conversation", "POST", "/message-conversations", "Parent", "Parent", map[string]any{
		"reference_type": "homework", "reference_id": s.ids["homework"], "teacher_id": s.ids["staff"], "parent_id": "user-parent-suite",
		"student_id": s.ids["new_student"], "title": "Runtime Conversation", "last_message": "", "last_message_time": "2026-05-01T00:00:00Z",
	}, http.StatusCreated, "conversation")
	s.expectDataID("Parent reply message", "POST", "/messages", "Parent", "Parent", map[string]any{
		"conversation_id": s.ids["conversation"], "sender_id": "user-parent-suite", "sender_role": "parent", "sender_name": "Suite Parent",
		"body": "Acknowledged", "sent_at": "2026-05-01T00:01:00Z",
	}, http.StatusCreated, "message")
	s.expect("Parent view fees", "GET", "/students/"+s.ids["new_student"]+"/fees", "Parent", "Parent", nil, http.StatusOK)

	s.expectDataID("Teacher submit leave", "POST", "/leave/applications", "Teacher", "Teacher", map[string]any{
		"staff_id": s.ids["staff"], "leave_type_id": "leave-suite", "from_date": "2026-05-07", "to_date": "2026-05-07", "reason": "API suite",
	}, http.StatusCreated, "leave")
	s.expectNoWrite("Teacher forbidden leave for other staff", "POST", "/leave/applications", "Teacher", "Teacher", map[string]any{
		"staff_id": "staff-not-self", "leave_type_id": "leave-suite", "from_date": "2026-05-08", "to_date": "2026-05-08", "reason": "Should fail",
	}, http.StatusForbidden, "leave_applications")
	s.expect("Principal list approvals", "GET", "/leave/applications", "Principal", "Principal", nil, http.StatusOK)
	s.expect("Principal approve leave", "PUT", "/leave/applications/"+s.ids["leave"]+"/approve", "Principal", "Principal", map[string]any{"status": "approved"}, http.StatusOK)
	s.expectDataID("Principal create announcement", "POST", "/announcements", "Principal", "Principal", map[string]any{
		"school_id": s.ids["school"], "title": "Runtime Announcement", "content": "API suite", "target_audience": "all", "created_by": "user-principal-suite",
	}, http.StatusCreated, "announcement")
	s.expect("Principal analytics students", "GET", "/students", "Principal", "Principal", nil, http.StatusOK)
	s.expect("Principal monitoring audit logs", "GET", "/audit-logs", "Principal", "Principal", nil, http.StatusOK)
	s.expect("Admin fetch audit logs", "GET", "/audit-logs", "Admin", "Admin", nil, http.StatusOK)
	s.expectAuditContains("Audit attendance marking", "Admin", "attendance", "update", "student_attendances", s.ids["attendance_session"], "Teacher")
	s.expectAuditContains("Audit exam marks entry", "Admin", "exams", "create", "student_marks", "", "Teacher")
	s.expectAuditContains("Audit leave approval", "Admin", "leave", "update", "leave_applications", s.ids["leave"], "Principal")
	s.expectAuditContains("Audit message conversation", "Admin", "message_conversations", "create", "message_conversations", s.ids["conversation"], "Parent")
	s.expectAuditContains("Audit message reply", "Admin", "messages", "create", "messages", s.ids["message"], "Parent")

	s.expect("Admin record fee payment", "POST", "/fees/payments", "Admin", "Admin", map[string]any{
		"invoice_id": s.ids["invoice"], "receipt_number": "RT-RCPT-001", "amount_paid": 1000, "payment_date": "2026-05-01", "payment_mode": "cash",
	}, http.StatusOK)
	s.expectAuditContains("Audit fee payment", "Admin", "fees", "create", "payments", "", "Admin")
	s.expect("Admin verify paid invoice", "GET", "/fees/invoices?student_id="+s.ids["new_student"], "Admin", "Admin", nil, http.StatusOK)
	s.expect("Admin verify marks", "GET", "/students/"+s.ids["new_student"]+"/marks?exam_id="+s.ids["exam"], "Admin", "Admin", nil, http.StatusOK)

	rep := s.buildReport()
	if err := writeReports(root, rep); err != nil {
		t.Fatalf("write reports: %v", err)
	}
	if rep.Failed > 0 {
		t.Fatalf("api suite failed: %d/%d failed; see test-report/report.html", rep.Failed, rep.Total)
	}
}

func prepareDatabase(path string) error {
	cfg := &config.Config{Environment: "test", DatabaseDSN: path, JWTSecret: "12345678901234567890123456789012", MigrateOnStart: true}
	if err := database.Initialize(cfg); err != nil {
		return err
	}
	if err := seedFixtures(); err != nil {
		return err
	}
	sqlDB, err := database.DB.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
	return nil
}

func seedFixtures() error {
	hash, err := database.HashPassword(pass)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	create := func(v any) error { return database.DB.Create(v).Error }
	schoolID, yearID, termID := "school-suite", "year-suite", "term-suite-1"
	gradeID, sectionID, subjectID, deptID := "grade-suite-10", "section-suite-a", "subject-suite-math", "dept-suite"
	staffID, studentID, otherStudentID, outsideStudentID, enrollmentID := "staff-suite-teacher", "student-suite-linked", "student-suite-other", "student-suite-outside", "enrollment-suite-linked"
	roles := map[string]string{"Admin": "role-admin-suite", "Principal": "role-principal-suite", "Teacher": "role-teacher-suite", "Parent": "role-parent-suite"}
	if err := create(&models.School{BaseModel: models.BaseModel{ID: schoolID}, Name: "Suite School", SchoolType: "cbse", Timezone: "Asia/Kolkata", Currency: "INR"}); err != nil {
		return err
	}
	if err := create(&models.AcademicYear{BaseModel: models.BaseModel{ID: yearID}, SchoolID: schoolID, YearLabel: "2026-2027", StartDate: now, EndDate: now.AddDate(1, 0, 0), IsCurrent: true, Status: "active"}); err != nil {
		return err
	}
	if err := create(&models.Term{BaseModel: models.BaseModel{ID: termID}, AcademicYearID: yearID, TermNumber: 1, TermName: "Term 1", StartDate: now, EndDate: now.AddDate(0, 6, 0), IsCurrent: true}); err != nil {
		return err
	}
	if err := create(&models.Department{BaseModel: models.BaseModel{ID: deptID}, SchoolID: schoolID, DepartmentName: "Math"}); err != nil {
		return err
	}
	if err := create(&models.Grade{BaseModel: models.BaseModel{ID: gradeID}, SchoolID: schoolID, GradeNumber: 10, GradeName: "Grade 10"}); err != nil {
		return err
	}
	if err := create(&models.Subject{BaseModel: models.BaseModel{ID: subjectID}, SchoolID: schoolID, DepartmentID: deptID, SubjectName: "Mathematics", SubjectCode: "MATH", SubjectType: "core"}); err != nil {
		return err
	}
	if err := create(&models.Section{BaseModel: models.BaseModel{ID: sectionID}, GradeID: gradeID, AcademicYearID: yearID, SectionName: "A", ClassTeacherID: &staffID, Capacity: 40}); err != nil {
		return err
	}
	if err := create(&models.Section{BaseModel: models.BaseModel{ID: "section-suite-b"}, GradeID: gradeID, AcademicYearID: yearID, SectionName: "B", Capacity: 40}); err != nil {
		return err
	}
	if err := create(&models.Staff{BaseModel: models.BaseModel{ID: staffID}, SchoolID: schoolID, StaffCode: "T-001", FirstName: "Suite", LastName: "Teacher", Email: "teacher@suite.test", DateOfBirth: now.AddDate(-35, 0, 0), Gender: "female", Designation: "Teacher", EmploymentType: "permanent", JoinDate: now.AddDate(-5, 0, 0), Status: "active"}); err != nil {
		return err
	}
	for role, id := range roles {
		if err := create(&models.Role{BaseModel: models.BaseModel{ID: id}, SchoolID: schoolID, RoleName: role, IsSystemRole: true}); err != nil {
			return err
		}
	}
	users := []models.User{
		{BaseModel: models.BaseModel{ID: "user-admin-suite"}, SchoolID: schoolID, Username: "admin", Email: "admin@suite.test", PasswordHash: hash, RoleID: roles["Admin"], LinkedType: "staff", IsActive: true, IsVerified: true},
		{BaseModel: models.BaseModel{ID: "user-principal-suite"}, SchoolID: schoolID, Username: "principal", Email: "principal@suite.test", PasswordHash: hash, RoleID: roles["Principal"], LinkedType: "staff", IsActive: true, IsVerified: true},
		{BaseModel: models.BaseModel{ID: "user-teacher-suite"}, SchoolID: schoolID, Username: "teacher", Email: "teacher@suite.test", PasswordHash: hash, RoleID: roles["Teacher"], LinkedType: "staff", LinkedID: &staffID, IsActive: true, IsVerified: true},
		{BaseModel: models.BaseModel{ID: "user-parent-suite"}, SchoolID: schoolID, Username: "parent", Email: "parent@suite.test", PasswordHash: hash, RoleID: roles["Parent"], LinkedType: "guardian", IsActive: true, IsVerified: true},
	}
	for i := range users {
		if err := create(&users[i]); err != nil {
			return err
		}
	}
	if err := create(&models.Student{BaseModel: models.BaseModel{ID: studentID}, SchoolID: schoolID, StudentCode: "ST-001", AdmissionNumber: "ADM-001", FirstName: "Linked", LastName: "Child", DateOfBirth: now.AddDate(-10, 0, 0), Gender: "male", AdmissionDate: now, CurrentSectionID: &sectionID, Status: "active"}); err != nil {
		return err
	}
	if err := create(&models.Student{BaseModel: models.BaseModel{ID: otherStudentID}, SchoolID: schoolID, StudentCode: "ST-002", AdmissionNumber: "ADM-002", FirstName: "Other", LastName: "Child", DateOfBirth: now.AddDate(-10, 0, 0), Gender: "female", AdmissionDate: now, CurrentSectionID: &sectionID, Status: "active"}); err != nil {
		return err
	}
	outsideSectionID := "section-suite-b"
	if err := create(&models.Student{BaseModel: models.BaseModel{ID: outsideStudentID}, SchoolID: schoolID, StudentCode: "ST-003", AdmissionNumber: "ADM-003", FirstName: "Outside", LastName: "Class", DateOfBirth: now.AddDate(-10, 0, 0), Gender: "female", AdmissionDate: now, CurrentSectionID: &outsideSectionID, Status: "active"}); err != nil {
		return err
	}
	if err := create(&models.Enrollment{BaseModel: models.BaseModel{ID: enrollmentID}, StudentID: studentID, SectionID: sectionID, AcademicYearID: yearID, RollNumber: "1", EnrollmentDate: now, Status: "enrolled"}); err != nil {
		return err
	}
	if err := create(&models.Enrollment{BaseModel: models.BaseModel{ID: "enrollment-suite-outside"}, StudentID: outsideStudentID, SectionID: outsideSectionID, AcademicYearID: yearID, RollNumber: "2", EnrollmentDate: now, Status: "enrolled"}); err != nil {
		return err
	}
	if err := create(&models.ParentStudentLink{SchoolID: schoolID, ParentUserID: "user-parent-suite", StudentID: studentID, StudentAdmissionNumber: "ADM-001"}); err != nil {
		return err
	}
	if err := create(&models.FeeCategory{BaseModel: models.BaseModel{ID: "fee-category-suite"}, SchoolID: schoolID, CategoryName: "Tuition", Frequency: "monthly"}); err != nil {
		return err
	}
	if err := create(&models.LeaveType{BaseModel: models.BaseModel{ID: "leave-suite"}, SchoolID: schoolID, LeaveName: "Casual", MaxDaysPerYear: 12, ApplicableTo: "all"}); err != nil {
		return err
	}
	return nil
}

func waitForHealth() error {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://127.0.0.1:19081/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout")
}

func (s *suite) login(role, username, password string) {
	body := map[string]any{"username": username, "password": password}
	entry, data := s.do("Login "+role, "POST", "/auth/login", "", role, body, http.StatusOK)
	if entry.Status == "PASS" {
		if token, ok := data["token"].(string); ok {
			s.tokens[role] = token
		}
	}
}

func (s *suite) expect(name, method, path, tokenRole, role string, body any, want int) {
	s.do(name, method, path, tokenRole, role, body, want)
}

func (s *suite) expectDataID(name, method, path, tokenRole, role string, body any, want int, key string) {
	entry, data := s.do(name, method, path, tokenRole, role, body, want)
	if entry.Status == "PASS" {
		if id, ok := data["id"].(string); ok && id != "" {
			s.ids[key] = id
			return
		}
		s.rows[len(s.rows)-1].Status = "FAIL"
		s.rows[len(s.rows)-1].Error = "response data.id missing"
	}
}

func (s *suite) expectNoWrite(name, method, path, tokenRole, role string, body any, want int, table string) {
	before := tableCount(table)
	s.expect(name, method, path, tokenRole, role, body, want)
	after := tableCount(table)
	if before != after {
		s.rows[len(s.rows)-1].Status = "FAIL"
		s.rows[len(s.rows)-1].Error = fmt.Sprintf("expected no DB write to %s, count changed from %d to %d", table, before, after)
	}
}

func tableCount(table string) int64 {
	var count int64
	database.DB.Table(table).Count(&count)
	return count
}

func (s *suite) expectAuditContains(name, tokenRole, module, action, entityType, entityID, role string) {
	path := "/audit-logs?module=" + module + "&page_size=100"
	req, _ := http.NewRequest("GET", baseURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+s.tokens[tokenRole])
	start := time.Now()
	resp, err := s.client.Do(req)
	elapsed := time.Since(start)
	entry := reportEntry{Name: name, Endpoint: "GET " + path, Role: tokenRole, ResponseTime: elapsed.String()}
	if err != nil {
		entry.Status = "FAIL"
		entry.Error = err.Error()
		s.rows = append(s.rows, entry)
		return
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	entry.StatusCode = resp.StatusCode
	entry.ResponseBody = truncate(buf.String(), 1200)
	if resp.StatusCode != http.StatusOK {
		entry.Status = "FAIL"
		entry.Error = fmt.Sprintf("expected HTTP %d, got %d", http.StatusOK, resp.StatusCode)
		s.rows = append(s.rows, entry)
		return
	}
	var parsed struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		entry.Status = "FAIL"
		entry.Error = "failed to parse audit response: " + err.Error()
		s.rows = append(s.rows, entry)
		return
	}
	for _, row := range parsed.Data {
		if row["module"] != module || row["action"] != action || row["entity_type"] != entityType || row["role"] != role {
			continue
		}
		if entityID != "" && row["entity_id"] != entityID {
			continue
		}
		entry.Status = "PASS"
		s.rows = append(s.rows, entry)
		return
	}
	entry.Status = "FAIL"
	entry.Error = fmt.Sprintf("missing audit row module=%s action=%s entity_type=%s entity_id=%s role=%s", module, action, entityType, entityID, role)
	s.rows = append(s.rows, entry)
}

func (s *suite) do(name, method, path, tokenRole, role string, body any, want int) (reportEntry, map[string]any) {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, baseURL+path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if tokenRole != "" {
		req.Header.Set("Authorization", "Bearer "+s.tokens[tokenRole])
	}
	start := time.Now()
	resp, err := s.client.Do(req)
	elapsed := time.Since(start)
	entry := reportEntry{Name: name, Endpoint: method + " " + path, Role: role, ResponseTime: elapsed.String()}
	if err != nil {
		entry.Status = "FAIL"
		entry.Error = err.Error()
		s.rows = append(s.rows, entry)
		return entry, nil
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	entry.StatusCode = resp.StatusCode
	entry.ResponseBody = truncate(buf.String(), 1200)
	if resp.StatusCode == want {
		entry.Status = "PASS"
	} else {
		entry.Status = "FAIL"
		entry.Error = fmt.Sprintf("expected HTTP %d, got %d", want, resp.StatusCode)
	}
	var parsed struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(buf.Bytes(), &parsed)
	s.rows = append(s.rows, entry)
	return entry, parsed.Data
}

func (s *suite) buildReport() report {
	rep := report{GeneratedAt: time.Now().UTC().Format(time.RFC3339), Tests: s.rows}
	rep.Total = len(s.rows)
	for _, row := range s.rows {
		if row.Status == "PASS" {
			rep.Passed++
		}
	}
	rep.Failed = rep.Total - rep.Passed
	if rep.Total > 0 {
		rep.SuccessPct = float64(rep.Passed) * 100 / float64(rep.Total)
	}
	return rep
}

func writeReports(root string, rep report) error {
	dir := filepath.Join(root, "test-report")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(rep, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "report.json"), raw, 0o644); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset='utf-8'><title>API Test Report</title><style>body{font-family:Arial,sans-serif;margin:24px}table{border-collapse:collapse;width:100%}td,th{border:1px solid #ddd;padding:8px;vertical-align:top}th{background:#f4f4f4}.PASS{color:#0a7a2f;font-weight:bold}.FAIL{color:#b00020;font-weight:bold}pre{white-space:pre-wrap;max-width:520px}</style></head><body>")
	b.WriteString(fmt.Sprintf("<h1>Schooldesk API Test Report</h1><p>Total: %d | Passed: %d | Failed: %d | Success: %.2f%%</p>", rep.Total, rep.Passed, rep.Failed, rep.SuccessPct))
	b.WriteString("<table><thead><tr><th>Test Name</th><th>Endpoint</th><th>Role</th><th>Status</th><th>Status Code</th><th>Response Time</th><th>Error Message</th><th>Response Body</th></tr></thead><tbody>")
	for _, row := range rep.Tests {
		b.WriteString("<tr>")
		b.WriteString("<td>" + html.EscapeString(row.Name) + "</td>")
		b.WriteString("<td>" + html.EscapeString(row.Endpoint) + "</td>")
		b.WriteString("<td>" + html.EscapeString(row.Role) + "</td>")
		b.WriteString("<td class='" + row.Status + "'>" + row.Status + "</td>")
		b.WriteString(fmt.Sprintf("<td>%d</td>", row.StatusCode))
		b.WriteString("<td>" + html.EscapeString(row.ResponseTime) + "</td>")
		b.WriteString("<td>" + html.EscapeString(row.Error) + "</td>")
		b.WriteString("<td><pre>" + html.EscapeString(row.ResponseBody) + "</pre></td>")
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table></body></html>")
	return os.WriteFile(filepath.Join(dir, "report.html"), []byte(b.String()), 0o644)
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "...[truncated]"
}

func TestReportFilesAreReadable(t *testing.T) {
	_, _ = sql.Open("sqlite", ":memory:")
}
