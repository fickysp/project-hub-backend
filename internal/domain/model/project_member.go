package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectMember struct {
	ID         uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProjectID  uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex:idx_project_user;index:idx_proj_stat_role;index:idx_proj_pos_stat" json:"project_id"`
	UserID     uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex:idx_project_user;index:idx_user_status" json:"user_id"`
	PositionID *uuid.UUID       `gorm:"type:uuid;index:idx_proj_pos_stat" json:"position_id,omitempty"`
	Role       ProjectRole      `gorm:"type:project_role;not null;default:'member';index:idx_proj_stat_role" json:"role"`
	Status     MembershipStatus `gorm:"type:membership_status;not null;default:'active';index:idx_proj_stat_role;index:idx_proj_pos_stat;index:idx_user_status" json:"status"`
	JoinedAt   time.Time        `gorm:"default:now();not null" json:"joined_at"`
	EndedAt    *time.Time       `json:"ended_at,omitempty"`
	CreatedAt  time.Time        `gorm:"default:now();not null" json:"created_at"`
	UpdatedAt  time.Time        `gorm:"default:now();not null" json:"updated_at"`

	Project Project `gorm:"foreignKey:ProjectID;references:ID" json:"project,omitempty"`
	User    User    `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`

	Position *ProjectPosition `gorm:"foreignKey:ProjectID,PositionID;references:ProjectID,ID;constraint:OnDelete:RESTRICT" json:"position,omitempty"`
}

func (p *ProjectMember) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
