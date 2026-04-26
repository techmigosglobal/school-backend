package models

import (
	"time"
)

type ExamType struct {
	BaseModel
	SchoolID           string   `gorm:"type:text;not null" json:"school_id"`
	Name               string   `gorm:"type:text;not null" json:"name"`
	WeightagePercent   float64  `json:"weightage_percent"`
	IsBoardExam        bool     `gorm:"default:false" json:"is_board_exam"`
	School             *School  `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Exams              []Exam   `gorm:"foreignKey:ExamTypeID" json:"exams,omitempty"`
}

type Exam struct {
	BaseModel
	SchoolID        string         `gorm:"type:text;not null" json:"school_id"`
	AcademicYearID  string         `gorm:"type:text;not null" json:"academic_year_id"`
	TermID          string         `gorm:"type:text;not null" json:"term_id"`
	ExamTypeID      string         `gorm:"type:text;not null" json:"exam_type_id"`
	ExamName        string         `gorm:"type:text;not null" json:"exam_name"`
	StartDate       time.Time      `json:"start_date"`
	EndDate         time.Time      `json:"end_date"`
	IsPublished     bool           `gorm:"default:false" json:"is_published"`
	School          *School        `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	AcademicYear    *AcademicYear  `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
	Term            *Term          `gorm:"foreignKey:TermID" json:"term,omitempty"`
	ExamType        *ExamType      `gorm:"foreignKey:ExamTypeID" json:"exam_type,omitempty"`
	Schedules       []ExamSchedule `gorm:"foreignKey:ExamID" json:"schedules,omitempty"`
	ReportCards     []ReportCard   `gorm:"foreignKey:ExamID" json:"report_cards,omitempty"`
}

type ExamSchedule struct {
	BaseModel
	ExamID      string         `gorm:"type:text;not null" json:"exam_id"`
	GradeID     string         `gorm:"type:text;not null" json:"grade_id"`
	SectionID   string         `gorm:"type:text;not null" json:"section_id"`
	SubjectID   string         `gorm:"type:text;not null" json:"subject_id"`
	ExamDate    time.Time      `json:"exam_date"`
	StartTime   string         `gorm:"type:text" json:"start_time"`
	EndTime     string         `gorm:"type:text" json:"end_time"`
	MaxMarks    int            `json:"max_marks"`
	PassMarks   int            `json:"pass_marks"`
	RoomID      *string        `gorm:"type:text" json:"room_id"`
	Exam        *Exam          `gorm:"foreignKey:ExamID" json:"exam,omitempty"`
	Grade       *Grade         `gorm:"foreignKey:GradeID" json:"grade,omitempty"`
	Section     *Section       `gorm:"foreignKey:SectionID" json:"section,omitempty"`
	Subject     *Subject       `gorm:"foreignKey:SubjectID" json:"subject,omitempty"`
	Room        *Room          `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	StudentMarks []StudentMark  `gorm:"foreignKey:ExamScheduleID" json:"student_marks,omitempty"`
}

type StudentMark struct {
	BaseModel
	ExamScheduleID string        `gorm:"type:text;not null" json:"exam_schedule_id"`
	StudentID      string        `gorm:"type:text;not null" json:"student_id"`
	EnrollmentID   string        `gorm:"type:text;not null" json:"enrollment_id"`
	MarksObtained  float64       `json:"marks_obtained"`
	GradeLabel     string        `gorm:"type:text" json:"grade_label"`
	IsAbsent       bool          `gorm:"default:false" json:"is_absent"`
	IsExempted     bool          `gorm:"default:false" json:"is_exempted"`
	EnteredBy      *string       `gorm:"type:text" json:"entered_by"`
	VerifiedBy     *string       `gorm:"type:text" json:"verified_by"`
	ExamSchedule   *ExamSchedule `gorm:"foreignKey:ExamScheduleID" json:"exam_schedule,omitempty"`
	Student        *Student      `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Enrollment     *Enrollment   `gorm:"foreignKey:EnrollmentID" json:"enrollment,omitempty"`
}

type GradingScale struct {
	BaseModel
	SchoolID    string   `gorm:"type:text;not null" json:"school_id"`
	GradeLabel  string   `gorm:"type:text;not null" json:"grade_label"`
	MinPercent  float64  `json:"min_percent"`
	MaxPercent  float64  `json:"max_percent"`
	GPAPoints   float64  `json:"gpa_points"`
	Description string   `gorm:"type:text" json:"description"`
	School      *School  `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
}

type ReportCard struct {
	BaseModel
	StudentID      string       `gorm:"type:text;not null" json:"student_id"`
	ExamID         string       `gorm:"type:text;not null" json:"exam_id"`
	EnrollmentID   string       `gorm:"type:text;not null" json:"enrollment_id"`
	TotalObtained  float64      `json:"total_obtained"`
	Percentage     float64      `json:"percentage"`
	OverallGrade   string       `gorm:"type:text" json:"overall_grade"`
	OverallGPA     float64      `json:"overall_gpa"`
	ClassRank      int          `json:"class_rank"`
	SectionRank    int          `json:"section_rank"`
	PublishedAt    time.Time    `json:"published_at"`
	Student        *Student     `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Exam           *Exam        `gorm:"foreignKey:ExamID" json:"exam,omitempty"`
	Enrollment     *Enrollment  `gorm:"foreignKey:EnrollmentID" json:"enrollment,omitempty"`
}