// Mencatat Timesheet
package models

import (
	"time"

	"gorm.io/gorm"
)

type TimesheetStatus string

const (
	TimesheetPending  TimesheetStatus = "pending"
	TimesheetApproved TimesheetStatus = "approved"
	TimesheetRejected TimesheetStatus = "rejected"
)

type Timesheet struct {
	ID                  uint            `gorm:"primaryKey" json:"id"`
	UserID              uint            `gorm:"not null" json:"user_id"`
	User                User            `gorm:"foreignKey:UserID" json:"user"`
	ProjectID           uint            `gorm:"not null" json:"project_id"`
	Project             Project         `gorm:"foreignKey:ProjectID" json:"project"`

	TaskID    *uint `gorm:"index" json:"task_id"` // Pointer agar bisa null
	Task      *Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	
	ClockIn             time.Time       `gorm:"not null" json:"clock_in"`
	ClockOut            *time.Time      `json:"clock_out,omitempty"`
	DurationMinutes     int             `json:"duration_minutes"` 
	DurationSeconds     int             `json:"duration_seconds"` // Baru: durasi dalam detik
	Description         string          `gorm:"type:text" json:"description"` // Unified description field
	
	// Security 
	FaceSimilarityScore float64         `json:"face_similarity_score"`
	
	// Approval
	Status              TimesheetStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	ApprovedByID        *uint           `json:"approved_by_id,omitempty"`
	ApprovedBy          *User           `gorm:"foreignKey:ApprovedByID" json:"approved_by,omitempty"`
	RejectionNote       string          `gorm:"type:text" json:"rejection_note,omitempty"`

	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	DeletedAt           gorm.DeletedAt  `gorm:"index" json:"-"`
}

func (Timesheet) TableName() string {
	return "timesheets"
}