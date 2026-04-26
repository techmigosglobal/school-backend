package models

import (
	"time"
)

type Announcement struct {
	BaseModel
	SchoolID        string    `gorm:"type:text;not null" json:"school_id"`
	Title           string    `gorm:"type:text;not null" json:"title"`
	Content         string    `gorm:"type:text" json:"content"`
	TargetAudience  string    `gorm:"type:text" json:"target_audience"`
	TargetGradeID   *string   `gorm:"type:text" json:"target_grade_id"`
	TargetSectionID *string   `gorm:"type:text" json:"target_section_id"`
	IsUrgent        bool      `gorm:"default:false" json:"is_urgent"`
	CreatedBy       string    `gorm:"type:text;not null" json:"created_by"`
	PublishedAt     time.Time `json:"published_at"`
	ExpiresAt       *time.Time `json:"expires_at"`
	AttachmentURL   string    `gorm:"type:text" json:"attachment_url"`
	School          *School   `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	TargetGrade     *Grade    `gorm:"foreignKey:TargetGradeID" json:"target_grade,omitempty"`
	TargetSection   *Section  `gorm:"foreignKey:TargetSectionID" json:"target_section,omitempty"`
	Creator         *Staff    `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

type EventCalendar struct {
	BaseModel
	SchoolID        string    `gorm:"type:text;not null" json:"school_id"`
	AcademicYearID  string    `gorm:"type:text;not null" json:"academic_year_id"`
	EventTitle      string    `gorm:"type:text;not null" json:"event_title"`
	EventType       string    `gorm:"type:text;not null" json:"event_type"`
	StartDatetime   time.Time `json:"start_datetime"`
	EndDatetime     time.Time `json:"end_datetime"`
	Location        string    `gorm:"type:text" json:"location"`
	IsHoliday       bool      `gorm:"default:false" json:"is_holiday"`
	CreatedBy       string    `gorm:"type:text;not null" json:"created_by"`
	School          *School   `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	AcademicYear    *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
	Creator         *Staff    `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	PTMSlots        []ParentTeacherMeeting `gorm:"foreignKey:EventID" json:"ptm_slots,omitempty"`
}

type ParentTeacherMeeting struct {
	BaseModel
	EventID     string    `gorm:"type:text;not null" json:"event_id"`
	SectionID   string    `gorm:"type:text;not null" json:"section_id"`
	SlotDate    time.Time `json:"slot_date"`
	SlotTime    string    `gorm:"type:text" json:"slot_time"`
	DurationMin int       `json:"duration_min"`
	TeacherID   string    `gorm:"type:text;not null" json:"teacher_id"`
	GuardianID  string    `gorm:"type:text;not null" json:"guardian_id"`
	StudentID   string    `gorm:"type:text;not null" json:"student_id"`
	Status      string    `gorm:"type:text;default:'scheduled'" json:"status"`
	Notes       string    `gorm:"type:text" json:"notes"`
	Event       *EventCalendar `gorm:"foreignKey:EventID" json:"event,omitempty"`
	Section     *Section  `gorm:"foreignKey:SectionID" json:"section,omitempty"`
	Teacher     *Staff    `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
	Guardian    *Guardian `gorm:"foreignKey:GuardianID" json:"guardian,omitempty"`
	Student     *Student  `gorm:"foreignKey:StudentID" json:"student,omitempty"`
}

type NotificationLog struct {
	BaseModel
	SchoolID        string    `gorm:"type:text;not null" json:"school_id"`
	RecipientUserID string    `gorm:"type:text;not null" json:"recipient_user_id"`
	Channel         string    `gorm:"type:text;not null" json:"channel"`
	Title           string    `gorm:"type:text" json:"title"`
	Body            string    `gorm:"type:text" json:"body"`
	ReferenceType   string    `gorm:"type:text" json:"reference_type"`
	ReferenceID     *string   `gorm:"type:text" json:"reference_id"`
	IsRead          bool      `gorm:"default:false" json:"is_read"`
	SentAt          time.Time `json:"sent_at"`
	DeliveryStatus  string    `gorm:"type:text;default:'pending'" json:"delivery_status"`
	School          *School   `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
}

type LeaveType struct {
	BaseModel
	SchoolID        string   `gorm:"type:text;not null" json:"school_id"`
	LeaveName       string   `gorm:"type:text;not null" json:"leave_name"`
	MaxDaysPerYear  int      `json:"max_days_per_year"`
	CarryForwardDays int     `json:"carry_forward_days"`
	IsPaid          bool     `gorm:"default:false" json:"is_paid"`
	ApplicableTo    string   `gorm:"type:text" json:"applicable_to"`
	School          *School  `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Balances        []LeaveBalance `gorm:"foreignKey:LeaveTypeID" json:"balances,omitempty"`
	Applications    []LeaveApplication `gorm:"foreignKey:LeaveTypeID" json:"applications,omitempty"`
}

type LeaveBalance struct {
	BaseModel
	StaffID        string     `gorm:"type:text;not null" json:"staff_id"`
	LeaveTypeID    string     `gorm:"type:text;not null" json:"leave_type_id"`
	AcademicYearID string     `gorm:"type:text;not null" json:"academic_year_id"`
	TotalEntitled  int        `json:"total_entitled"`
	UsedDays       float64    `json:"used_days"`
	PendingDays    float64    `json:"pending_days"`
	RemainingDays  float64    `json:"remaining_days"`
	Staff          *Staff     `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
	LeaveType      *LeaveType `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
	AcademicYear   *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
}

type LeaveApplication struct {
	BaseModel
	StaffID        string      `gorm:"type:text;not null" json:"staff_id"`
	LeaveTypeID    string      `gorm:"type:text;not null" json:"leave_type_id"`
	FromDate       time.Time   `json:"from_date"`
	ToDate         time.Time   `json:"to_date"`
	HalfDay        bool        `gorm:"default:false" json:"half_day"`
	TotalDays      float64     `json:"total_days"`
	Reason         string      `gorm:"type:text" json:"reason"`
	Status         string      `gorm:"type:text;default:'pending'" json:"status"`
	AppliedAt      time.Time   `json:"applied_at"`
	ApprovedBy     *string     `gorm:"type:text" json:"approved_by"`
	RejectionReason string     `gorm:"type:text" json:"rejection_reason"`
	Staff          *Staff      `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
	LeaveType      *LeaveType  `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
	Approver       *Staff      `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
}

type Payroll struct {
	BaseModel
	StaffID        string    `gorm:"type:text;not null" json:"staff_id"`
	AcademicYearID string    `gorm:"type:text;not null" json:"academic_year_id"`
	Month          int       `gorm:"not null" json:"month"`
	Year           int       `gorm:"not null" json:"year"`
	BasicSalary    float64   `json:"basic_salary"`
	HRA            float64   `json:"hra"`
	DA             float64   `json:"da"`
	GrossSalary    float64   `json:"gross_salary"`
	PFDeduction    float64   `json:"pf_deduction"`
	ESIDeduction   float64   `json:"esi_deduction"`
	TDSDeduction   float64   `json:"tds_deduction"`
	NetSalary      float64   `json:"net_salary"`
	PaymentDate    time.Time `json:"payment_date"`
	PaymentMode    string    `gorm:"type:text" json:"payment_mode"`
	Status         string    `gorm:"type:text;default:'pending'" json:"status"`
	PayslipURL     string    `gorm:"type:text" json:"payslip_url"`
	Staff          *Staff    `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
	AcademicYear   *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
}