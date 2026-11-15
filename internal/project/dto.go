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