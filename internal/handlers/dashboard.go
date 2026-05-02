package handlers

import (
	"net/http"
	"strings"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

type feeSummary struct {
	TotalInvoiced float64 `json:"total_invoiced"`
	TotalPaid     float64 `json:"total_paid"`
	TotalBalance  float64 `json:"total_balance"`
	PendingCount  int64   `json:"pending_count"`
	PaidCount     int64   `json:"paid_count"`
	CollectionPct float64 `json:"collection_pct"`
}

type teacherClassSummary struct {
	ID          string `json:"id"`
	SectionName string `json:"section_name"`
	GradeName   string `json:"grade_name"`
}

type parentChildSummary struct {
	ID                string  `json:"id"`
	FirstName         string  `json:"first_name"`
	LastName          string  `json:"last_name"`
	AdmissionNumber   string  `json:"admission_number"`
	CurrentSectionID  *string `json:"current_section_id"`
	PendingFeeBalance float64 `json:"pending_fee_balance"`
}

func (h *DashboardHandler) Admin(c *gin.Context) {
	schoolID := scopedSchoolID(c)
	var row struct {
		TotalStudents     int64   `json:"total_students"`
		ActiveStudents    int64   `json:"active_students"`
		TotalStaff        int64   `json:"total_staff"`
		ActiveStaff       int64   `json:"active_staff"`
		TotalClasses      int64   `json:"total_classes"`
		PendingApprovals  int64   `json:"pending_approvals"`
		UnreadAlerts      int64   `json:"unread_alerts"`
		TotalInvoiced     float64 `json:"total_invoiced"`
		TotalPaid         float64 `json:"total_paid"`
		TotalBalance      float64 `json:"total_balance"`
		PendingInvoices   int64   `json:"pending_invoices"`
		PaidInvoices      int64   `json:"paid_invoices"`
		TodaySessions     int64   `json:"today_sessions"`
		TodayPresent      int64   `json:"today_present"`
		TodayMarked       int64   `json:"today_marked"`
		UpcomingExams     int64   `json:"upcoming_exams"`
		OpenConversations int64   `json:"open_conversations"`
	}
	todayStart, todayEnd := dayRange(time.Now())
	if err := database.DB.Raw(`
		SELECT
			(SELECT COUNT(*) FROM students WHERE school_id = ? AND status != 'inactive') AS total_students,
			(SELECT COUNT(*) FROM students WHERE school_id = ? AND status = 'active') AS active_students,
			(SELECT COUNT(*) FROM staffs WHERE school_id = ?) AS total_staff,
			(SELECT COUNT(*) FROM staffs WHERE school_id = ? AND status = 'active') AS active_staff,
			(SELECT COUNT(*) FROM sections JOIN grades ON grades.id = sections.grade_id WHERE grades.school_id = ?) AS total_classes,
			(SELECT COUNT(*) FROM leave_applications JOIN staffs ON staffs.id = leave_applications.staff_id WHERE staffs.school_id = ? AND leave_applications.status = 'pending') AS pending_approvals,
			(SELECT COUNT(*) FROM notification_logs WHERE school_id = ? AND is_read = false) AS unread_alerts,
			COALESCE((SELECT SUM(fee_invoices.net_amount) FROM fee_invoices JOIN students ON students.id = fee_invoices.student_id WHERE students.school_id = ?), 0) AS total_invoiced,
			COALESCE((SELECT SUM(fee_invoices.paid_amount) FROM fee_invoices JOIN students ON students.id = fee_invoices.student_id WHERE students.school_id = ?), 0) AS total_paid,
			COALESCE((SELECT SUM(fee_invoices.balance) FROM fee_invoices JOIN students ON students.id = fee_invoices.student_id WHERE students.school_id = ?), 0) AS total_balance,
			(SELECT COUNT(*) FROM fee_invoices JOIN students ON students.id = fee_invoices.student_id WHERE students.school_id = ? AND fee_invoices.status != 'paid') AS pending_invoices,
			(SELECT COUNT(*) FROM fee_invoices JOIN students ON students.id = fee_invoices.student_id WHERE students.school_id = ? AND fee_invoices.status = 'paid') AS paid_invoices,
			(SELECT COUNT(*) FROM attendance_sessions JOIN sections ON sections.id = attendance_sessions.section_id JOIN grades ON grades.id = sections.grade_id WHERE grades.school_id = ? AND attendance_sessions.date >= ? AND attendance_sessions.date < ?) AS today_sessions,
			COALESCE((SELECT SUM(attendance_sessions.present_count) FROM attendance_sessions JOIN sections ON sections.id = attendance_sessions.section_id JOIN grades ON grades.id = sections.grade_id WHERE grades.school_id = ? AND attendance_sessions.date >= ? AND attendance_sessions.date < ?), 0) AS today_present,
			COALESCE((SELECT SUM(attendance_sessions.total_students) FROM attendance_sessions JOIN sections ON sections.id = attendance_sessions.section_id JOIN grades ON grades.id = sections.grade_id WHERE grades.school_id = ? AND attendance_sessions.date >= ? AND attendance_sessions.date < ?), 0) AS today_marked,
			(SELECT COUNT(*) FROM exams WHERE school_id = ? AND start_date >= ?) AS upcoming_exams,
			(SELECT COUNT(*) FROM message_conversations WHERE school_id = ?) AS open_conversations
	`, schoolID, schoolID, schoolID, schoolID, schoolID, schoolID, schoolID, schoolID, schoolID, schoolID, schoolID, schoolID, schoolID, todayStart, todayEnd, schoolID, todayStart, todayEnd, schoolID, todayStart, todayEnd, schoolID, todayStart, schoolID).Scan(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to load admin dashboard")
		return
	}
	success(c, http.StatusOK, gin.H{
		"role": "Admin",
		"metrics": gin.H{
			"students":           gin.H{"total": row.TotalStudents, "active": row.ActiveStudents},
			"staff":              gin.H{"total": row.TotalStaff, "active": row.ActiveStaff},
			"classes":            row.TotalClasses,
			"pending_approvals":  row.PendingApprovals,
			"unread_alerts":      row.UnreadAlerts,
			"upcoming_exams":     row.UpcomingExams,
			"open_conversations": row.OpenConversations,
		},
		"fees":             buildFeeSummary(row.TotalInvoiced, row.TotalPaid, row.TotalBalance, row.PendingInvoices, row.PaidInvoices),
		"today_attendance": buildAttendanceSummary(row.TodayPresent, row.TodayMarked, row.TodaySessions),
	}, "")
}

func (h *DashboardHandler) Principal(c *gin.Context) {
	schoolID := scopedSchoolID(c)
	var row struct {
		TotalStudents    int64   `json:"total_students"`
		TotalStaff       int64   `json:"total_staff"`
		TotalClasses     int64   `json:"total_classes"`
		PendingApprovals int64   `json:"pending_approvals"`
		UrgentNotices    int64   `json:"urgent_notices"`
		UpcomingEvents   int64   `json:"upcoming_events"`
		UpcomingExams    int64   `json:"upcoming_exams"`
		TotalInvoiced    float64 `json:"total_invoiced"`
		TotalPaid        float64 `json:"total_paid"`
		TotalBalance     float64 `json:"total_balance"`
		PendingInvoices  int64   `json:"pending_invoices"`
		PaidInvoices     int64   `json:"paid_invoices"`
		TodaySessions    int64   `json:"today_sessions"`
		TodayPresent     int64   `json:"today_present"`
		TodayMarked      int64   `json:"today_marked"`
	}
	todayStart, todayEnd := dayRange(time.Now())
	if err := database.DB.Raw(`
		SELECT
			(SELECT COUNT(*) FROM students WHERE school_id = ? AND status != 'inactive') AS total_students,
			(SELECT COUNT(*) FROM staffs WHERE school_id = ? AND status = 'active') AS total_staff,
			(SELECT COUNT(*) FROM sections JOIN grades ON grades.id = sections.grade_id WHERE grades.school_id = ?) AS total_classes,
			(SELECT COUNT(*) FROM leave_applications JOIN staffs ON staffs.id = leave_applications.staff_id WHERE staffs.school_id = ? AND leave_applications.status = 'pending') AS pending_approvals,
			(SELECT COUNT(*) FROM announcements WHERE school_id = ? AND is_urgent = true) AS urgent_notices,
			(SELECT COUNT(*) FROM event_calendars WHERE school_id = ? AND start_datetime >= ?) AS upcoming_events,
			(SELECT COUNT(*) FROM exams WHERE school_id = ? AND start_date >= ?) AS upcoming_exams,
			COALESCE((SELECT SUM(fee_invoices.net_amount) FROM fee_invoices JOIN students ON students.id = fee_invoices.student_id WHERE students.school_id = ?), 0) AS total_invoiced,
			COALESCE((SELECT SUM(fee_invoices.paid_amount) FROM fee_invoices JOIN students ON students.id = fee_invoices.student_id WHERE students.school_id = ?), 0) AS total_paid,
			COALESCE((SELECT SUM(fee_invoices.balance) FROM fee_invoices JOIN students ON students.id = fee_invoices.student_id WHERE students.school_id = ?), 0) AS total_balance,
			(SELECT COUNT(*) FROM fee_invoices JOIN students ON students.id = fee_invoices.student_id WHERE students.school_id = ? AND fee_invoices.status != 'paid') AS pending_invoices,
			(SELECT COUNT(*) FROM fee_invoices JOIN students ON students.id = fee_invoices.student_id WHERE students.school_id = ? AND fee_invoices.status = 'paid') AS paid_invoices,
			(SELECT COUNT(*) FROM attendance_sessions JOIN sections ON sections.id = attendance_sessions.section_id JOIN grades ON grades.id = sections.grade_id WHERE grades.school_id = ? AND attendance_sessions.date >= ? AND attendance_sessions.date < ?) AS today_sessions,
			COALESCE((SELECT SUM(attendance_sessions.present_count) FROM attendance_sessions JOIN sections ON sections.id = attendance_sessions.section_id JOIN grades ON grades.id = sections.grade_id WHERE grades.school_id = ? AND attendance_sessions.date >= ? AND attendance_sessions.date < ?), 0) AS today_present,
			COALESCE((SELECT SUM(attendance_sessions.total_students) FROM attendance_sessions JOIN sections ON sections.id = attendance_sessions.section_id JOIN grades ON grades.id = sections.grade_id WHERE grades.school_id = ? AND attendance_sessions.date >= ? AND attendance_sessions.date < ?), 0) AS today_marked
	`, schoolID, schoolID, schoolID, schoolID, schoolID, schoolID, todayStart, schoolID, todayStart, schoolID, schoolID, schoolID, schoolID, schoolID, schoolID, todayStart, todayEnd, schoolID, todayStart, todayEnd, schoolID, todayStart, todayEnd).Scan(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to load principal dashboard")
		return
	}
	success(c, http.StatusOK, gin.H{
		"role": "Principal",
		"metrics": gin.H{
			"total_students":    row.TotalStudents,
			"total_staff":       row.TotalStaff,
			"total_classes":     row.TotalClasses,
			"pending_approvals": row.PendingApprovals,
			"urgent_notices":    row.UrgentNotices,
			"upcoming_events":   row.UpcomingEvents,
			"upcoming_exams":    row.UpcomingExams,
		},
		"fees":             buildFeeSummary(row.TotalInvoiced, row.TotalPaid, row.TotalBalance, row.PendingInvoices, row.PaidInvoices),
		"today_attendance": buildAttendanceSummary(row.TodayPresent, row.TodayMarked, row.TodaySessions),
	}, "")
}

func (h *DashboardHandler) Teacher(c *gin.Context) {
	schoolID := scopedSchoolID(c)
	staffID := dashboardStaffID(c)
	if staffID == "" {
		fail(c, http.StatusForbidden, "teacher staff linkage missing")
		return
	}
	todayStart, todayEnd := dayRange(time.Now())
	var row struct {
		AssignedClasses  int64 `json:"assigned_classes"`
		AssignedStudents int64 `json:"assigned_students"`
		TodaySessions    int64 `json:"today_sessions"`
		TodayPresent     int64 `json:"today_present"`
		TodayMarked      int64 `json:"today_marked"`
		HomeworkTotal    int64 `json:"homework_total"`
		HomeworkDue      int64 `json:"homework_due"`
		DiaryToday       int64 `json:"diary_today"`
		UnreadMessages   int64 `json:"unread_messages"`
	}
	if err := database.DB.Raw(`
		WITH assigned_sections AS (
			SELECT DISTINCT sections.id
			FROM sections
			JOIN grades ON grades.id = sections.grade_id
			LEFT JOIN timetable_slots ON timetable_slots.section_id = sections.id
			WHERE grades.school_id = ? AND (sections.class_teacher_id = ? OR timetable_slots.staff_id = ?)
		)
		SELECT
			(SELECT COUNT(*) FROM assigned_sections) AS assigned_classes,
			(SELECT COUNT(*) FROM students WHERE school_id = ? AND current_section_id IN (SELECT id FROM assigned_sections) AND status != 'inactive') AS assigned_students,
			(SELECT COUNT(*) FROM attendance_sessions WHERE staff_id = ? AND date >= ? AND date < ?) AS today_sessions,
			COALESCE((SELECT SUM(present_count) FROM attendance_sessions WHERE staff_id = ? AND date >= ? AND date < ?), 0) AS today_present,
			COALESCE((SELECT SUM(total_students) FROM attendance_sessions WHERE staff_id = ? AND date >= ? AND date < ?), 0) AS today_marked,
			(SELECT COUNT(*) FROM homeworks WHERE school_id = ? AND teacher_id = ?) AS homework_total,
			(SELECT COUNT(*) FROM homeworks WHERE school_id = ? AND teacher_id = ? AND due_date >= ? AND status != 'completed') AS homework_due,
			(SELECT COUNT(*) FROM diary_entries WHERE school_id = ? AND teacher_id = ? AND entry_date >= ? AND entry_date < ?) AS diary_today,
			(SELECT COUNT(*) FROM messages JOIN message_conversations ON message_conversations.id = messages.conversation_id WHERE message_conversations.school_id = ? AND message_conversations.teacher_id = ? AND messages.sender_role = 'parent' AND messages.is_read = false) AS unread_messages
	`, schoolID, staffID, staffID, schoolID, staffID, todayStart, todayEnd, staffID, todayStart, todayEnd, staffID, todayStart, todayEnd, schoolID, staffID, schoolID, staffID, todayStart, schoolID, staffID, todayStart, todayEnd, schoolID, staffID).Scan(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to load teacher dashboard")
		return
	}
	var classes []teacherClassSummary
	if err := database.DB.Raw(`
		SELECT DISTINCT sections.id, sections.section_name, grades.grade_name
		FROM sections
		JOIN grades ON grades.id = sections.grade_id
		LEFT JOIN timetable_slots ON timetable_slots.section_id = sections.id
		WHERE grades.school_id = ? AND (sections.class_teacher_id = ? OR timetable_slots.staff_id = ?)
		ORDER BY grades.grade_number, sections.section_name
	`, schoolID, staffID, staffID).Scan(&classes).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to load assigned classes")
		return
	}
	success(c, http.StatusOK, gin.H{
		"role":     "Teacher",
		"staff_id": staffID,
		"metrics": gin.H{
			"assigned_classes":    row.AssignedClasses,
			"assigned_students":   row.AssignedStudents,
			"homework_total":      row.HomeworkTotal,
			"homework_due":        row.HomeworkDue,
			"diary_entries_today": row.DiaryToday,
			"unread_messages":     row.UnreadMessages,
		},
		"today_attendance": buildAttendanceSummary(row.TodayPresent, row.TodayMarked, row.TodaySessions),
		"assigned_classes": classes,
	}, "")
}

func (h *DashboardHandler) Parent(c *gin.Context) {
	schoolID := scopedSchoolID(c)
	userID := c.GetString("user_id")
	var row struct {
		LinkedChildren    int64   `json:"linked_children"`
		AttendancePresent int64   `json:"attendance_present"`
		AttendanceMarked  int64   `json:"attendance_marked"`
		PendingFees       float64 `json:"pending_fees"`
		PendingInvoices   int64   `json:"pending_invoices"`
		HomeworkOpen      int64   `json:"homework_open"`
		UnreadMessages    int64   `json:"unread_messages"`
	}
	if err := database.DB.Raw(`
		WITH linked_students AS (
			SELECT student_id FROM parent_student_links WHERE school_id = ? AND parent_user_id = ?
		)
		SELECT
			(SELECT COUNT(*) FROM linked_students) AS linked_children,
			(SELECT COUNT(*) FROM student_attendances WHERE student_id IN (SELECT student_id FROM linked_students) AND LOWER(status) IN ('present', 'late')) AS attendance_present,
			(SELECT COUNT(*) FROM student_attendances WHERE student_id IN (SELECT student_id FROM linked_students)) AS attendance_marked,
			COALESCE((SELECT SUM(balance) FROM fee_invoices WHERE student_id IN (SELECT student_id FROM linked_students) AND status != 'paid'), 0) AS pending_fees,
			(SELECT COUNT(*) FROM fee_invoices WHERE student_id IN (SELECT student_id FROM linked_students) AND status != 'paid') AS pending_invoices,
			(SELECT COUNT(*) FROM homeworks WHERE school_id = ? AND student_id IN (SELECT student_id FROM linked_students) AND status != 'completed') AS homework_open,
			(SELECT COUNT(*) FROM messages JOIN message_conversations ON message_conversations.id = messages.conversation_id WHERE message_conversations.school_id = ? AND message_conversations.parent_id = ? AND messages.sender_role = 'teacher' AND messages.is_read = false) AS unread_messages
	`, schoolID, userID, schoolID, schoolID, userID).Scan(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to load parent dashboard")
		return
	}
	var children []parentChildSummary
	if err := database.DB.Raw(`
		SELECT
			students.id,
			students.first_name,
			students.last_name,
			students.admission_number,
			students.current_section_id,
			COALESCE(SUM(CASE WHEN fee_invoices.status != 'paid' THEN fee_invoices.balance ELSE 0 END), 0) AS pending_fee_balance
		FROM parent_student_links
		JOIN students ON students.id = parent_student_links.student_id
		LEFT JOIN fee_invoices ON fee_invoices.student_id = students.id
		WHERE parent_student_links.school_id = ? AND parent_student_links.parent_user_id = ?
		GROUP BY students.id, students.first_name, students.last_name, students.admission_number, students.current_section_id
		ORDER BY students.first_name, students.last_name
	`, schoolID, userID).Scan(&children).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to load child summaries")
		return
	}
	success(c, http.StatusOK, gin.H{
		"role": "Parent",
		"metrics": gin.H{
			"linked_children":     row.LinkedChildren,
			"pending_fee_balance": row.PendingFees,
			"pending_invoices":    row.PendingInvoices,
			"open_homework":       row.HomeworkOpen,
			"unread_messages":     row.UnreadMessages,
		},
		"attendance": buildAttendanceSummary(row.AttendancePresent, row.AttendanceMarked, 0),
		"children":   children,
	}, "")
}

func buildFeeSummary(total, paid, balance float64, pendingCount, paidCount int64) feeSummary {
	pct := 0.0
	if total > 0 {
		pct = paid * 100 / total
	}
	return feeSummary{
		TotalInvoiced: total,
		TotalPaid:     paid,
		TotalBalance:  balance,
		PendingCount:  pendingCount,
		PaidCount:     paidCount,
		CollectionPct: pct,
	}
}

func buildAttendanceSummary(present, marked, sessions int64) gin.H {
	pct := 0.0
	if marked > 0 {
		pct = float64(present) * 100 / float64(marked)
	}
	return gin.H{
		"sessions":       sessions,
		"present":        present,
		"marked":         marked,
		"attendance_pct": pct,
	}
}

func dashboardStaffID(c *gin.Context) string {
	if !strings.EqualFold(strings.TrimSpace(c.GetString("linked_type")), "staff") {
		return ""
	}
	if linkedID := strings.TrimSpace(c.GetString("linked_id")); linkedID != "" {
		return linkedID
	}
	var user models.User
	if err := database.DB.First(&user, "id = ? AND school_id = ?", c.GetString("user_id"), scopedSchoolID(c)).Error; err != nil {
		return ""
	}
	if user.LinkedID != nil && strings.TrimSpace(*user.LinkedID) != "" {
		return strings.TrimSpace(*user.LinkedID)
	}
	var staff models.Staff
	if err := database.DB.First(&staff, "school_id = ? AND email = ?", scopedSchoolID(c), c.GetString("email")).Error; err != nil {
		return ""
	}
	return staff.ID
}

func dayRange(t time.Time) (time.Time, time.Time) {
	utc := t.UTC()
	start := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 0, 1)
}
