package models

import (
	"time"
	"gorm.io/gorm"
)

type ContractPayment struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	ContractID uint `gorm:"not null" json:"contract_id"`

	Name        string    `gorm:"type:varchar(255)" json:"name"` // Contoh: "Pembayaran Tahap 1"
	Amount      float64   `gorm:"not null" json:"amount"`        // Nilai Realisasi
	PaidAt      time.Time `json:"paid_at"`                       // Tanggal Bayar
	Description string    `gorm:"type:text" json:"description"`

	// Relations
	Contract Contract `json:"-"`
	
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}