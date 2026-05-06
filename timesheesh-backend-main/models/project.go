package models

import (
	"time"

	"gorm.io/gorm"
)

// Enum Status Project
type ProjectStatus string

const (
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusOnHold    ProjectStatus = "on_hold"
)

type Project struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	Name        string        `gorm:"not null" json:"name"`
	ClientName  string        `gorm:"not null;type:varchar(255)" json:"client_name"`
	ClientEmail *string       `gorm:"type:varchar(255)" json:"client_email,omitempty"`
	
	// Status Project
	Status      ProjectStatus `gorm:"type:varchar(50);default:'active'" json:"status"`
	
	// REVENUE: Nilai Kontrak dengan Client (Pemasukan)
	// Digunakan untuk laporan Profit & Loss.
	BudgetRevenue *float64 `gorm:"type:decimal(15,2)" json:"budget_revenue"`

	// COST: Pagu Anggaran Total (Modal/Pengeluaran)
	// Ini adalah batas maksimal uang yang boleh keluar untuk proyek ini.
	// Realisasi Cost = (Total Gaji via Contract) + (Expenses via ResourceRequest).
	BudgetCost          *float64 `gorm:"type:decimal(15,2)" json:"budget_cost"`
	
	// Threshold: Batas peringatan (misal: budget sisa 20% kasih warning)
	BudgetCostThreshold *float64 `gorm:"type:decimal(15,2)" json:"budget_cost_threshold"`

	// TIME: Pagu Anggaran Waktu (Jam)
	BudgetedHours     *int64 `gorm:"default:0" json:"budgeted_hours"`
	HourThreshold     *int64 `gorm:"default:0" json:"hour_threshold"`

	// === RELATIONS ===
	Members          []ProjectMember   `json:"members,omitempty"`
	ResourceRequests []ResourceRequest `json:"resource_requests,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Project) TableName() string {
	return "projects"
}