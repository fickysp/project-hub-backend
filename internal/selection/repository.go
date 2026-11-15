package selection

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/fickysp/project-hub-backend.git/pkg/sentinel"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApplicationListResult struct {
	Applications []model.Application
	Total        int64
	Occupied     map[uuid.UUID]int64
}

type Repository interface {
	List(
		ctx context.Context,
		projectID uuid.UUID,
		leadID uuid.UUID,
		request ListApplicationsRequest,
	) (*ApplicationListResult, error)

	Detail(
		ctx context.Context,
		projectID uuid.UUID,
		applicationID uuid.UUID,
		leadID uuid.UUID,
	) (*model.Application, int64, error)

	Summary(
		ctx context.Context,
		projectID uuid.UUID,
		leadID uuid.UUID,
	) (*ApplicationSummaryResponse, error)

	Review(
		ctx context.Context,
		projectID uuid.UUID,
		applicationID uuid.UUID,
		leadID uuid.UUID,
	) (*model.Application, int64, error)

	Accept(
		ctx context.Context,
		projectID uuid.UUID,
		applicationID uuid.UUID,
		leadID uuid.UUID,
		note string,
	) (*model.Application, int64, error)

	Reject(
		ctx context.Context,
		projectID uuid.UUID,
		applicationID uuid.UUID,
		leadID uuid.UUID,
		note string,
	) (*model.Application, int64, error)
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
	leadID uuid.UUID,
	request ListApplicationsRequest,
) (*ApplicationListResult, error) {
	result := &ApplicationListResult{
		Applications: make([]model.Application, 0),
		Occupied:     make(map[uuid.UUID]int64),
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := lockSelectionLead(
			tx,
			projectID,
			leadID,
			"SHARE",
		); err != nil {
			return err
		}

		query := tx.Model(&model.Application{}).
			Where("applications.project_id = ?", projectID)

		if request.Search != "" {
			query = query.Where(`
				EXISTS (
					SELECT 1
					FROM users u
					WHERE u.id = applications.applicant_id
						AND u.name ILIKE ?
				)
			`, "%"+request.Search+"%")
		}

		if request.PositionID != "" {
			query = query.Where(
				"applications.position_id = ?",
				request.PositionID,
			)
		}

		if request.Status != "" {
			query = query.Where(
				"applications.status = ?",
				request.Status,
			)
		}

		if err := query.Session(&gorm.Session{}).
			Count(&result.Total).Error; err != nil {
			return err
		}

		err := preloadApplication(query).
			Order("applications.created_at DESC").
			Order("applications.id DESC").
			Limit(request.Limit).
			Offset((request.Page - 1) * request.Limit).
			Find(&result.Applications).Error

		if err != nil {
			return err
		}

		positionIDs := make([]uuid.UUID, 0, len(result.Applications))

		for _, application := range result.Applications {
			positionIDs = append(positionIDs, application.PositionID)
		}

		counts, err := countPositionMembers(tx, projectID, positionIDs)
		if err != nil {
			return err
		}

		result.Occupied = counts

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repository) Detail(
	ctx context.Context,
	projectID uuid.UUID,
	applicationID uuid.UUID,
	leadID uuid.UUID,
) (*model.Application, int64, error) {
	var application model.Application
	var occupied int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := lockSelectionLead(
			tx,
			projectID,
			leadID,
			"SHARE",
		); err != nil {
			return err
		}

		var err error

		application, occupied, err = loadApplication(
			tx,
			projectID,
			applicationID,
		)

		return err
	})

	if err != nil {
		return nil, 0, err
	}

	return &application, occupied, nil
}

func (r *repository) Summary(
	ctx context.Context,
	projectID uuid.UUID,
	leadID uuid.UUID,
) (*ApplicationSummaryResponse, error) {
	var response ApplicationSummaryResponse

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		project, err := lockSelectionLead(
			tx,
			projectID,
			leadID,
			"SHARE",
		)
		if err != nil {
			return err
		}

		response.ProjectID = project.ID
		response.ProjectName = project.Name
		response.ProjectStatus = project.Status

		var counts struct {
			Total     int64
			Submitted int64
			Reviewing int64
			Accepted  int64
			Rejected  int64
			Cancelled int64
		}

		err = tx.Model(&model.Application{}).
			Select(`
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE status = 'submitted') AS submitted,
				COUNT(*) FILTER (WHERE status = 'reviewing') AS reviewing,
				COUNT(*) FILTER (WHERE status = 'accepted') AS accepted,
				COUNT(*) FILTER (WHERE status = 'rejected') AS rejected,
				COUNT(*) FILTER (WHERE status = 'cancelled') AS cancelled
			`).
			Where("project_id = ?", projectID).
			Scan(&counts).Error

		if err != nil {
			return err
		}

		response.Total = counts.Total
		response.Submitted = counts.Submitted
		response.Reviewing = counts.Reviewing
		response.Accepted = counts.Accepted
		response.Rejected = counts.Rejected
		response.Cancelled = counts.Cancelled

		if project.Status != model.ProjectStatusRecruiting {
			return nil
		}

		return tx.Raw(`
			SELECT COALESCE(
				SUM(
					GREATEST(
						pp.capacity - COALESCE(pm.occupied, 0),
						0
					)
				),
				0
			)
			FROM project_positions pp
			LEFT JOIN (
				SELECT position_id, COUNT(*) AS occupied
				FROM project_members
				WHERE project_id = ?
					AND status = 'active'
				GROUP BY position_id
			) pm ON pm.position_id = pp.id
			WHERE pp.project_id = ?
				AND pp.status = 'open'
				AND pp.deleted_at IS NULL
		`, projectID, projectID).
			Scan(&response.AvailableSlots).Error
	})

	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (r *repository) Review(
	ctx context.Context,
	projectID uuid.UUID,
	applicationID uuid.UUID,
	leadID uuid.UUID,
) (*model.Application, int64, error) {
	return r.changeStatus(
		ctx,
		projectID,
		applicationID,
		leadID,
		model.AppStatusReviewing,
		"",
	)
}

func (r *repository) Accept(
	ctx context.Context,
	projectID uuid.UUID,
	applicationID uuid.UUID,
	leadID uuid.UUID,
	note string,
) (*model.Application, int64, error) {
	return r.changeStatus(
		ctx,
		projectID,
		applicationID,
		leadID,
		model.AppStatusAccepted,
		note,
	)
}

func (r *repository) Reject(
	ctx context.Context,
	projectID uuid.UUID,
	applicationID uuid.UUID,
	leadID uuid.UUID,
	note string,
) (*model.Application, int64, error) {
	return r.changeStatus(
		ctx,
		projectID,
		applicationID,
		leadID,
		model.AppStatusRejected,
		note,
	)
}

func (r *repository) changeStatus(
	ctx context.Context,
	projectID uuid.UUID,
	applicationID uuid.UUID,
	leadID uuid.UUID,
	target model.ApplicationStatus,
	note string,
) (*model.Application, int64, error) {
	var result model.Application
	var occupied int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		project, err := lockSelectionLead(
			tx,
			projectID,
			leadID,
			"UPDATE",
		)
		if err != nil {
			return err
		}

		if project.Status == model.ProjectStatusCompleted {
			return sentinel.ErrProjectCompleted
		}

		var application model.Application

		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(
				"id = ? AND project_id = ?",
				applicationID,
				projectID,
			).
			Take(&application).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return sentinel.ErrApplicationNotFound
		}

		if err != nil {
			return err
		}

		switch target {
		case model.AppStatusReviewing:
			if application.Status != model.AppStatusSubmitted {
				return sentinel.ErrInvalidApplicationStatus
			}

		case model.AppStatusAccepted, model.AppStatusRejected:
			if application.Status != model.AppStatusReviewing {
				return sentinel.ErrInvalidApplicationStatus
			}

		default:
			return sentinel.ErrInvalidApplicationStatus
		}

		now := time.Now()

		if target == model.AppStatusAccepted {
			if err := acceptMember(tx, &application, now); err != nil {
				return err
			}
		}

		updates := map[string]interface{}{
			"status":     target,
			"updated_at": now,
		}

		if target == model.AppStatusAccepted ||
			target == model.AppStatusRejected {
			updates["decided_by"] = leadID
			updates["decided_at"] = now

			if note == "" {
				updates["selection_note"] = nil
			} else {
				updates["selection_note"] = note
			}
		}

		update := tx.Model(&model.Application{}).
			Where(
				"id = ? AND project_id = ? AND status = ?",
				applicationID,
				projectID,
				application.Status,
			).
			Updates(updates)

		if update.Error != nil {
			return update.Error
		}

		if update.RowsAffected != 1 {
			return sentinel.ErrInvalidApplicationStatus
		}

		result, occupied, err = loadApplication(
			tx,
			projectID,
			applicationID,
		)
		if err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]interface{}{
			"type":           "application.updated",
			"project_id":     projectID,
			"application_id": applicationID,
			"applicant_id":   application.ApplicantID,
			"status":         target,
		})
		if err != nil {
			return err
		}

		return tx.Exec(
			"SELECT pg_notify('selection_events', ?)",
			string(payload),
		).Error
	})

	if err != nil {
		return nil, 0, err
	}

	return &result, occupied, nil
}

func acceptMember(
	tx *gorm.DB,
	application *model.Application,
	now time.Time,
) error {
	var position model.ProjectPosition

	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"id = ? AND project_id = ?",
			application.PositionID,
			application.ProjectID,
		).
		Take(&position).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return sentinel.ErrPositionNotFound
	}

	if err != nil {
		return err
	}

	var occupied int64

	err = tx.Model(&model.ProjectMember{}).
		Where(
			"project_id = ? AND position_id = ? AND status = ?",
			application.ProjectID,
			position.ID,
			model.MembershipStatusActive,
		).
		Count(&occupied).Error

	if err != nil {
		return err
	}

	if occupied >= int64(position.Capacity) {
		return sentinel.ErrPositionFull
	}

	var member model.ProjectMember

	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"project_id = ? AND user_id = ?",
			application.ProjectID,
			application.ApplicantID,
		).
		Take(&member).Error

	if err == nil {
		if member.Status == model.MembershipStatusActive {
			return sentinel.ErrAlreadyProjectMember
		}

		return tx.Model(&model.ProjectMember{}).
			Where("id = ?", member.ID).
			Updates(map[string]interface{}{
				"position_id": position.ID,
				"role":        model.ProjectRoleMember,
				"status":      model.MembershipStatusActive,
				"joined_at":   now,
				"ended_at":    nil,
				"updated_at":  now,
			}).Error
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	member = model.ProjectMember{
		ProjectID:  application.ProjectID,
		UserID:     application.ApplicantID,
		PositionID: &position.ID,
		Role:       model.ProjectRoleMember,
		Status:     model.MembershipStatusActive,
		JoinedAt:   now,
	}

	return tx.Omit(clause.Associations).Create(&member).Error
}

func lockSelectionLead(
	tx *gorm.DB,
	projectID uuid.UUID,
	leadID uuid.UUID,
	strength string,
) (*model.Project, error) {
	var project model.Project

	err := tx.Clauses(clause.Locking{Strength: strength}).
		Where("id = ?", projectID).
		Take(&project).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, sentinel.ErrProjectNotFound
	}

	if err != nil {
		return nil, err
	}

	var member model.ProjectMember

	err = tx.Clauses(clause.Locking{Strength: strength}).
		Where(
			"project_id = ? AND user_id = ? AND status = ?",
			projectID,
			leadID,
			model.MembershipStatusActive,
		).
		Take(&member).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, sentinel.ErrForbidden
	}

	if err != nil {
		return nil, err
	}

	if member.Role != model.ProjectRoleLead {
		return nil, sentinel.ErrForbidden
	}

	return &project, nil
}

func preloadApplication(query *gorm.DB) *gorm.DB {
	return query.
		Preload("Applicant", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "email")
		}).
		Preload("Position", func(db *gorm.DB) *gorm.DB {
			return db.Unscoped()
		}).
		Preload("Position.Skills")
}

func loadApplication(
	tx *gorm.DB,
	projectID uuid.UUID,
	applicationID uuid.UUID,
) (model.Application, int64, error) {
	var application model.Application

	err := preloadApplication(tx).
		Where(
			"applications.id = ? AND applications.project_id = ?",
			applicationID,
			projectID,
		).
		Take(&application).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return application, 0, sentinel.ErrApplicationNotFound
	}

	if err != nil {
		return application, 0, err
	}

	counts, err := countPositionMembers(
		tx,
		projectID,
		[]uuid.UUID{application.PositionID},
	)
	if err != nil {
		return application, 0, err
	}

	return application, counts[application.PositionID], nil
}

func countPositionMembers(
	tx *gorm.DB,
	projectID uuid.UUID,
	positionIDs []uuid.UUID,
) (map[uuid.UUID]int64, error) {
	result := make(map[uuid.UUID]int64)

	if len(positionIDs) == 0 {
		return result, nil
	}

	var rows []struct {
		PositionID uuid.UUID
		Occupied   int64
	}

	err := tx.Model(&model.ProjectMember{}).
		Select("position_id, COUNT(*) AS occupied").
		Where(
			"project_id = ? AND position_id IN ? AND status = ?",
			projectID,
			positionIDs,
			model.MembershipStatusActive,
		).
		Group("position_id").
		Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.PositionID] = row.Occupied
	}

	return result, nil
}
