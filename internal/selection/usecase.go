package selection

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/fickysp/project-hub-backend.git/pkg/sentinel"
	"github.com/google/uuid"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

type Usecase interface {
	List(
		ctx context.Context,
		projectID uuid.UUID,
		leadID uuid.UUID,
		request ListApplicationsRequest,
	) (*ListApplicationsResponse, error)

	Detail(
		ctx context.Context,
		projectID uuid.UUID,
		applicationID uuid.UUID,
		leadID uuid.UUID,
	) (*ApplicationResponse, error)

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
	) (*ApplicationResponse, error)

	Accept(
		ctx context.Context,
		projectID uuid.UUID,
		applicationID uuid.UUID,
		leadID uuid.UUID,
		request AcceptApplicationRequest,
	) (*ApplicationResponse, error)

	Reject(
		ctx context.Context,
		projectID uuid.UUID,
		applicationID uuid.UUID,
		leadID uuid.UUID,
		request RejectApplicationRequest,
	) (*ApplicationResponse, error)
}

type usecase struct {
	repository Repository
}

func NewUsecase(repository Repository) Usecase {
	return &usecase{
		repository: repository,
	}
}

func (u *usecase) List(
	ctx context.Context,
	projectID uuid.UUID,
	leadID uuid.UUID,
	request ListApplicationsRequest,
) (*ListApplicationsResponse, error) {
	if err := validateProjectAccessIDs(projectID, leadID); err != nil {
		return nil, err
	}

	request.Search = strings.TrimSpace(request.Search)

	if utf8.RuneCountInString(request.Search) > 255 {
		return nil, &ValidationError{
			Field:   "search",
			Message: "must not exceed 255 characters",
		}
	}

	if request.Page == 0 {
		request.Page = 1
	}

	if request.Limit == 0 {
		request.Limit = 10
	}

	if request.Page < 1 || request.Page > 1000000 {
		return nil, &ValidationError{
			Field:   "page",
			Message: "must be between 1 and 1000000",
		}
	}

	if request.Limit < 1 || request.Limit > 100 {
		return nil, &ValidationError{
			Field:   "limit",
			Message: "must be between 1 and 100",
		}
	}

	switch request.Status {
	case "",
		model.AppStatusSubmitted,
		model.AppStatusReviewing,
		model.AppStatusAccepted,
		model.AppStatusRejected,
		model.AppStatusCancelled:
	default:
		return nil, &ValidationError{
			Field: "status",
			Message: "must be submitted, reviewing, accepted, " +
				"rejected, or cancelled",
		}
	}

	if request.PositionID != "" {
		positionID, err := uuid.Parse(request.PositionID)
		if err != nil || positionID == uuid.Nil {
			return nil, &ValidationError{
				Field:   "position_id",
				Message: "must be a valid non-empty UUID",
			}
		}

		request.PositionID = positionID.String()
	}

	result, err := u.repository.List(
		ctx,
		projectID,
		leadID,
		request,
	)
	if err != nil {
		return nil, err
	}

	items := make([]ApplicationResponse, 0, len(result.Applications))

	for _, application := range result.Applications {
		items = append(
			items,
			toApplicationResponse(
				&application,
				result.Occupied[application.PositionID],
			),
		)
	}

	totalPages := result.Total / int64(request.Limit)
	if result.Total%int64(request.Limit) != 0 {
		totalPages++
	}

	return &ListApplicationsResponse{
		Data: items,
		Pagination: PaginationResponse{
			Page:       request.Page,
			Limit:      request.Limit,
			Total:      result.Total,
			TotalPages: totalPages,
		},
	}, nil
}

func (u *usecase) Detail(
	ctx context.Context,
	projectID uuid.UUID,
	applicationID uuid.UUID,
	leadID uuid.UUID,
) (*ApplicationResponse, error) {
	if err := validateApplicationAccessIDs(
		projectID,
		applicationID,
		leadID,
	); err != nil {
		return nil, err
	}

	application, occupied, err := u.repository.Detail(
		ctx,
		projectID,
		applicationID,
		leadID,
	)
	if err != nil {
		return nil, err
	}

	response := toApplicationResponse(application, occupied)

	return &response, nil
}

func (u *usecase) Summary(
	ctx context.Context,
	projectID uuid.UUID,
	leadID uuid.UUID,
) (*ApplicationSummaryResponse, error) {
	if err := validateProjectAccessIDs(projectID, leadID); err != nil {
		return nil, err
	}

	return u.repository.Summary(ctx, projectID, leadID)
}

func (u *usecase) Review(
	ctx context.Context,
	projectID uuid.UUID,
	applicationID uuid.UUID,
	leadID uuid.UUID,
) (*ApplicationResponse, error) {
	if err := validateApplicationAccessIDs(
		projectID,
		applicationID,
		leadID,
	); err != nil {
		return nil, err
	}

	application, occupied, err := u.repository.Review(
		ctx,
		projectID,
		applicationID,
		leadID,
	)
	if err != nil {
		return nil, err
	}

	response := toApplicationResponse(application, occupied)

	return &response, nil
}

func (u *usecase) Accept(
	ctx context.Context,
	projectID uuid.UUID,
	applicationID uuid.UUID,
	leadID uuid.UUID,
	request AcceptApplicationRequest,
) (*ApplicationResponse, error) {
	if err := validateApplicationAccessIDs(
		projectID,
		applicationID,
		leadID,
	); err != nil {
		return nil, err
	}

	note, err := validateSelectionNote(request.SelectionNote, false)
	if err != nil {
		return nil, err
	}

	application, occupied, err := u.repository.Accept(
		ctx,
		projectID,
		applicationID,
		leadID,
		note,
	)
	if err != nil {
		return nil, err
	}

	response := toApplicationResponse(application, occupied)

	return &response, nil
}

func (u *usecase) Reject(
	ctx context.Context,
	projectID uuid.UUID,
	applicationID uuid.UUID,
	leadID uuid.UUID,
	request RejectApplicationRequest,
) (*ApplicationResponse, error) {
	if err := validateApplicationAccessIDs(
		projectID,
		applicationID,
		leadID,
	); err != nil {
		return nil, err
	}

	note, err := validateSelectionNote(request.SelectionNote, true)
	if err != nil {
		return nil, err
	}

	application, occupied, err := u.repository.Reject(
		ctx,
		projectID,
		applicationID,
		leadID,
		note,
	)
	if err != nil {
		return nil, err
	}

	response := toApplicationResponse(application, occupied)

	return &response, nil
}

func validateProjectAccessIDs(
	projectID uuid.UUID,
	leadID uuid.UUID,
) error {
	if leadID == uuid.Nil {
		return sentinel.ErrUnauthorized
	}

	if projectID == uuid.Nil {
		return &ValidationError{
			Field:   "project_id",
			Message: "must not be an empty UUID",
		}
	}

	return nil
}

func validateApplicationAccessIDs(
	projectID uuid.UUID,
	applicationID uuid.UUID,
	leadID uuid.UUID,
) error {
	if err := validateProjectAccessIDs(projectID, leadID); err != nil {
		return err
	}

	if applicationID == uuid.Nil {
		return &ValidationError{
			Field:   "application_id",
			Message: "must not be an empty UUID",
		}
	}

	return nil
}

func validateSelectionNote(note string, required bool) (string, error) {
	note = strings.TrimSpace(note)

	if required && note == "" {
		return "", &ValidationError{
			Field:   "selection_note",
			Message: "a rejection note is required",
		}
	}

	if utf8.RuneCountInString(note) > 2000 {
		return "", &ValidationError{
			Field:   "selection_note",
			Message: fmt.Sprintf("must not exceed %d characters", 2000),
		}
	}

	return note, nil
}

func toApplicationResponse(
	application *model.Application,
	occupied int64,
) ApplicationResponse {
	position := application.Position

	skills := make([]SkillResponse, 0, len(position.Skills))

	for _, skill := range position.Skills {
		skills = append(skills, SkillResponse{
			ID:   skill.ID,
			Name: skill.Name,
		})
	}

	available := int64(position.Capacity) - occupied

	if available < 0 || position.DeletedAt.Valid {
		available = 0
	}

	return ApplicationResponse{
		ID:        application.ID,
		ProjectID: application.ProjectID,
		Applicant: ApplicantResponse{
			ID:    application.ApplicantID,
			Name:  application.Applicant.Name,
			Email: application.Applicant.Email,
		},
		Position: PositionResponse{
			ID:          application.PositionID,
			Name:        position.Name,
			Description: position.Description,
			Status:      position.Status,
			Capacity:    position.Capacity,
			Occupied:    occupied,
			Available:   available,
			Skills:      skills,
		},
		RelevantSkills: append([]string{}, application.RelevantSkills...),
		Motivation:     application.Motivation,
		Status:         application.Status,
		SelectionNote:  application.SelectionNote,
		PortfolioURL:   application.PortfolioUrl,
		DecidedBy:      application.DecidedBy,
		DecidedAt:      application.DecidedAt,
		CreatedAt:      application.CreatedAt,
		UpdatedAt:      application.UpdatedAt,
	}
}
