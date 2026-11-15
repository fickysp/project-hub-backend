package model

import "github.com/google/uuid"

type ProjectSkill struct {
	ProjectID uuid.UUID `gorm:"type:uuid;not null;primaryKey" json:"project_id"`
	SkillID   uuid.UUID `gorm:"type:uuid;not null;primaryKey;index" json:"skill_id"`
}
