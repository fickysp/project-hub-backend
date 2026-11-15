package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Project struct {
	ID          	uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CreatedBy   	uuid.UUID      `gorm:"type:uuid;not null;index" json:"created_by"`
	Name        	string         `gorm:"type:varchar(150);not null" json:"name"`
	Summary     	string         `gorm:"type:varchar(255);not null" json:"summary"`
	Description 	string         `gorm:"type:text;not null" json:"description"`
	Goal        	string         `gorm:"type:text;not null" json:"goal"`
	Status      	ProjectStatus  `gorm:"type:project_status;not null;default:'recruiting';index" json:"status"`
	AttachmentUrl 	*string        `gorm:"type:varchar(255)" json:"attachment_url,omitempty"`
	CreatedAt   	time.Time      `gorm:"default:now();not null" json:"created_at"`
	UpdatedAt   	time.Time      `gorm:"default:now();not null" json:"updated_at"`
	DeletedAt   	gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Creator      User              `gorm:"foreignKey:CreatedBy;references:ID" json:"creator,omitempty"`
	Positions    []ProjectPosition `gorm:"foreignKey:ProjectID;references:ID;constraint:OnDelete:CASCADE" json:"positions,omitempty"`
	Members      []ProjectMember   `gorm:"foreignKey:ProjectID;references:ID;constraint:OnDelete:CASCADE" json:"members,omitempty"`
	Applications []Application     `gorm:"foreignKey:ProjectID;references:ID;constraint:OnDelete:CASCADE" json:"applications,omitempty"`

	Skills []Skill `gorm:"many2many:project_skills;foreignKey:ID;joinForeignKey:ProjectID;References:ID;joinReferences:SkillID;constraint:OnDelete:CASCADE" json:"skills,omitempty"`
}

func (p *Project) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
