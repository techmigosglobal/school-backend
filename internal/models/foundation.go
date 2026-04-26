package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        string    `gorm:"type:text;primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

type School struct {
	BaseModel
	Name             string `gorm:"type:text;not null" json:"name"`
	SchoolType       string `gorm:"type:text;not null" json:"school_type"`
	AffiliationBoard string `gorm:"type:text" json:"affiliation_board"`
	Email            string `gorm:"type:text" json:"email"`
	Phone            string `gorm:"type:text" json:"phone"`
	City             string `gorm:"type:text" json:"city"`
	State            string `gorm:"type:text" json:"state"`
	Timezone         string `gorm:"type:text" json:"timezone"`
	Currency         string `gorm:"type:text" json:"currency"`
}

type AcademicYear struct {
	BaseModel
	SchoolID   string     `gorm:"type:text;not null" json:"school_id"`
	YearLabel  string     `gorm:"type:text;not null" json:"year_label"`
	StartDate  time.Time  `json:"start_date"`
	EndDate    time.Time  `json:"end_date"`
	IsCurrent  bool       `gorm:"default:false" json:"is_current"`
	Status     string     `gorm:"type:text;default:'upcoming'" json:"status"`
	School     *School    `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Terms      []Term     `gorm:"foreignKey:AcademicYearID" json:"terms,omitempty"`
	Holidays   []Holiday  `gorm:"foreignKey:AcademicYearID" json:"holidays,omitempty"`
}

type Term struct {
	BaseModel
	AcademicYearID string       `gorm:"type:text;not null" json:"academic_year_id"`
	TermNumber     int          `gorm:"not null" json:"term_number"`
	TermName       string       `gorm:"type:text;not null" json:"term_name"`
	StartDate      time.Time    `json:"start_date"`
	EndDate        time.Time    `json:"end_date"`
	IsCurrent      bool         `gorm:"default:false" json:"is_current"`
	AcademicYear   *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
}

type Holiday struct {
	BaseModel
	SchoolID       string        `gorm:"type:text;not null" json:"school_id"`
	AcademicYearID string        `gorm:"type:text;not null" json:"academic_year_id"`
	HolidayName    string        `gorm:"type:text;not null" json:"holiday_name"`
	FromDate       time.Time     `json:"from_date"`
	ToDate         time.Time     `json:"to_date"`
	Type           string        `gorm:"type:text" json:"type"`
	School         *School       `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	AcademicYear   *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
}

type WorkingDayConfig struct {
	BaseModel
	SchoolID           string    `gorm:"type:text;not null" json:"school_id"`
	DayOfWeek          int       `gorm:"not null" json:"day_of_week"`
	IsWorking          bool      `gorm:"default:true" json:"is_working"`
	PeriodsPerDay      int       `json:"periods_per_day"`
	PeriodDurationMin  int       `json:"period_duration_min"`
	SchoolStartTime    string    `gorm:"type:text" json:"school_start_time"`
	SchoolEndTime      string    `gorm:"type:text" json:"school_end_time"`
	School             *School   `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
}

type Department struct {
	BaseModel
	SchoolID        string   `gorm:"type:text;not null" json:"school_id"`
	DepartmentName  string   `gorm:"type:text;not null" json:"department_name"`
	HODStaffID      *string  `gorm:"type:text" json:"hod_staff_id"`
	Description     string   `gorm:"type:text" json:"description"`
	School          *School  `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	HODStaff        *Staff   `gorm:"foreignKey:HODStaffID" json:"hod_staff,omitempty"`
	Subjects        []Subject `gorm:"foreignKey:DepartmentID" json:"subjects,omitempty"`
}

type Subject struct {
	BaseModel
	SchoolID       string      `gorm:"type:text;not null" json:"school_id"`
	DepartmentID   string      `gorm:"type:text;not null" json:"department_id"`
	SubjectName    string      `gorm:"type:text;not null" json:"subject_name"`
	SubjectCode    string      `gorm:"type:text" json:"subject_code"`
	SubjectType    string      `gorm:"type:text" json:"subject_type"`
	CreditHours    float64     `json:"credit_hours"`
	School         *School     `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Department     *Department `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
}

type Grade struct {
	BaseModel
	SchoolID      string          `gorm:"type:text;not null" json:"school_id"`
	GradeNumber   int             `gorm:"not null" json:"grade_number"`
	GradeName     string          `gorm:"type:text;not null" json:"grade_name"`
	School        *School         `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	GradeSubjects []GradeSubject  `gorm:"foreignKey:GradeID" json:"grade_subjects,omitempty"`
	Sections      []Section       `gorm:"foreignKey:GradeID" json:"sections,omitempty"`
}

type GradeSubject struct {
	BaseModel
	GradeID          string   `gorm:"type:text;not null" json:"grade_id"`
	SubjectID        string   `gorm:"type:text;not null" json:"subject_id"`
	PeriodsPerWeek   int      `json:"periods_per_week"`
	MaxMarks         int      `json:"max_marks"`
	PassMarks        int      `json:"pass_marks"`
	IsMandatory      bool     `gorm:"default:true" json:"is_mandatory"`
	Grade            *Grade   `gorm:"foreignKey:GradeID" json:"grade,omitempty"`
	Subject          *Subject `gorm:"foreignKey:SubjectID" json:"subject,omitempty"`
}

type Section struct {
	BaseModel
	GradeID         string     `gorm:"type:text;not null" json:"grade_id"`
	AcademicYearID  string     `gorm:"type:text;not null" json:"academic_year_id"`
	SectionName     string     `gorm:"type:text;not null" json:"section_name"`
	ClassTeacherID  *string    `gorm:"type:text" json:"class_teacher_id"`
	RoomID          *string    `gorm:"type:text" json:"room_id"`
	Capacity        int        `json:"capacity"`
	Grade           *Grade     `gorm:"foreignKey:GradeID" json:"grade,omitempty"`
	AcademicYear    *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
	ClassTeacher    *Staff     `gorm:"foreignKey:ClassTeacherID" json:"class_teacher,omitempty"`
	Room            *Room      `gorm:"foreignKey:RoomID" json:"room,omitempty"`
}

type Room struct {
	BaseModel
	SchoolID   string   `gorm:"type:text;not null" json:"school_id"`
	RoomNumber string   `gorm:"type:text;not null" json:"room_number"`
	RoomType   string   `gorm:"type:text" json:"room_type"`
	Block      string   `gorm:"type:text" json:"block"`
	Floor      int      `json:"floor"`
	Capacity   int      `json:"capacity"`
	School     *School  `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
}