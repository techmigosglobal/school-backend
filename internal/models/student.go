package models

import (
	"time"
)

type Student struct {
	BaseModel
	SchoolID          string            `gorm:"type:uuid;not null" json:"school_id"`
	StudentCode       string            `gorm:"size:50;unique" json:"student_code"`
	AdmissionNumber   string            `gorm:"size:50;unique" json:"admission_number"`
	FirstName         string            `gorm:"size:100;not null" json:"first_name"`
	LastName          string            `gorm:"size:100;not null" json:"last_name"`
	DateOfBirth       time.Time         `json:"date_of_birth"`
	Gender            string            `gorm:"type:text" json:"gender"`
	CasteCategory     string            `gorm:"type:text" json:"caste_category"`
	Nationality       string            `gorm:"size:100" json:"nationality"`
	AdmissionDate     time.Time         `json:"admission_date"`
	CurrentSectionID  *string           `gorm:"type:uuid" json:"current_section_id"`
	AadharNumber      string            `gorm:"type:text" json:"aadhar_number"`
	Status            string            `gorm:"type:text;default:'active'" json:"status"`
	School            *School           `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	CurrentSection    *Section          `gorm:"foreignKey:CurrentSectionID" json:"current_section,omitempty"`
	Guardians         []Guardian        `gorm:"foreignKey:StudentID" json:"guardians,omitempty"`
	MedicalRecord     *MedicalRecord    `gorm:"foreignKey:StudentID" json:"medical_record,omitempty"`
	Documents         []StudentDocument `gorm:"foreignKey:StudentID" json:"documents,omitempty"`
	Enrollments       []Enrollment      `gorm:"foreignKey:StudentID" json:"enrollments,omitempty"`
}

type Guardian struct {
	BaseModel
	StudentID     string    `gorm:"type:uuid;not null" json:"student_id"`
	FullName      string    `gorm:"size:255;not null" json:"full_name"`
	Relationship  string    `gorm:"type:text" json:"relationship"`
	Phone         string    `gorm:"size:50" json:"phone"`
	Email         string    `gorm:"size:255" json:"email"`
	Occupation    string    `gorm:"size:100" json:"occupation"`
	AnnualIncome  float64   `json:"annual_income"`
	IsPrimary     bool      `gorm:"default:false" json:"is_primary"`
	CanPickup     bool      `gorm:"default:false" json:"can_pickup"`
	Student       *Student  `gorm:"foreignKey:StudentID" json:"student,omitempty"`
}

type MedicalRecord struct {
	BaseModel
	StudentID    string    `gorm:"type:uuid;not null" json:"student_id"`
	Conditions   string    `gorm:"type:text" json:"conditions"`
	Allergies    string    `gorm:"type:text" json:"allergies"`
	Medications  string    `gorm:"type:text" json:"medications"`
	DoctorName   string    `gorm:"size:255" json:"doctor_name"`
	DoctorPhone  string    `gorm:"size:50" json:"doctor_phone"`
	UpdatedAt    time.Time `json:"updated_at"`
	Student      *Student  `gorm:"foreignKey:StudentID" json:"student,omitempty"`
}

type StudentDocument struct {
	BaseModel
	StudentID   string     `gorm:"type:uuid;not null" json:"student_id"`
	DocType     string     `gorm:"type:text" json:"doc_type"`
	FileURL     string     `gorm:"type:text" json:"file_url"`
	Verified    bool       `gorm:"default:false" json:"verified"`
	UploadedAt  time.Time  `json:"uploaded_at"`
	Student     *Student   `gorm:"foreignKey:StudentID" json:"student,omitempty"`
}

type Enrollment struct {
	BaseModel
	StudentID       string         `gorm:"type:uuid;not null" json:"student_id"`
	SectionID       string         `gorm:"type:uuid;not null" json:"section_id"`
	AcademicYearID  string         `gorm:"type:uuid;not null" json:"academic_year_id"`
	RollNumber      string         `gorm:"size:50" json:"roll_number"`
	EnrollmentDate  time.Time      `json:"enrollment_date"`
	Status          string         `gorm:"type:text;default:'enrolled'" json:"status"`
	PromotedFromID  *string        `gorm:"type:uuid" json:"promoted_from_id"`
	Student         *Student       `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Section         *Section       `gorm:"foreignKey:SectionID" json:"section,omitempty"`
	AcademicYear    *AcademicYear  `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
	PromotedFrom    *Enrollment    `gorm:"foreignKey:PromotedFromID" json:"promoted_from,omitempty"`
}

type TransferRecord struct {
	BaseModel
	StudentID           string    `gorm:"type:uuid;not null" json:"student_id"`
	TransferType        string    `gorm:"type:text;not null" json:"transfer_type"`
	FromSchool          string    `gorm:"size:255" json:"from_school"`
	ToSchool            string    `gorm:"size:255" json:"to_school"`
	TransferDate        time.Time `json:"transfer_date"`
	TransferCertNumber  string    `gorm:"size:100" json:"transfer_cert_number"`
	LastGradeID         *string   `gorm:"type:uuid" json:"last_grade_id"`
	Reason              string    `gorm:"type:text" json:"reason"`
	IssuedBy            *string   `gorm:"type:uuid" json:"issued_by"`
	Student             *Student  `gorm:"foreignKey:StudentID" json:"student,omitempty"`
}

type PromotionRule struct {
	BaseModel
	SchoolID           string        `gorm:"type:uuid;not null" json:"school_id"`
	AcademicYearID     string        `gorm:"type:uuid;not null" json:"academic_year_id"`
	FromGradeID        string        `gorm:"type:uuid;not null" json:"from_grade_id"`
	ToGradeID          string        `gorm:"type:uuid;not null" json:"to_grade_id"`
	MinAttendancePct   float64       `json:"min_attendance_pct"`
	MinPassPercentage  float64       `json:"min_pass_percentage"`
	SubjectsMustPass   int           `json:"subjects_must_pass"`
	School             *School       `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	AcademicYear       *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
	FromGrade          *Grade        `gorm:"foreignKey:FromGradeID" json:"from_grade,omitempty"`
	ToGrade            *Grade        `gorm:"foreignKey:ToGradeID" json:"to_grade,omitempty"`
}