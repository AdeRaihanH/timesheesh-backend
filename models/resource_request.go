package models

import (
	"time"

	"gorm.io/gorm"
)

// Enum untuk Tipe Pengeluaran/Resource
type ResourceType string

const (
	ResTypeTool          ResourceType = "tools"           // Laptop, Software License, Aset
	ResTypeInfra         ResourceType = "infrastructure"  // Server, Cloud, Domain
	ResTypeAccom         ResourceType = "accommodation"   // Hotel, Transport, Konsumsi
	ResTypeManpower      ResourceType = "manpower"        // Request Penambahan Orang
)

// Enum Status (Sudah ada sebelumnya, pastikan tetap ada)
type RequestStatus string

const (
	RequestStatusPending  RequestStatus = "pending"
	RequestStatusApproved RequestStatus = "approved"
	RequestStatusRejected RequestStatus = "rejected"
)

type ResourceRequest struct {
	ID        uint          `gorm:"primaryKey" json:"id"`
	ProjectID uint          `gorm:"not null" json:"project_id"`
	Project   Project       `gorm:"foreignKey:ProjectID" json:"-"` // Relasi ke Project
	UserID    uint          `gorm:"not null" json:"user_id"`
	User      User          `gorm:"foreignKey:UserID" json:"user"` // Relasi ke User (Requestor)

	// Kategori: tools, infrastructure, accommodation, manpower
	Type      ResourceType  `gorm:"type:varchar(50);not null" json:"type"`
	
	// Detail deskripsi request
	Details   string        `gorm:"type:text" json:"details"`

	// Nominal Biaya (Real Cost)
	// - Jika Tool/Infra/Accom: Diisi harga beli/sewa.
	// - Jika Manpower: Default 0 (karena biaya gaji masuk via Timesheet), 
	//   kecuali ada biaya rekrutmen/agency di awal.
	Amount    float64       `gorm:"type:decimal(15,2);default:0" json:"amount"`

	Status    RequestStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ResourceRequest) TableName() string {
	return "resource_requests"
}