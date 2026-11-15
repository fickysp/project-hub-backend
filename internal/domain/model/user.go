package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primary key" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"type:varchar(255);not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	OAuthAccounts []OAuthAccount  `gorm:"foreignKey:UserID;references:ID" json:"oauth_accounts,omitempty"`
	Projects      []Project       `gorm:"foreignKey:CreatedBy;references:ID" json:"projects,omitempty"`
	Memberships   []ProjectMember `gorm:"foreignKey:UserID;references:ID" json:"memberships,omitempty"`
	Applications  []Application   `gorm:"foreignKey:ApplicantID;references:ID" json:"applications,omitempty"`
	Bookmarks     []Bookmark      `gorm:"foreignKey:UserID;references:ID" json:"bookmarks,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
