package project

import (
	"time"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/google/uuid"
)

type CreateProjectRequest struct {
	Name        string      `json:"name" binding:"required,max=150"`
	Summary     string      `json:"summary" binding:"required,max=255"`
	Description string      `json:"description" binding:"required,max=20000"`
	Goal        string      `json:"goal" binding:"required,max=20000"`
	SkillIDs    []uuid.UUID `json:"skill_ids" binding:"max=50"`
}

type UpdateProjectRequest struct {
	Name        *string              `json:"name" binding:"omitempty,min=1,max=150"`
	Summary     *string              `json:"summary" binding:"omitempty,min=1,max=255"`
	Description *string              `json:"description" binding:"omitempty,min=1,max=20000"`
	Goal        *string              `json:"goal" binding:"omitempty,min=1,max=20000"`
	Status      *model.ProjectStatus `json:"status" binding:"omitempty,oneof=recruiting active completed"`
	SkillIDs    *[]uuid.UUID         `json:"skill_ids" binding:"omitempty,max=50"`
}

type SkillResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ProjectResponse struct {
	ID          uuid.UUID           `json:"id"`
	CreatedBy   uuid.UUID           `json:"created_by"`
	Name        string              `json:"name"`
	Summary     string              `json:"summary"`
	Description string              `json:"description"`
	Goal        string              `json:"goal"`
	Status      model.ProjectStatus `json:"status"`
	Skills      []SkillResponse     `json:"skills"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type ProjectLeadResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

type ProjectCatalogItem struct {
	ID                uuid.UUID             `json:"id"`
	Name              string                `json:"name"`
	Summary           string                `json:"summary"`
	Status            model.ProjectStatus   `json:"status"`
	Skills            []SkillResponse       `json:"skills"`
	Leads             []ProjectLeadResponse `json:"leads"`
	OpenPositions     int                   `json:"open_positions"`
	AvailableSlots    int64                 `json:"available_slots"`
	OpenPositionNames []string              `json:"open_position_names"`
}

type ListProjectsRequest struct {
	Search  string              `form:"search" binding:"max=150"`
	Status  model.ProjectStatus `form:"status" binding:"omitempty,oneof=recruiting active completed"`
	SkillID string              `form:"skill_id" binding:"omitempty,uuid"`
	Page    int                 `form:"page,default=1" binding:"min=1,max=1000000"`
	Limit   int                 `form:"limit,default=10" binding:"min=1,max=100"`
}

type ListProjectsResponse struct {
	Data       []ProjectCatalogItem `json:"data"`
	Pagination PaginationResponse   `json:"pagination"`
}

func toProjectResponse(project *model.Project) ProjectResponse {
	skills := make([]SkillResponse, 0, len(project.Skills))

	for _, skill := range project.Skills {
		skills = append(skills, SkillResponse{
			ID:   skill.ID,
			Name: skill.Name,
		})
	}

	return ProjectResponse{
		ID:          project.ID,
		CreatedBy:   project.CreatedBy,
		Name:        project.Name,
		Summary:     project.Summary,
		Description: project.Description,
		Goal:        project.Goal,
		Status:      project.Status,
		Skills:      skills,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}
}