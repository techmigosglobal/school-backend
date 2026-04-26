package models

import (
	"time"
)

type FeeCategory struct {
	BaseModel
	SchoolID       string           `gorm:"type:uuid;not null" json:"school_id"`
	CategoryName   string           `gorm:"size:255;not null" json:"category_name"`
	Frequency      string           `gorm:"type:enum('monthly','quarterly','half_yearly','yearly','one_time');not null" json:"frequency"`
	IsRefundable   bool             `gorm:"default:false" json:"is_refundable"`
	School         *School          `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Structures     []FeeStructure   `gorm:"foreignKey:FeeCategoryID" json:"structures,omitempty"`
	Concessions    []FeeConcession  `gorm:"foreignKey:FeeCategoryID" json:"concessions,omitempty"`
	InvoiceItems   []FeeInvoiceItem `gorm:"foreignKey:FeeCategoryID" json:"invoice_items,omitempty"`
}

type FeeStructure struct {
	BaseModel
	SchoolID         string       `gorm:"type:uuid;not null" json:"school_id"`
	AcademicYearID   string       `gorm:"type:uuid;not null" json:"academic_year_id"`
	GradeID          string       `gorm:"type:uuid;not null" json:"grade_id"`
	FeeCategoryID    string       `gorm:"type:uuid;not null" json:"fee_category_id"`
	Amount           float64      `json:"amount"`
	DueDay           int          `json:"due_day"`
	LateFinePerDay   float64      `json:"late_fine_per_day"`
	School           *School      `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	AcademicYear     *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
	Grade            *Grade       `gorm:"foreignKey:GradeID" json:"grade,omitempty"`
	FeeCategory      *FeeCategory `gorm:"foreignKey:FeeCategoryID" json:"fee_category,omitempty"`
}

type FeeConcession struct {
	BaseModel
	StudentID      string       `gorm:"type:uuid;not null" json:"student_id"`
	FeeCategoryID  string       `gorm:"type:uuid;not null" json:"fee_category_id"`
	AcademicYearID string       `gorm:"type:uuid;not null" json:"academic_year_id"`
	ConcessionType string       `gorm:"type:enum('scholarship','merit','financial_aid','sibling','other');not null" json:"concession_type"`
	Value          float64      `json:"value"`
	Reason         string       `gorm:"type:text" json:"reason"`
	ApprovedBy     *string      `gorm:"type:uuid" json:"approved_by"`
	Student        *Student     `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	FeeCategory    *FeeCategory `gorm:"foreignKey:FeeCategoryID" json:"fee_category,omitempty"`
	AcademicYear   *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
}

type FeeInvoice struct {
	BaseModel
	StudentID      string           `gorm:"type:uuid;not null" json:"student_id"`
	AcademicYearID string           `gorm:"type:uuid;not null" json:"academic_year_id"`
	InvoiceNumber  string           `gorm:"size:100;unique" json:"invoice_number"`
	InvoiceDate    time.Time        `json:"invoice_date"`
	DueDate        time.Time        `json:"due_date"`
	TotalAmount    float64          `json:"total_amount"`
	DiscountAmount float64          `json:"discount_amount"`
	NetAmount      float64          `json:"net_amount"`
	PaidAmount     float64          `json:"paid_amount"`
	Balance        float64          `json:"balance"`
	Status         string           `gorm:"type:enum('pending','partial','paid','overdue','cancelled');default:'pending'" json:"status"`
	Student        *Student         `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	AcademicYear   *AcademicYear    `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
	Items          []FeeInvoiceItem `gorm:"foreignKey:InvoiceID" json:"items,omitempty"`
	Payments       []Payment        `gorm:"foreignKey:InvoiceID" json:"payments,omitempty"`
}

type FeeInvoiceItem struct {
	BaseModel
	InvoiceID     string       `gorm:"type:uuid;not null" json:"invoice_id"`
	FeeCategoryID string       `gorm:"type:uuid;not null" json:"fee_category_id"`
	Amount        float64      `json:"amount"`
	Description   string       `gorm:"size:255" json:"description"`
	Invoice       *FeeInvoice  `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
	FeeCategory   *FeeCategory `gorm:"foreignKey:FeeCategoryID" json:"fee_category,omitempty"`
}

type Payment struct {
	BaseModel
	InvoiceID       string     `gorm:"type:uuid;not null" json:"invoice_id"`
	ReceiptNumber   string     `gorm:"size:100;unique" json:"receipt_number"`
	AmountPaid      float64    `json:"amount_paid"`
	PaymentDate     time.Time  `json:"payment_date"`
	PaymentMode     string     `gorm:"type:enum('cash','cheque','dd','online','upi','card','bank_transfer');not null" json:"payment_mode"`
	TransactionID   string     `gorm:"size:255" json:"transaction_id"`
	ReceivedBy      *string    `gorm:"type:uuid" json:"received_by"`
	CreatedAt       time.Time  `json:"created_at"`
	Invoice         *FeeInvoice `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
}