package models

import (
	"time"
)

type Staff struct {
	BaseModel
	SchoolID       string               `gorm:"type:text;not null" json:"school_id"`
	StaffCode      string               `gorm:"type:text;unique" json:"staff_code"`
	FirstName      string               `gorm:"type:text;not null" json:"first_name"`
	LastName       string               `gorm:"type:text;not null" json:"last_name"`
	Email          string               `gorm:"type:text" json:"email"`
	Phone          string               `gorm:"type:text" json:"phone"`
	DateOfBirth    time.Time            `json:"date_of_birth"`
	Gender         string               `gorm:"type:text" json:"gender"`
	DepartmentID   *string              `gorm:"type:text" json:"department_id"`
	Designation    string               `gorm:"type:text" json:"designation"`
	EmploymentType string               `gorm:"type:text" json:"employment_type"`
	JoinDate       time.Time            `json:"join_date"`
	ExitDate       *time.Time           `json:"exit_date"`
	BasicSalary    float64              `json:"basic_salary"`
	Status         string               `gorm:"type:text;default:'active'" json:"status"`
	School         *School              `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Department     *Department          `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
	Qualifications []StaffQualification `gorm:"foreignKey:StaffID" json:"qualifications,omitempty"`
	Subjects       []StaffSubject       `gorm:"foreignKey:StaffID" json:"subjects,omitempty"`
	Documents      []StaffDocument      `gorm:"foreignKey:StaffID" json:"documents,omitempty"`
}

type StaffQualification struct {
	BaseModel
	StaffID       string `gorm:"type:text;not null" json:"staff_id"`
	Degree        string `gorm:"type:text;not null" json:"degree"`
	Institution   string `gorm:"type:text" json:"institution"`
	YearOfPassing int    `json:"year_of_passing"`
	GradeDivision string `gorm:"type:text" json:"grade_division"`
	DocURL        string `gorm:"type:text" json:"doc_url"`
	Staff         *Staff `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
}

type StaffSubject struct {
	BaseModel
	StaffID   string   `gorm:"type:text;not null" json:"staff_id"`
	SubjectID string   `gorm:"type:text;not null" json:"subject_id"`
	GradeID   string   `gorm:"type:text;not null" json:"grade_id"`
	IsPrimary bool     `gorm:"default:false" json:"is_primary"`
	Staff     *Staff   `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
	Subject   *Subject `gorm:"foreignKey:SubjectID" json:"subject,omitempty"`
	Grade     *Grade   `gorm:"foreignKey:GradeID" json:"grade,omitempty"`
}

type StaffDocument struct {
	BaseModel
	StaffID    string     `gorm:"type:text;not null" json:"staff_id"`
	DocType    string     `gorm:"type:text" json:"doc_type"`
	FileURL    string     `gorm:"type:text" json:"file_url"`
	ExpiryDate *time.Time `json:"expiry_date"`
	Verified   bool       `gorm:"default:false" json:"verified"`
	UploadedAt time.Time  `json:"uploaded_at"`
	Staff      *Staff     `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
}
