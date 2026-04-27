package models

type UserResponse struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	SchoolID   string `json:"school_id"`
	RoleID     string `json:"role_id"`
	RoleName   string `json:"role_name"`
	IsActive   bool   `json:"is_active"`
	IsVerified bool   `json:"is_verified"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    int64        `json:"expires_at"`
	User         UserResponse `json:"user"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	SchoolID string `json:"school_id" binding:"required"`
	RoleID   string `json:"role_id"`
}

type APIResponse struct {
	Success   bool        `json:"success"`
	Code      string      `json:"code,omitempty"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	Total      int64       `json:"total"`
	TotalPages int         `json:"total_pages"`
}

type CreateSchoolRequest struct {
	Name             string `json:"name" binding:"required"`
	SchoolType       string `json:"school_type" binding:"required"`
	AffiliationBoard string `json:"affiliation_board"`
	Email            string `json:"email"`
	Phone            string `json:"phone"`
	City             string `json:"city"`
	State            string `json:"state"`
	Timezone         string `json:"timezone"`
	Currency         string `json:"currency"`
}

type CreateAcademicYearRequest struct {
	SchoolID  string `json:"school_id" binding:"required"`
	YearLabel string `json:"year_label" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
	IsCurrent bool   `json:"is_current"`
}

type CreateStaffRequest struct {
	SchoolID       string  `json:"school_id" binding:"required"`
	StaffCode      string  `json:"staff_code"`
	FirstName      string  `json:"first_name" binding:"required"`
	LastName       string  `json:"last_name" binding:"required"`
	Email          string  `json:"email"`
	Phone          string  `json:"phone"`
	DateOfBirth    string  `json:"date_of_birth"`
	Gender         string  `json:"gender"`
	DepartmentID   string  `json:"department_id"`
	Designation    string  `json:"designation"`
	EmploymentType string  `json:"employment_type"`
	JoinDate       string  `json:"join_date"`
	BasicSalary    float64 `json:"basic_salary"`
}

type CreateStudentRequest struct {
	SchoolID         string `json:"school_id" binding:"required"`
	StudentCode      string `json:"student_code"`
	AdmissionNumber  string `json:"admission_number"`
	FirstName        string `json:"first_name" binding:"required"`
	LastName         string `json:"last_name" binding:"required"`
	DateOfBirth      string `json:"date_of_birth" binding:"required"`
	Gender           string `json:"gender" binding:"required"`
	AdmissionDate    string `json:"admission_date"`
	CurrentSectionID string `json:"current_section_id"`
}

type CreateGradeRequest struct {
	SchoolID    string `json:"school_id" binding:"required"`
	GradeNumber int    `json:"grade_number" binding:"required"`
	GradeName   string `json:"grade_name" binding:"required"`
}

type CreateSectionRequest struct {
	GradeID        string `json:"grade_id" binding:"required"`
	AcademicYearID string `json:"academic_year_id" binding:"required"`
	SectionName    string `json:"section_name" binding:"required"`
	ClassTeacherID string `json:"class_teacher_id"`
	RoomID         string `json:"room_id"`
	Capacity       int    `json:"capacity"`
}

type CreateSubjectRequest struct {
	SchoolID     string  `json:"school_id" binding:"required"`
	DepartmentID string  `json:"department_id" binding:"required"`
	SubjectName  string  `json:"subject_name" binding:"required"`
	SubjectCode  string  `json:"subject_code"`
	SubjectType  string  `json:"subject_type"`
	CreditHours  float64 `json:"credit_hours"`
}

type CreateEnrollmentRequest struct {
	StudentID      string `json:"student_id" binding:"required"`
	SectionID      string `json:"section_id" binding:"required"`
	AcademicYearID string `json:"academic_year_id" binding:"required"`
	RollNumber     string `json:"roll_number"`
	EnrollmentDate string `json:"enrollment_date"`
}

type CreateFeeStructureRequest struct {
	SchoolID       string  `json:"school_id" binding:"required"`
	AcademicYearID string  `json:"academic_year_id" binding:"required"`
	GradeID        string  `json:"grade_id" binding:"required"`
	FeeCategoryID  string  `json:"fee_category_id" binding:"required"`
	Amount         float64 `json:"amount" binding:"required"`
	DueDay         int     `json:"due_day"`
	LateFinePerDay float64 `json:"late_fine_per_day"`
}

type CreateLeaveApplicationRequest struct {
	StaffID     string `json:"staff_id" binding:"required"`
	LeaveTypeID string `json:"leave_type_id" binding:"required"`
	FromDate    string `json:"from_date" binding:"required"`
	ToDate      string `json:"to_date" binding:"required"`
	HalfDay     bool   `json:"half_day"`
	Reason      string `json:"reason"`
}

type CreateTimetableSlotRequest struct {
	SectionID      string `json:"section_id" binding:"required"`
	AcademicYearID string `json:"academic_year_id" binding:"required"`
	TermID         string `json:"term_id" binding:"required"`
	DayOfWeek      int    `json:"day_of_week" binding:"required"`
	PeriodNumber   int    `json:"period_number" binding:"required"`
	SubjectID      string `json:"subject_id" binding:"required"`
	StaffID        string `json:"staff_id" binding:"required"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
	RoomID         string `json:"room_id"`
}

type CreateExamRequest struct {
	SchoolID       string `json:"school_id" binding:"required"`
	AcademicYearID string `json:"academic_year_id" binding:"required"`
	TermID         string `json:"term_id" binding:"required"`
	ExamTypeID     string `json:"exam_type_id" binding:"required"`
	ExamName       string `json:"exam_name" binding:"required"`
	StartDate      string `json:"start_date" binding:"required"`
	EndDate        string `json:"end_date" binding:"required"`
}
