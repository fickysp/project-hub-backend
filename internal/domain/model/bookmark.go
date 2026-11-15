package model

import (
	"time"

	"github.com/google/uuid"
)

type Bookmark struct {
	UserID    uuid.UUID `gorm:"type:uuid;not null;primaryKey" json:"user_id"`
	ProjectID uuid.UUID `gorm:"type:uuid;not null;primaryKey;index" json:"project_id"`
	CreatedAt time.Time `gorm:"default:now();not null" json:"created_at"`

	User    User    `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Project Project `gorm:"foreignKey:ProjectID;references:ID;constraint:OnDelete:CASCADE" json:"project,omitempty"`
}
