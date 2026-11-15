package selection

import (
	"time"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/google/uuid"
)

type ListApplicationsRequest struct {
	Search     string                  `form:"search" binding:"max=255"`
	PositionID string                  `form:"position_id" binding:"omitempty,uuid"`
	Status     model.ApplicationStatus `form:"status" binding:"omitempty,oneof=submitted reviewing accepted rejected cancelled"`
	Page       int                     `form:"page,default=1" binding:"min=1,max=1000000"`
	Limit      int                     `form:"limit,default=10" binding:"min=1,max=100"`
}

type AcceptApplicationRequest struct {
	SelectionNote string `json:"selection_note" binding:"max=2000"`
}

type RejectApplicationRequest struct {
	SelectionNote string `json:"selection_note" binding:"required,max=2000"`
}

type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

type ApplicantResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type SkillResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type PositionResponse struct {
	ID          uuid.UUID            `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Status      model.PositionStatus `json:"status"`
	Capacity    int                  `json:"capacity"`
	Occupied    int64                `json:"occupied"`
	Available   int64                `json:"available"`
	Skills      []SkillResponse      `json:"skills"`
}

type ApplicationResponse struct {
	ID             uuid.UUID               `json:"id"`
	ProjectID      uuid.UUID               `json:"project_id"`
	Applicant      ApplicantResponse       `json:"applicant"`
	Position       PositionResponse        `json:"position"`
	RelevantSkills string                  `json:"relevant_skills"`
	Motivation     string                  `json:"motivation"`
	Status         model.ApplicationStatus `json:"status"`
	SelectionNote  *string                 `json:"selection_note"`
	PortfolioURL   *string                 `json:"portfolio_url,omitempty"`
	DecidedBy      *uuid.UUID              `json:"decided_by"`
	DecidedAt      *time.Time              `json:"decided_at"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

type ListApplicationsResponse struct {
	Data       []ApplicationResponse `json:"data"`
	Pagination PaginationResponse    `json:"pagination"`
}

type ApplicationSummaryResponse struct {
	ProjectID      uuid.UUID           `json:"project_id"`
	ProjectName    string              `json:"project_name"`
	ProjectStatus  model.ProjectStatus `json:"project_status"`
	Total          int64               `json:"total"`
	Submitted      int64               `json:"submitted"`
	Reviewing      int64               `json:"reviewing"`
	Accepted       int64               `json:"accepted"`
	Rejected       int64               `json:"rejected"`
	Cancelled      int64               `json:"cancelled"`
	AvailableSlots int64               `json:"available_slots"`
}