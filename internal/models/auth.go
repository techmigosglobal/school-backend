package models

import (
	"time"
)

type Role struct {
	BaseModel
	SchoolID      string       `gorm:"type:text;not null" json:"school_id"`
	RoleName      string       `gorm:"type:text;not null" json:"role_name"`
	Description   string       `gorm:"type:text" json:"description"`
	IsSystemRole  bool         `gorm:"default:false" json:"is_system_role"`
	School        *School      `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Permissions   []Permission `gorm:"foreignKey:RoleID" json:"permissions,omitempty"`
	Users         []User       `gorm:"foreignKey:RoleID" json:"users,omitempty"`
}

type Permission struct {
	BaseModel
	RoleID     string `gorm:"type:text;not null" json:"role_id"`
	Module     string `gorm:"type:text;not null" json:"module"`
	CanRead    bool   `gorm:"default:false" json:"can_read"`
	CanCreate  bool   `gorm:"default:false" json:"can_create"`
	CanUpdate  bool   `gorm:"default:false" json:"can_update"`
	CanDelete  bool   `gorm:"default:false" json:"can_delete"`
	CanExport  bool   `gorm:"default:false" json:"can_export"`
	Role       *Role  `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

type User struct {
	BaseModel
	SchoolID       string     `gorm:"type:text;not null" json:"school_id"`
	Email          string     `gorm:"type:text;not null" json:"email"`
	Phone          string     `gorm:"type:text" json:"phone"`
	PasswordHash   string     `gorm:"type:text;not null" json:"-"`
	RoleID         string     `gorm:"type:text;not null" json:"role_id"`
	LinkedType     string     `gorm:"type:text" json:"linked_type"`
	LinkedID       *string    `gorm:"type:text" json:"linked_id"`
	IsActive       bool       `gorm:"default:true" json:"is_active"`
	IsVerified     bool       `gorm:"default:false" json:"is_verified"`
	LastLogin      *time.Time `json:"last_login"`
	FailedAttempts int        `gorm:"default:0" json:"failed_attempts"`
	LockedUntil    *time.Time `json:"locked_until"`
	CreatedAt      time.Time  `json:"created_at"`
	School         *School    `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Role           *Role      `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	Sessions       []UserSession `gorm:"foreignKey:UserID" json:"sessions,omitempty"`
	AuditLogs      []AuditLog `gorm:"foreignKey:UserID" json:"audit_logs,omitempty"`
}

type UserSession struct {
	BaseModel
	UserID           string    `gorm:"type:text;not null" json:"user_id"`
	RefreshTokenHash string    `gorm:"type:text" json:"-"`
	DeviceInfo       string    `gorm:"type:text" json:"device_info"`
	IPAddress        string    `gorm:"type:text" json:"ip_address"`
	UserAgent        string    `gorm:"type:text" json:"user_agent"`
	IssuedAt         time.Time `json:"issued_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	IsRevoked        bool      `gorm:"default:false" json:"is_revoked"`
	User             *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type OTPVerification struct {
	BaseModel
	Identifier  string     `gorm:"type:text;not null" json:"identifier"`
	OTPHash     string     `gorm:"type:text;not null" json:"-"`
	Purpose     string     `gorm:"type:text;not null" json:"purpose"`
	ExpiresAt   time.Time  `json:"expires_at"`
	IsUsed      bool       `gorm:"default:false" json:"is_used"`
	Attempts    int        `gorm:"default:0" json:"attempts"`
}

type AuditLog struct {
	BaseModel
	UserID     string    `gorm:"type:text;not null" json:"user_id"`
	Action     string    `gorm:"type:text;not null" json:"action"`
	Module     string    `gorm:"type:text" json:"module"`
	TableName  string    `gorm:"type:text" json:"table_name"`
	RecordID   *string   `gorm:"type:text" json:"record_id"`
	OldValue   string    `gorm:"type:text" json:"old_value"`
	NewValue   string    `gorm:"type:text" json:"new_value"`
	IPAddress  string    `gorm:"type:text" json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
	User       *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}