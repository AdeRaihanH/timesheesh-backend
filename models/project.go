package models

import (
	"time"

	"gorm.io/gorm"
)

// BudgetType enum untuk jenis anggaran
type BudgetType string

const (
	BudgetTypeProject BudgetType = "project_based" // Berdasarkan Proyek
	BudgetTypeUser    BudgetType = "user_based"    // Berdasarkan Pengguna
)

// Project model
type Project struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"not null" json:"name"`
	ClientName  string     `gorm:"not null;type:varchar(255)" json:"client_name"`   // Nama client
	ClientEmail *string    `gorm:"type:varchar(255)" json:"client_email,omitempty"` // Email client
	BudgetType  BudgetType `gorm:"type:varchar(50);not null" json:"budget_type"`

	// Berdasarkan Proyek (Project Based)
	BudgetedHours       *int64 `gorm:"type:bigint" json:"budgeted_hours,omitempty"`        // 1a. jam yang dianggarkan
	HourThreshold       *int64 `gorm:"type:bigint" json:"hour_threshold,omitempty"`        // 1b. ambang batas jam
	BudgetCost          *int64 `gorm:"type:bigint" json:"budget_cost,omitempty"`           // 1c. anggaran biaya (dalam rupiah)
	BudgetCostThreshold *int64 `gorm:"type:bigint" json:"budget_cost_threshold,omitempty"` // 1d. ambang batas anggaran biaya (dalam rupiah)
	BudgetRevenue       *int64 `gorm:"type:bigint" json:"budget_revenue,omitempty"`        // 1e. anggaran pendapatan (dalam rupiah)

	// Berdasarkan Pengguna (User Based) - berdasarkan jam project
	CostPerHour *int64 `gorm:"type:bigint" json:"cost_per_hour,omitempty"` // 2a1. biaya per jam (dalam rupiah)
	RatePerHour *int64 `gorm:"type:bigint" json:"rate_per_hour,omitempty"` // 2a2. tarif per jam (dalam rupiah)

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for Project model
func (Project) TableName() string {
	return "projects"
}

// IsProjectBased checks if project uses project-based budget
func (p *Project) IsProjectBased() bool {
	return p.BudgetType == BudgetTypeProject
}

// IsUserBased checks if project uses user-based budget
func (p *Project) IsUserBased() bool {
	return p.BudgetType == BudgetTypeUser
}
