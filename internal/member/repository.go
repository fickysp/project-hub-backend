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
	List(
		ctx context.Context,
		projectID uuid.UUID,
		userID uuid.UUID,
		request ListMembersRequest,
	) ([]model.ProjectMember, int64, error)

	UpdatePosition(
		ctx context.Context,
		projectID uuid.UUID,
		memberID uuid.UUID,
		actorID uuid.UUID,
		positionID uuid.UUID,
	) (*model.ProjectMember, error)

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

func (r *repository) List(
	ctx context.Context,
	projectID uuid.UUID,
	userID uuid.UUID,
	request ListMembersRequest,
) ([]model.ProjectMember, int64, error) {
	members := make([]model.ProjectMember, 0)
	var total int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := lockProjectAccess(
			tx,
			projectID,
			userID,
			"SHARE",
			false,
		); err != nil {
			return err
		}

		query := tx.Model(&model.ProjectMember{}).
			Where(
				"project_id = ? AND status = ?",
				projectID,
				model.MembershipStatusActive,
			)

		if err := query.Session(&gorm.Session{}).
			Count(&total).Error; err != nil {
			return err
		}

		return preloadMember(query).
			Order("joined_at ASC").
			Order("id ASC").
			Offset((request.Page - 1) * request.Limit).
			Limit(request.Limit).
			Find(&members).Error
	})

	if err != nil {
		return nil, 0, err
	}

	return members, total, nil
}

func (r *repository) UpdatePosition(
	ctx context.Context,
	projectID uuid.UUID,
	memberID uuid.UUID,
	actorID uuid.UUID,
	positionID uuid.UUID,
) (*model.ProjectMember, error) {
	var result model.ProjectMember

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := lockProjectAccess(
			tx,
			projectID,
			actorID,
			"UPDATE",
			true,
		); err != nil {
			return err
		}

		target, err := lockActiveMember(tx, projectID, memberID)
		if err != nil {
			return err
		}

		var position model.ProjectPosition

		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(
				"id = ? AND project_id = ?",
				positionID,
				projectID,
			).
			Take(&position).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPositionNotFound
		}

		if err != nil {
			return err
		}

		if target.PositionID == nil || *target.PositionID != positionID {
			var occupied int64

			err = tx.Model(&model.ProjectMember{}).
				Where(
					"project_id = ? AND position_id = ? AND status = ?",
					projectID,
					positionID,
					model.MembershipStatusActive,
				).
				Count(&occupied).Error

			if err != nil {
				return err
			}

			if occupied >= int64(position.Capacity) {
				return ErrPositionFull
			}

			update := tx.Model(&model.ProjectMember{}).
				Where(
					"id = ? AND project_id = ? AND status = ?",
					memberID,
					projectID,
					model.MembershipStatusActive,
				).
				Updates(map[string]interface{}{
					"position_id": positionID,
					"updated_at":  time.Now(),
				})

			if update.Error != nil {
				return update.Error
			}

			if update.RowsAffected != 1 {
				return ErrMemberInactive
			}
		}

		return preloadMember(tx).
			Where("id = ? AND project_id = ?", memberID, projectID).
			Take(&result).Error
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *repository) Leave(
	ctx context.Context,
	projectID uuid.UUID,
	userID uuid.UUID,
) (*model.ProjectMember, error) {
	return r.endMembership(ctx, projectID, uuid.Nil, userID, false)
}

func (r *repository) Remove(
	ctx context.Context,
	projectID uuid.UUID,
	memberID uuid.UUID,
	actorID uuid.UUID,
) (*model.ProjectMember, error) {
	return r.endMembership(ctx, projectID, memberID, actorID, true)
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
		actor, err := lockProjectAccess(
			tx,
			projectID,
			actorID,
			"UPDATE",
			remove,
		)
		if err != nil {
			return err
		}

		targetStatus := model.MembershipStatusLeft

		if remove {
			target, err := lockActiveMember(tx, projectID, memberID)
			if err != nil {
				return err
			}

			if target.UserID == actorID {
				return ErrUseLeaveEndpoint
			}

			result = *target
			targetStatus = model.MembershipStatusRemoved
		} else {
			result = *actor
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

func lockProjectAccess(
	tx *gorm.DB,
	projectID uuid.UUID,
	userID uuid.UUID,
	strength string,
	requireLead bool,
) (*model.ProjectMember, error) {
	var project model.Project

	err := tx.Clauses(clause.Locking{Strength: strength}).
		Where("id = ?", projectID).
		Take(&project).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProjectNotFound
	}

	if err != nil {
		return nil, err
	}

	var actor model.ProjectMember

	err = tx.Clauses(clause.Locking{Strength: strength}).
		Where(
			"project_id = ? AND user_id = ? AND status = ?",
			projectID,
			userID,
			model.MembershipStatusActive,
		).
		Take(&actor).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrForbidden
	}

	if err != nil {
		return nil, err
	}

	if requireLead && actor.Role != model.ProjectRoleLead {
		return nil, ErrForbidden
	}

	return &actor, nil
}

func lockActiveMember(
	tx *gorm.DB,
	projectID uuid.UUID,
	memberID uuid.UUID,
) (*model.ProjectMember, error) {
	var member model.ProjectMember

	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"id = ? AND project_id = ?",
			memberID,
			projectID,
		).
		Take(&member).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMemberNotFound
	}

	if err != nil {
		return nil, err
	}

	if member.Status != model.MembershipStatusActive {
		return nil, ErrMemberInactive
	}

	return &member, nil
}

func preloadMember(query *gorm.DB) *gorm.DB {
	return query.
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name")
		}).
		Preload("Position")
}
