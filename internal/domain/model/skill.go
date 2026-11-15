package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Skill struct {
	ID   uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
}

func (a *Skill) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
