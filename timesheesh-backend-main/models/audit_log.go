// Untuk mencatat jejak aktivitas sistem
package models

import (
	"time"
)

type AuditAction string

const (
	AuditCreate  AuditAction = "CREATE"
	AuditUpdate  AuditAction = "UPDATE"
	AuditDelete  AuditAction = "DELETE"
	AuditLogin   AuditAction = "LOGIN"
	AuditProxy   AuditAction = "LOGIN_AS_PROXY"
)

type AuditLog struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	UserID    uint        `json:"user_id"` // Bisa null jika system action, tapi biasanya user
	User      User        `gorm:"foreignKey:UserID" json:"-"`
	Action    AuditAction `gorm:"type:varchar(50);not null" json:"action"`
	TargetTable string      `gorm:"type:varchar(100)" json:"target_table"`
	RecordID  uint        `json:"record_id"`
	
	// Simpan perubahan data dalam bentuk JSON String/Text
	OldValue  string      `gorm:"type:text" json:"old_value,omitempty"` 
	NewValue  string      `gorm:"type:text" json:"new_value,omitempty"`
	
	CreatedAt time.Time   `json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}