package project

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
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

type CacheClient interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	DeletePrefix(ctx context.Context, prefix string) error
}

type Usecase interface {
	List(
		ctx context.Context,
		userID uuid.UUID,
		request ListProjectsRequest,
	) (*ListProjectsResponse, error)

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
	cache      CacheClient
}

func NewUsecase(repository Repository, cache CacheClient) Usecase {
	return &usecase{
		repository: repository,
		cache:      cache,
	}
}

func (u *usecase) List(
	ctx context.Context,
	userID uuid.UUID,
	request ListProjectsRequest,
) (*ListProjectsResponse, error) {
	if userID == uuid.Nil {
		return nil, sentinel.ErrUnauthorized
	}

	request.Search = strings.TrimSpace(request.Search)

	if utf8.RuneCountInString(request.Search) > 150 {
		return nil, &ValidationError{
			Field:   "search",
			Message: "must not exceed 150 characters",
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
		model.ProjectStatusRecruiting,
		model.ProjectStatusActive,
		model.ProjectStatusCompleted:
	default:
		return nil, &ValidationError{
			Field:   "status",
			Message: "must be recruiting, active, or completed",
		}
	}

	if request.SkillID != "" {
		skillID, err := uuid.Parse(request.SkillID)
		if err != nil || skillID == uuid.Nil {
			return nil, &ValidationError{
				Field:   "skill_id",
				Message: "must be a valid non-empty UUID",
			}
		}

		request.SkillID = skillID.String()
	}

	cacheKey := fmt.Sprintf("projects:list:search=%s:status=%s:skill=%s:page=%d:limit=%d",
		request.Search, request.Status, request.SkillID, request.Page, request.Limit)

	if cachedData, err := u.cache.Get(ctx, cacheKey); err == nil && cachedData != nil {
		var response ListProjectsResponse
		if err := json.Unmarshal(cachedData, &response); err == nil {
			return &response, nil
		}
	}

	projects, total, err := u.repository.List(ctx, request)
	if err != nil {
		return nil, err
	}

	positionIDs := make([]uuid.UUID, 0)

	for _, project := range projects {
		if project.Status != model.ProjectStatusRecruiting {
			continue
		}

		for _, position := range project.Positions {
			if position.Status == model.PositionStatusOpen &&
				!position.DeletedAt.Valid {
				positionIDs = append(positionIDs, position.ID)
			}
		}
	}

	memberCounts, err := u.repository.CountActiveMembersByPosition(
		ctx,
		positionIDs,
	)
	if err != nil {
		return nil, err
	}

	items := make([]ProjectCatalogItem, 0, len(projects))

	for _, project := range projects {
		item := ProjectCatalogItem{
			ID:                project.ID,
			Name:              project.Name,
			Summary:           project.Summary,
			Status:            project.Status,
			Skills:            make([]SkillResponse, 0, len(project.Skills)),
			Leads:             make([]ProjectLeadResponse, 0),
			OpenPositionNames: make([]string, 0),
		}

		for _, skill := range project.Skills {
			item.Skills = append(item.Skills, SkillResponse{
				ID:   skill.ID,
				Name: skill.Name,
			})
		}

		for _, member := range project.Members {
			if member.Role != model.ProjectRoleLead ||
				member.Status != model.MembershipStatusActive {
				continue
			}

			item.Leads = append(item.Leads, ProjectLeadResponse{
				ID:   member.UserID,
				Name: member.User.Name,
			})
		}

		if project.Status == model.ProjectStatusRecruiting {
			for _, position := range project.Positions {
				if position.Status != model.PositionStatusOpen ||
					position.DeletedAt.Valid {
					continue
				}

				available := int64(position.Capacity) -
					memberCounts[position.ID]

				if available <= 0 {
					continue
				}

				item.OpenPositions++
				item.AvailableSlots += available
				item.OpenPositionNames = append(
					item.OpenPositionNames,
					position.Name,
				)
			}
		}

		items = append(items, item)
	}

	totalPages := total / int64(request.Limit)
	if total%int64(request.Limit) != 0 {
		totalPages++
	}

	response := &ListProjectsResponse{
		Data: items,
		Pagination: PaginationResponse{
			Page:       request.Page,
			Limit:      request.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	if responseBytes, err := json.Marshal(response); err == nil {
		_ = u.cache.Set(ctx, cacheKey, responseBytes, 5*time.Minute)
	}

	return response, nil
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

	_ = u.cache.DeletePrefix(ctx, "projects:list:")

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

	_ = u.cache.DeletePrefix(ctx, "projects:list:")

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

	err := u.repository.Delete(ctx, projectID, userID)
	if err != nil {
		return err
	}

	_ = u.cache.DeletePrefix(ctx, "projects:list:")

	return nil
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
