package application

import (
	"context"
	"errors"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApplicationRepository interface {
	Create(ctx context.Context, app *model.Application) error
	UpdateStatus(ctx context.Context, id uuid.UUID, updateData map[string]interface{}) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Application, error)
	FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]model.Application, error)
	FindByApplicantID(ctx context.Context, applicantID uuid.UUID) ([]model.Application, error)
	FindByProjectAndPosition(ctx context.Context, projectID, positionID uuid.UUID) ([]model.Application, error)
	CheckPositionExists(ctx context.Context, positionID, projectID uuid.UUID) (bool, error)
	CheckHasActiveApplication(ctx context.Context, applicantID, projectID uuid.UUID) (bool, error)
	CheckIsProjectLead(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
	CheckPositionAvailable(ctx context.Context, positionID uuid.UUID) (bool, error)
}

type applicationRepo struct {
	db *gorm.DB
}

func NewApplicationRepository(db *gorm.DB) ApplicationRepository {
	return &applicationRepo{db}
}

func (r *applicationRepo) Create(ctx context.Context, app *model.Application) error {
	return r.db.WithContext(ctx).Create(app).Error
}

func (r *applicationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, updateData map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&model.Application{}).
		Where("id = ?", id).
		Updates(updateData).Error
}

func (r *applicationRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Application, error) {
	var app model.Application
	err := r.db.WithContext(ctx).
		Preload("Position").
		Preload("Applicant").
		First(&app, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("pengajuan tidak ditemukan")
	}
	return &app, err
}

func (r *applicationRepo) FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]model.Application, error) {
	var apps []model.Application
	err := r.db.WithContext(ctx).
		Preload("Position").
		Preload("Applicant").
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Find(&apps).Error
	return apps, err
}

func (r *applicationRepo) FindByApplicantID(ctx context.Context, applicantID uuid.UUID) ([]model.Application, error) {
	var apps []model.Application
	err := r.db.WithContext(ctx).
		Preload("Position").
		Where("applicant_id = ?", applicantID).
		Order("created_at DESC").
		Find(&apps).Error
	return apps, err
}

func (r *applicationRepo) FindByProjectAndPosition(ctx context.Context, projectID, positionID uuid.UUID) ([]model.Application, error) {
	var apps []model.Application
	err := r.db.WithContext(ctx).
		Preload("Applicant").
		Where("project_id = ? AND position_id = ?", projectID, positionID).
		Order("created_at DESC").
		Find(&apps).Error
	return apps, err
}

func (r *applicationRepo) CheckPositionExists(ctx context.Context, positionID, projectID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ProjectPosition{}).
		Where("id = ? AND project_id = ?", positionID, projectID).
		Count(&count).Error
	return count > 0, err
}

func (r *applicationRepo) CheckHasActiveApplication(ctx context.Context, applicantID, projectID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Application{}).
		Where(
			"applicant_id = ? AND project_id = ? AND status IN ('submitted', 'reviewing')",
			applicantID,
			projectID,
		).
		Count(&count).Error
	return count > 0, err
}

func (r *applicationRepo) CheckIsProjectLead(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ProjectMember{}).
		Where(
			"project_id = ? AND user_id = ? AND role = 'lead' AND status = 'active'",
			projectID,
			userID,
		).
		Count(&count).Error
	return count > 0, err
}

func (r *applicationRepo) CheckPositionAvailable(ctx context.Context, positionID uuid.UUID) (bool, error) {
	var pos model.ProjectPosition
	err := r.db.WithContext(ctx).
		Select("capacity, status").
		First(&pos, "id = ?", positionID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, errors.New("posisi tidak ditemukan")
	}
	if err != nil {
		return false, err
	}
	if pos.Status != model.PositionStatusOpen {
		return false, nil
	}

	var activeMembers int64
	err = r.db.WithContext(ctx).
		Model(&model.ProjectMember{}).
		Where("position_id = ? AND status = 'active'", positionID).
		Count(&activeMembers).Error
	if err != nil {
		return false, err
	}

	return int(activeMembers) < pos.Capacity, nil
}
