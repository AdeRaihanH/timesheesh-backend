package models

import (
	"time"

	"gorm.io/gorm"
)

// Role enum untuk tipe user dalam project timesheet
type Role string

const (
	RoleAdmin          Role = "admin"          // 1. Admin - akses penuh ke sistem
	RoleProjectManager Role = "projectmanager" // 2. Project Manager - mengelola proyek
	RoleEmployee       Role = "employee"       // 3. Employee - karyawan yang mengisi timesheet
	RoleFinance        Role = "finance"        // 4. Finance/Management - mengelola keuangan dan laporan
)

// EmployeeType enum
type EmployeeType string

const (
	EmployeeTypeFulltime  EmployeeType = "fulltime"
	EmployeeTypeParttime  EmployeeType = "parttime"
	EmployeeTypeFreelance EmployeeType = "freelance"
)

// User model
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	Password     string         `gorm:"not null" json:"-"` // Hidden from JSON
	FullName     string         `gorm:"not null" json:"full_name"`
	Role         Role           `gorm:"type:varchar(50);not null" json:"role"`
	EmployeeType *EmployeeType  `gorm:"type:varchar(50)" json:"employee_type,omitempty"` // Nullable, only for employee role
	
	// New
	FaceEmbedding string         `gorm:"type:text" json:"-"` // Simpan vektor wajah (hash/string)
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	
	// Relations (Optional, for preloading)
	Contracts     []Contract     `json:"contracts,omitempty"`
	Timesheets    []Timesheet    `json:"timesheets,omitempty"`

	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for User model
func (User) TableName() string {
	return "users"
}

// IsEmployee checks if user is an employee
func (u *User) IsEmployee() bool {
	return u.Role == RoleEmployee
}

// RequiresEmployeeType checks if role requires employee type
func (u *User) RequiresEmployeeType() bool {
	return u.Role == RoleEmployee
}
