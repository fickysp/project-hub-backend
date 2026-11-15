package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectPosition struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProjectID   uuid.UUID      `gorm:"type:uuid;not null;index:idx_project_status;uniqueIndex:idx_project_position_id" json:"project_id"`
	Name        string         `gorm:"type:varchar(100);not null" json:"name"`
	Description string         `gorm:"type:text;not null" json:"description"`
	Capacity    int            `gorm:"not null" json:"capacity"`
	Status      PositionStatus `gorm:"type:position_status;not null;default:'open';index:idx_project_status" json:"status"`
	CreatedAt   time.Time      `gorm:"default:now();not null" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"default:now();not null" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Project Project `gorm:"foreignKey:ProjectID;references:ID" json:"project,omitempty"`

	Applications []Application   `gorm:"foreignKey:ProjectID,PositionID;references:ProjectID,ID;constraint:OnDelete:RESTRICT" json:"applications,omitempty"`
	Members      []ProjectMember `gorm:"foreignKey:ProjectID,PositionID;references:ProjectID,ID;constraint:OnDelete:RESTRICT" json:"members,omitempty"`

	Skills []Skill `gorm:"many2many:position_skills;foreignKey:ID;joinForeignKey:PositionID;References:ID;joinReferences:SkillID;constraint:OnDelete:CASCADE" json:"skills,omitempty"`
}

func (p *ProjectPosition) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
