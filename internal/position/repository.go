package projectposition

import (
	"context"
	"errors"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PositionRepository interface {
	Create(ctx context.Context, pos *model.ProjectPosition) error
	UpdatePartial(ctx context.Context, id uuid.UUID, updateData map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.ProjectPosition, error)
	GetActiveMembersCount(ctx context.Context, positionID uuid.UUID) (int64, error)
	GetPendingAppsCount(ctx context.Context, positionID uuid.UUID) (int64, error)
	CheckIsProjectLead(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
	FindAllByProjectID(ctx context.Context, projectID uuid.UUID) ([]model.ProjectPosition, error)
}

type positionRepo struct {
	db *gorm.DB
}

func NewPositionRepository(db *gorm.DB) PositionRepository {
	return &positionRepo{db}
}

func (r *positionRepo) Create(ctx context.Context, pos *model.ProjectPosition) error {
	return r.db.WithContext(ctx).Create(pos).Error
}

func (r *positionRepo) UpdatePartial(ctx context.Context, id uuid.UUID, updateData map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.ProjectPosition{}).Where("id = ?", id).Updates(updateData).Error
}

func (r *positionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.ProjectPosition{}, id).Error
}

func (r *positionRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.ProjectPosition, error) {
	var pos model.ProjectPosition
	err := r.db.WithContext(ctx).First(&pos, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("posisi tidak ditemukan")
	}
	return &pos, err
}

func (r *positionRepo) GetActiveMembersCount(ctx context.Context, positionID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ProjectMember{}).
		Where("position_id = ? AND status = 'active'", positionID).Count(&count).Error
	return count, err
}

func (r *positionRepo) GetPendingAppsCount(ctx context.Context, positionID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Application{}).
		Where("position_id = ? AND status IN ('submitted', 'reviewing')", positionID).Count(&count).Error
	return count, err
}

func (r *positionRepo) CheckIsProjectLead(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ? AND role = 'lead'", projectID, userID).Count(&count).Error
	return count > 0, err
}

func (r *positionRepo) FindAllByProjectID(ctx context.Context, projectID uuid.UUID) ([]model.ProjectPosition, error) {
	var positions []model.ProjectPosition
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at ASC").
		Find(&positions).Error
	return positions, err
}
