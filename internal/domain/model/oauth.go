package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OAuthAccount struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Provider       string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_provider_user" json:"provider"`
	ProviderUserID string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_provider_user" json:"provider_user_id"`
	CreatedAt      time.Time `gorm:"default:now();not null" json:"created_at"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

func (o *OAuthAccount) BeforeCreate(tx *gorm.DB) (err error) {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}
