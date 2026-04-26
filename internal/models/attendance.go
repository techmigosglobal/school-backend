package models

import (
	"time"
)

type TimetableSlot struct {
	BaseModel
	SectionID       string     `gorm:"type:text;not null" json:"section_id"`
	AcademicYearID  string     `gorm:"type:text;not null" json:"academic_year_id"`
	TermID          string     `gorm:"type:text;not null" json:"term_id"`
	DayOfWeek       int        `gorm:"not null" json:"day_of_week"`
	PeriodNumber    int        `gorm:"not null" json:"period_number"`
	StartTime       string     `gorm:"type:text" json:"start_time"`
	EndTime         string     `gorm:"type:text" json:"end_time"`
	SubjectID       string     `gorm:"type:text;not null" json:"subject_id"`
	StaffID         string     `gorm:"type:text;not null" json:"staff_id"`
	RoomID          *string    `gorm:"type:text" json:"room_id"`
	SlotType        string     `gorm:"type:text;default:'regular'" json:"slot_type"`
	Section         *Section   `gorm:"foreignKey:SectionID" json:"section,omitempty"`
	AcademicYear    *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
	Term            *Term      `gorm:"foreignKey:TermID" json:"term,omitempty"`
	Subject         *Subject   `gorm:"foreignKey:SubjectID" json:"subject,omitempty"`
	Staff           *Staff     `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
	Room            *Room      `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	Substitutions   []Substitution `gorm:"foreignKey:TimetableSlotID" json:"substitutions,omitempty"`
}

type Substitution struct {
	BaseModel
	TimetableSlotID  string    `gorm:"type:text;not null" json:"timetable_slot_id"`
	Date             time.Time `json:"date"`
	OriginalStaffID  string    `gorm:"type:text;not null" json:"original_staff_id"`
	SubstituteStaffID string   `gorm:"type:text;not null" json:"substitute_staff_id"`
	Reason           string    `gorm:"type:text" json:"reason"`
	ApprovedBy       *string   `gorm:"type:text" json:"approved_by"`
	CreatedAt        time.Time `json:"created_at"`
	TimetableSlot    *TimetableSlot `gorm:"foreignKey:TimetableSlotID" json:"timetable_slot,omitempty"`
	OriginalStaff    *Staff    `gorm:"foreignKey:OriginalStaffID" json:"original_staff,omitempty"`
	SubstituteStaff  *Staff    `gorm:"foreignKey:SubstituteStaffID" json:"substitute_staff,omitempty"`
}

type AttendanceSession struct {
	BaseModel
	SectionID       string    `gorm:"type:text;not null" json:"section_id"`
	TimetableSlotID *string   `gorm:"type:text" json:"timetable_slot_id"`
	SubjectID       string    `gorm:"type:text;not null" json:"subject_id"`
	StaffID         string    `gorm:"type:text;not null" json:"staff_id"`
	Date            time.Time `json:"date"`
	PeriodNumber    int       `json:"period_number"`
	TotalStudents   int       `json:"total_students"`
	PresentCount    int       `json:"present_count"`
	IsFinalized     bool      `gorm:"default:false" json:"is_finalized"`
	CreatedAt       time.Time `json:"created_at"`
	Section         *Section  `gorm:"foreignKey:SectionID" json:"section,omitempty"`
	Subject         *Subject  `gorm:"foreignKey:SubjectID" json:"subject,omitempty"`
	Staff           *Staff    `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
	StudentAttendances []StudentAttendance `gorm:"foreignKey:SessionID" json:"student_attendances,omitempty"`
}

type StudentAttendance struct {
	BaseModel
	SessionID    string     `gorm:"type:text;not null" json:"session_id"`
	StudentID    string     `gorm:"type:text;not null" json:"student_id"`
	EnrollmentID string     `gorm:"type:text;not null" json:"enrollment_id"`
	Status       string     `gorm:"type:text;not null" json:"status"`
	Reason       string     `gorm:"type:text" json:"reason"`
	MarkedAt     time.Time  `json:"marked_at"`
	MarkedBy     *string    `gorm:"type:text" json:"marked_by"`
	Session      *AttendanceSession `gorm:"foreignKey:SessionID" json:"session,omitempty"`
	Student      *Student   `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Enrollment   *Enrollment `gorm:"foreignKey:EnrollmentID" json:"enrollment,omitempty"`
}

type StaffAttendance struct {
	BaseModel
	StaffID     string     `gorm:"type:text;not null" json:"staff_id"`
	Date        time.Time  `json:"date"`
	CheckIn     *time.Time `json:"check_in"`
	CheckOut    *time.Time `json:"check_out"`
	Status      string     `gorm:"type:text;not null" json:"status"`
	BiometricID string     `gorm:"type:text" json:"biometric_id"`
	ApprovedBy  *string    `gorm:"type:text" json:"approved_by"`
	Staff       *Staff     `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
}

type AttendanceSummary struct {
	BaseModel
	StudentID      string        `gorm:"type:text;not null" json:"student_id"`
	SectionID      string        `gorm:"type:text;not null" json:"section_id"`
	AcademicYearID string        `gorm:"type:text;not null" json:"academic_year_id"`
	TermID         *string       `gorm:"type:text" json:"term_id"`
	TotalDays      int           `json:"total_days"`
	PresentDays    int           `json:"present_days"`
	AbsentDays     int           `json:"absent_days"`
	LateCount      int           `json:"late_count"`
	AttendancePct  float64       `json:"attendance_pct"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Student        *Student      `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Section        *Section      `gorm:"foreignKey:SectionID" json:"section,omitempty"`
	AcademicYear   *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
	Term           *Term         `gorm:"foreignKey:TermID" json:"term,omitempty"`
}