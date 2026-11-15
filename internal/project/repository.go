package project

import (
	"context"
	"errors"
	"time"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/fickysp/project-hub-backend.git/pkg/sentinel"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(
		ctx context.Context,
		userID uuid.UUID,
		request CreateProjectRequest,
	) (*model.Project, error)

	Update(
		ctx context.Context,
		projectID uuid.UUID,
		userID uuid.UUID,
		request UpdateProjectRequest,
	) (*model.Project, error)

	Delete(
		ctx context.Context,
		projectID uuid.UUID,
		userID uuid.UUID,
	) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(
	ctx context.Context,
	userID uuid.UUID,
	request CreateProjectRequest,
) (*model.Project, error) {
	project := model.Project{
		CreatedBy:   userID,
		Name:        request.Name,
		Summary:     request.Summary,
		Description: request.Description,
		Goal:        request.Goal,
		Status:      model.ProjectStatusRecruiting,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).
			Create(&project).Error; err != nil {
			return err
		}

		member := model.ProjectMember{
			ProjectID: project.ID,
			UserID:    userID,
			Role:      model.ProjectRoleLead,
			Status:    model.MembershipStatusActive,
			JoinedAt:  time.Now(),
		}

		if err := tx.Omit(clause.Associations).
			Create(&member).Error; err != nil {
			return err
		}

		if err := replaceProjectSkills(
			tx,
			project.ID,
			request.SkillIDs,
		); err != nil {
			return err
		}

		return tx.Preload("Skills").
			Where("id = ?", project.ID).
			Take(&project).Error
	})

	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (r *repository) Update(
	ctx context.Context,
	projectID uuid.UUID,
	userID uuid.UUID,
	request UpdateProjectRequest,
) (*model.Project, error) {
	var project model.Project

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockProjectLead(
			tx,
			projectID,
			userID,
			&project,
		); err != nil {
			return err
		}

		if request.Status != nil &&
			*request.Status == model.ProjectStatusCompleted {
			var count int64

			err := tx.Model(&model.Application{}).
				Where(
					"project_id = ? AND status IN ?",
					projectID,
					[]model.ApplicationStatus{
						model.AppStatusSubmitted,
						model.AppStatusReviewing,
					},
				).
				Count(&count).Error

			if err != nil {
				return err
			}

			if count > 0 {
				return sentinel.ErrPendingApplications
			}
		}

		updates := map[string]interface{}{
			"updated_at": time.Now(),
		}

		if request.Name != nil {
			updates["name"] = *request.Name
		}

		if request.Summary != nil {
			updates["summary"] = *request.Summary
		}

		if request.Description != nil {
			updates["description"] = *request.Description
		}

		if request.Goal != nil {
			updates["goal"] = *request.Goal
		}

		if request.Status != nil {
			updates["status"] = *request.Status
		}

		if err := tx.Model(&model.Project{}).
			Where("id = ?", projectID).
			Updates(updates).Error; err != nil {
			return err
		}

		if request.SkillIDs != nil {
			if err := replaceProjectSkills(
				tx,
				projectID,
				*request.SkillIDs,
			); err != nil {
				return err
			}
		}

		return tx.Preload("Skills").
			Where("id = ?", projectID).
			Take(&project).Error
	})

	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (r *repository) Delete(
	ctx context.Context,
	projectID uuid.UUID,
	userID uuid.UUID,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project model.Project

		if err := lockProjectLead(
			tx,
			projectID,
			userID,
			&project,
		); err != nil {
			return err
		}

		if err := tx.Where("project_id = ?", projectID).
			Delete(&model.ProjectPosition{}).Error; err != nil {
			return err
		}

		now := time.Now()

		if err := tx.Model(&model.ProjectMember{}).
			Where(
				"project_id = ? AND status = ?",
				projectID,
				model.MembershipStatusActive,
			).
			Updates(map[string]interface{}{
				"status":     model.MembershipStatusRemoved,
				"ended_at":   now,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		return tx.Delete(&project).Error
	})
}

func lockProjectLead(
	tx *gorm.DB,
	projectID uuid.UUID,
	userID uuid.UUID,
	project *model.Project,
) error {
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", projectID).
		Take(project).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return sentinel.ErrProjectNotFound
	}

	if err != nil {
		return err
	}

	var member model.ProjectMember

	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"project_id = ? AND user_id = ? AND status = ?",
			projectID,
			userID,
			model.MembershipStatusActive,
		).
		Take(&member).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return sentinel.ErrForbidden
	}

	if err != nil {
		return err
	}

	if member.Role != model.ProjectRoleLead {
		return sentinel.ErrForbidden
	}

	return nil
}

func replaceProjectSkills(
	tx *gorm.DB,
	projectID uuid.UUID,
	skillIDs []uuid.UUID,
) error {
	uniqueIDs := make([]uuid.UUID, 0, len(skillIDs))
	seen := make(map[uuid.UUID]bool)

	for _, id := range skillIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	if len(uniqueIDs) > 0 {
		var skills []model.Skill

		err := tx.Clauses(clause.Locking{Strength: "KEY SHARE"}).
			Where("id IN ?", uniqueIDs).
			Find(&skills).Error

		if err != nil {
			return err
		}

		if len(skills) != len(uniqueIDs) {
			return sentinel.ErrInvalidSkills
		}
	}

	if err := tx.Where("project_id = ?", projectID).
		Delete(&model.ProjectSkill{}).Error; err != nil {
		return err
	}

	if len(uniqueIDs) == 0 {
		return nil
	}

	projectSkills := make([]model.ProjectSkill, 0, len(uniqueIDs))

	for _, skillID := range uniqueIDs {
		projectSkills = append(projectSkills, model.ProjectSkill{
			ProjectID: projectID,
			SkillID:   skillID,
		})
	}

	return tx.Create(&projectSkills).Error
}