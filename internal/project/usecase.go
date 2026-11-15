package project

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
	Create(
		ctx context.Context,
		userID uuid.UUID,
		request CreateProjectRequest,
	) (*ProjectResponse, error)

	Update(
		ctx context.Context,
		projectID uuid.UUID,
		userID uuid.UUID,
		request UpdateProjectRequest,
	) (*ProjectResponse, error)

	Delete(
		ctx context.Context,
		projectID uuid.UUID,
		userID uuid.UUID,
	) error
}

type usecase struct {
	repository Repository
}

func NewUsecase(repository Repository) Usecase {
	return &usecase{
		repository: repository,
	}
}

func (u *usecase) Create(
	ctx context.Context,
	userID uuid.UUID,
	request CreateProjectRequest,
) (*ProjectResponse, error) {
	if userID == uuid.Nil {
		return nil, sentinel.ErrUnauthorized
	}

	fields := []struct {
		name   string
		value  *string
		maxLen int
	}{
		{"name", &request.Name, 150},
		{"summary", &request.Summary, 255},
		{"description", &request.Description, 20000},
		{"goal", &request.Goal, 20000},
	}

	for _, field := range fields {
		if err := validateProjectText(
			field.name,
			field.value,
			field.maxLen,
		); err != nil {
			return nil, err
		}
	}

	skillIDs, err := normalizeSkillIDs(request.SkillIDs)
	if err != nil {
		return nil, err
	}

	request.SkillIDs = skillIDs

	project, err := u.repository.Create(ctx, userID, request)
	if err != nil {
		return nil, err
	}

	response := toProjectResponse(project)

	return &response, nil
}

func (u *usecase) Update(
	ctx context.Context,
	projectID uuid.UUID,
	userID uuid.UUID,
	request UpdateProjectRequest,
) (*ProjectResponse, error) {
	if userID == uuid.Nil {
		return nil, sentinel.ErrUnauthorized
	}

	if projectID == uuid.Nil {
		return nil, &ValidationError{
			Field:   "project_id",
			Message: "must not be an empty UUID",
		}
	}

	if request.Name == nil &&
		request.Summary == nil &&
		request.Description == nil &&
		request.Goal == nil &&
		request.Status == nil &&
		request.SkillIDs == nil {
		return nil, &ValidationError{
			Field:   "body",
			Message: "provide at least one field to update",
		}
	}

	fields := []struct {
		name   string
		value  *string
		maxLen int
	}{
		{"name", request.Name, 150},
		{"summary", request.Summary, 255},
		{"description", request.Description, 20000},
		{"goal", request.Goal, 20000},
	}

	for _, field := range fields {
		if field.value == nil {
			continue
		}

		if err := validateProjectText(
			field.name,
			field.value,
			field.maxLen,
		); err != nil {
			return nil, err
		}
	}

	if request.Status != nil {
		switch *request.Status {
		case model.ProjectStatusRecruiting,
			model.ProjectStatusActive,
			model.ProjectStatusCompleted:
		default:
			return nil, &ValidationError{
				Field:   "status",
				Message: "must be recruiting, active, or completed",
			}
		}
	}

	if request.SkillIDs != nil {
		skillIDs, err := normalizeSkillIDs(*request.SkillIDs)
		if err != nil {
			return nil, err
		}

		request.SkillIDs = &skillIDs
	}

	project, err := u.repository.Update(
		ctx,
		projectID,
		userID,
		request,
	)
	if err != nil {
		return nil, err
	}

	response := toProjectResponse(project)

	return &response, nil
}

func (u *usecase) Delete(
	ctx context.Context,
	projectID uuid.UUID,
	userID uuid.UUID,
) error {
	if userID == uuid.Nil {
		return sentinel.ErrUnauthorized
	}

	if projectID == uuid.Nil {
		return &ValidationError{
			Field:   "project_id",
			Message: "must not be an empty UUID",
		}
	}

	return u.repository.Delete(ctx, projectID, userID)
}

func validateProjectText(
	field string,
	value *string,
	maxLen int,
) error {
	*value = strings.TrimSpace(*value)

	if *value == "" {
		return &ValidationError{
			Field:   field,
			Message: "must not be blank",
		}
	}

	if utf8.RuneCountInString(*value) > maxLen {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must not exceed %d characters", maxLen),
		}
	}

	return nil
}

func normalizeSkillIDs(skillIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(skillIDs) > 50 {
		return nil, &ValidationError{
			Field:   "skill_ids",
			Message: "must not contain more than 50 skills",
		}
	}

	result := make([]uuid.UUID, 0, len(skillIDs))
	seen := make(map[uuid.UUID]bool)

	for _, id := range skillIDs {
		if id == uuid.Nil {
			return nil, &ValidationError{
				Field:   "skill_ids",
				Message: "must not contain an empty UUID",
			}
		}

		if seen[id] {
			continue
		}

		seen[id] = true
		result = append(result, id)
	}

	return result, nil
}