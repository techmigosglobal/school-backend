package models

import (
	"time"
)

type BookCategory struct {
	BaseModel
	SchoolID     string  `gorm:"type:text;not null" json:"school_id"`
	CategoryName string  `gorm:"type:text;not null" json:"category_name"`
	Description  string  `gorm:"type:text" json:"description"`
	School       *School `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Books        []Book  `gorm:"foreignKey:CategoryID" json:"books,omitempty"`
}

type Book struct {
	BaseModel
	SchoolID        string        `gorm:"type:text;not null" json:"school_id"`
	CategoryID      string        `gorm:"type:text;not null" json:"category_id"`
	ISBN            string        `gorm:"type:text" json:"isbn"`
	Title           string        `gorm:"type:text;not null" json:"title"`
	Author          string        `gorm:"type:text" json:"author"`
	Publisher       string        `gorm:"type:text" json:"publisher"`
	Edition         string        `gorm:"type:text" json:"edition"`
	PublicationYear int           `json:"publication_year"`
	Language        string        `gorm:"type:text" json:"language"`
	TotalCopies     int           `json:"total_copies"`
	AvailableCopies int           `json:"available_copies"`
	Price           float64       `json:"price"`
	LocationCode    string        `gorm:"type:text" json:"location_code"`
	School          *School       `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Category        *BookCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Issues          []BookIssue   `gorm:"foreignKey:BookID" json:"issues,omitempty"`
}

type BookIssue struct {
	BaseModel
	BookID            string     `gorm:"type:text;not null" json:"book_id"`
	BorrowerType      string     `gorm:"type:text;not null" json:"borrower_type"`
	BorrowerID        string     `gorm:"type:text;not null" json:"borrower_id"`
	IssuedDate        time.Time  `json:"issued_date"`
	DueDate           time.Time  `json:"due_date"`
	ReturnDate        *time.Time `json:"return_date"`
	FinePerDay        float64    `json:"fine_per_day"`
	FineAmount        float64    `json:"fine_amount"`
	FinePaid          bool       `gorm:"default:false" json:"fine_paid"`
	ConditionOnReturn string     `gorm:"type:text" json:"condition_on_return"`
	IssuedBy          *string    `gorm:"type:text" json:"issued_by"`
	ReturnedTo        *string    `gorm:"type:text" json:"returned_to"`
	Book              *Book      `gorm:"foreignKey:BookID" json:"book,omitempty"`
}

type Vehicle struct {
	BaseModel
	SchoolID        string  `gorm:"type:text;not null" json:"school_id"`
	VehicleNumber   string  `gorm:"type:text;unique;not null" json:"vehicle_number"`
	VehicleType     string  `gorm:"type:text;not null" json:"vehicle_type"`
	Capacity        int     `json:"capacity"`
	MakeModel       string  `gorm:"type:text" json:"make_model"`
	FuelType        string  `gorm:"type:text" json:"fuel_type"`
	FitnessExpiry   string  `gorm:"type:text" json:"fitness_expiry"`
	InsuranceExpiry string  `gorm:"type:text" json:"insurance_expiry"`
	DriverName      string  `gorm:"type:text" json:"driver_name"`
	DriverPhone     string  `gorm:"type:text" json:"driver_phone"`
	GPSDeviceID     string  `gorm:"type:text" json:"gps_device_id"`
	Status          string  `gorm:"type:text;default:'active'" json:"status"`
	School          *School `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Routes          []Route `gorm:"foreignKey:VehicleID" json:"routes,omitempty"`
}

type Route struct {
	BaseModel
	SchoolID           string             `gorm:"type:text;not null" json:"school_id"`
	RouteName          string             `gorm:"type:text;not null" json:"route_name"`
	RouteCode          string             `gorm:"type:text" json:"route_code"`
	VehicleID          *string            `gorm:"type:text" json:"vehicle_id"`
	TotalDistanceKm    float64            `json:"total_distance_km"`
	MorningStartTime   string             `gorm:"type:text" json:"morning_start_time"`
	AfternoonStartTime string             `gorm:"type:text" json:"afternoon_start_time"`
	School             *School            `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Vehicle            *Vehicle           `gorm:"foreignKey:VehicleID" json:"vehicle,omitempty"`
	Stops              []RouteStop        `gorm:"foreignKey:RouteID" json:"stops,omitempty"`
	StudentTransports  []StudentTransport `gorm:"foreignKey:RouteID" json:"student_transports,omitempty"`
}

type RouteStop struct {
	BaseModel
	RouteID            string             `gorm:"type:text;not null" json:"route_id"`
	StopName           string             `gorm:"type:text;not null" json:"stop_name"`
	SequenceNumber     int                `json:"sequence_number"`
	PickupTime         string             `gorm:"type:text" json:"pickup_time"`
	DropTime           string             `gorm:"type:text" json:"drop_time"`
	Landmark           string             `gorm:"type:text" json:"landmark"`
	Latitude           float64            `json:"latitude"`
	Longitude          float64            `json:"longitude"`
	Route              *Route             `gorm:"foreignKey:RouteID" json:"route,omitempty"`
	StudentTransports  []StudentTransport `gorm:"foreignKey:StopID" json:"student_transports,omitempty"`
}

type StudentTransport struct {
	BaseModel
	StudentID      string        `gorm:"type:text;not null" json:"student_id"`
	AcademicYearID string        `gorm:"type:text;not null" json:"academic_year_id"`
	RouteID        string        `gorm:"type:text;not null" json:"route_id"`
	StopID         string        `gorm:"type:text;not null" json:"stop_id"`
	Direction      string        `gorm:"type:text" json:"transport_direction"`
	FeeAmount      float64       `json:"fee_amount"`
	IsActive       bool          `gorm:"default:true" json:"is_active"`
	Student        *Student      `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	AcademicYear   *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
	Route          *Route        `gorm:"foreignKey:RouteID" json:"route,omitempty"`
	Stop           *RouteStop    `gorm:"foreignKey:StopID" json:"stop,omitempty"`
}
