package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/fickysp/project-hub-backend.git/pkg/sentinel"
	"gorm.io/gorm"
)

type AuthRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindById(ctx context.Context, id string) (*model.User, error)
	UpdateRefreshToken(ctx context.Context, userID string, refreshToken string) error
	FindByRefreshToken(ctx context.Context, refreshToken string) (*model.User, error)
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return err
	}
	return nil
}

func (r *authRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) FindById(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) UpdateRefreshToken(ctx context.Context, userID string, refreshToken string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Update("refresh_token", refreshToken).Error
}

func (r *authRepository) FindByRefreshToken(ctx context.Context, refreshToken string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("refresh_token = ?", refreshToken).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) Logout(ctx context.Context, userID string) error {
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Update("refresh_token", nil)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return sentinel.ErrUserNotFound
	}
	return nil
}
