// Pivot table untuk assign user ke project dengan custom rate
package models

import (
	"time"
)

type ProjectMember struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProjectID     uint      `gorm:"not null" json:"project_id"`
	Project       Project   `gorm:"foreignKey:ProjectID" json:"-"`
	UserID        uint      `gorm:"not null" json:"user_id"`
	User          User      `gorm:"foreignKey:UserID" json:"user"` // Include user detail saat fetch
	RoleInProject string    `gorm:"type:varchar(100)" json:"role_in_project"`
	CustomRate    *int64    `gorm:"type:bigint" json:"custom_rate,omitempty"` // Override rate kontrak
	JoinedAt      time.Time `json:"joined_at"`
}

func (ProjectMember) TableName() string {
	return "project_members"
}