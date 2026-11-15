package member

import (
	"context"
	"errors"
	"time"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Leave(
		ctx context.Context,
		projectID uuid.UUID,
		userID uuid.UUID,
	) (*model.ProjectMember, error)

	Remove(
		ctx context.Context,
		projectID uuid.UUID,
		memberID uuid.UUID,
		actorID uuid.UUID,
	) (*model.ProjectMember, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Leave(
	ctx context.Context,
	projectID uuid.UUID,
	userID uuid.UUID,
) (*model.ProjectMember, error) {
	return r.endMembership(
		ctx,
		projectID,
		uuid.Nil,
		userID,
		false,
	)
}

func (r *repository) Remove(
	ctx context.Context,
	projectID uuid.UUID,
	memberID uuid.UUID,
	actorID uuid.UUID,
) (*model.ProjectMember, error) {
	return r.endMembership(
		ctx,
		projectID,
		memberID,
		actorID,
		true,
	)
}

func (r *repository) endMembership(
	ctx context.Context,
	projectID uuid.UUID,
	memberID uuid.UUID,
	actorID uuid.UUID,
	remove bool,
) (*model.ProjectMember, error) {
	var result model.ProjectMember

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project model.Project

		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", projectID).
			Take(&project).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProjectNotFound
		}

		if err != nil {
			return err
		}

		var actor model.ProjectMember

		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(
				"project_id = ? AND user_id = ? AND status = ?",
				projectID,
				actorID,
				model.MembershipStatusActive,
			).
			Take(&actor).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrForbidden
		}

		if err != nil {
			return err
		}

		targetStatus := model.MembershipStatusLeft

		if remove {
			if actor.Role != model.ProjectRoleLead {
				return ErrForbidden
			}

			err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where(
					"id = ? AND project_id = ?",
					memberID,
					projectID,
				).
				Take(&result).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrMemberNotFound
			}

			if err != nil {
				return err
			}

			if result.UserID == actorID {
				return ErrUseLeaveEndpoint
			}

			targetStatus = model.MembershipStatusRemoved
		} else {
			result = actor
		}

		if result.Status != model.MembershipStatusActive {
			return ErrMemberInactive
		}

		if result.Role == model.ProjectRoleLead {
			var leadCount int64

			err = tx.Model(&model.ProjectMember{}).
				Where(
					"project_id = ? AND role = ? AND status = ?",
					projectID,
					model.ProjectRoleLead,
					model.MembershipStatusActive,
				).
				Count(&leadCount).Error

			if err != nil {
				return err
			}

			if leadCount <= 1 {
				return ErrLastProjectLead
			}
		}

		now := time.Now()

		update := tx.Model(&model.ProjectMember{}).
			Where(
				"id = ? AND project_id = ? AND status = ?",
				result.ID,
				projectID,
				model.MembershipStatusActive,
			).
			Updates(map[string]interface{}{
				"status":     targetStatus,
				"ended_at":   now,
				"updated_at": now,
			})

		if update.Error != nil {
			return update.Error
		}

		if update.RowsAffected != 1 {
			return ErrMemberInactive
		}

		result.Status = targetStatus
		result.EndedAt = &now
		result.UpdatedAt = now

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}