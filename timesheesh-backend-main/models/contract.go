package models

import (
	"time"

	"gorm.io/gorm"
)

type ContractType string

const (
	ContractTypeYearly    ContractType = "yearly"
	ContractTypeMonthly   ContractType = "monthly"
	ContractTypeMandays   ContractType = "mandays"   // Harian
	ContractTypeTimesheet ContractType = "timesheet" // Hourly
)

// Enum Payment Scheme (BARU)
type PaymentScheme string

const (
	SchemeMonthly    PaymentScheme = "monthly"      // Rutin tiap bulan
	SchemeTermin     PaymentScheme = "termin"       // Berdasarkan progress project
	SchemeBackToBack PaymentScheme = "back_to_back" // Setelah client bayar
)

// Enum Payment Status (Global)
type PaymentStatus string

const (
	StatusPending       PaymentStatus = "pending"
	StatusPaid          PaymentStatus = "paid"
	StatusPartiallyPaid PaymentStatus = "partially_paid"
)

type Contract struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"not null" json:"user_id"`

	// Project Specific Contract
	// Jika NULL: Contract Global (Default).
	// Jika Terisi: Contract Temporary khusus project ini.
	ProjectID *uint `json:"project_id,omitempty"`

	ContractType  ContractType  `gorm:"type:varchar(50);not null" json:"contract_type"`
	PaymentScheme PaymentScheme `gorm:"type:varchar(50);not null" json:"payment_scheme"`

	TotalPaid     float64       `gorm:"default:0" json:"total_paid"`     // Akumulasi Pembayaran
	PaymentStatus PaymentStatus `gorm:"type:varchar(50);default:'pending'" json:"payment_status"`

	RateAmount int64      `gorm:"not null" json:"rate_amount"` // Gaji atau Rate
	StartDate  time.Time  `gorm:"type:date;not null" json:"start_date"`
	EndDate    *time.Time `gorm:"type:date" json:"end_date"`
	IsActive   bool       `gorm:"default:true" json:"is_active"`

	// Relations
	User    User     `json:"-"`
	Project *Project `json:"-"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

	// Tambahan