package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Application struct {
	ID             uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProjectID      uuid.UUID         `gorm:"type:uuid;not null;index:idx_proj_status;index:idx_proj_pos_status" json:"project_id"`
	PositionID     uuid.UUID         `gorm:"type:uuid;not null;index:idx_proj_pos_status" json:"position_id"`
	ApplicantID    uuid.UUID         `gorm:"type:uuid;not null;index:idx_applicant_created" json:"applicant_id"`
	RelevantSkills string            `gorm:"type:text;not null" json:"relevant_skills"`
	Motivation     string            `gorm:"type:text;not null" json:"motivation"`
	Status         ApplicationStatus `gorm:"type:application_status;not null;default:'submitted';index:idx_proj_status;index:idx_proj_pos_status" json:"status"`
	SelectionNote  *string           `gorm:"type:text" json:"selection_note,omitempty"`
	PortfolioUrl   *string           `gorm:"type:varchar(255)" json:"portfolio_url,omitempty"`
	DecidedBy      *uuid.UUID        `gorm:"type:uuid" json:"decided_by,omitempty"`
	DecidedAt      *time.Time        `json:"decided_at,omitempty"`
	CreatedAt      time.Time         `gorm:"default:now();not null;index:idx_applicant_created" json:"created_at"`
	UpdatedAt      time.Time         `gorm:"default:now();not null" json:"updated_at"`

	Project   Project         `gorm:"foreignKey:ProjectID;references:ID" json:"project,omitempty"`
	Applicant User            `gorm:"foreignKey:ApplicantID;references:ID" json:"applicant,omitempty"`
	Decider   *User           `gorm:"foreignKey:DecidedBy;references:ID" json:"decider,omitempty"`
	Position  ProjectPosition `gorm:"foreignKey:ProjectID,PositionID;references:ProjectID,ID;constraint:OnDelete:RESTRICT" json:"position,omitempty"`
}

func (a *Application) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
