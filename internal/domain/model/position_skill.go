package model

import "github.com/google/uuid"

type PositionSkill struct {
	PositionID uuid.UUID `gorm:"type:uuid;not null;primaryKey" json:"position_id"`
	SkillID    uuid.UUID `gorm:"type:uuid;not null;primaryKey;index" json:"skill_id"`
}
