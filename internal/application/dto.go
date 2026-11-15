package application

import (
	"time"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/google/uuid"
)

type SubmitApplicationReq struct {
	PositionID     uuid.UUID `json:"position_id" binding:"required"`
	RelevantSkills []string  `json:"relevant_skills" binding:"required,min=1,max=20,dive,required"`
	PortfolioURL   *string   `json:"portfolio_url" binding:"omitempty,url"`
	Motivation     string    `json:"motivation" binding:"required,min=10,max=5000"`
}

type ReviewApplicationReq struct {
	Status        string  `json:"status" binding:"required,oneof=reviewing accepted rejected"`
	SelectionNote *string `json:"selection_note" binding:"omitempty,max=1000"`
}

type ApplicationWsResponse struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	PositionID  uuid.UUID  `json:"position_id"`
	ApplicantID uuid.UUID  `json:"applicant_id"`
	Status      string     `json:"status"`
	DecidedBy   *uuid.UUID `json:"decided_by"`
	DecidedAt   *time.Time `json:"decided_at"`
}

type ApplicationResponse struct {
	ID             uuid.UUID  `json:"id"`
	ProjectID      uuid.UUID  `json:"project_id"`
	PositionID     uuid.UUID  `json:"position_id"`
	PositionName   string     `json:"position_name"`
	ApplicantID    uuid.UUID  `json:"applicant_id"`
	ApplicantName  string     `json:"applicant_name"`
	RelevantSkills []string   `json:"relevant_skills"`
	PortfolioURL   *string    `json:"portfolio_url"`
	Motivation     string     `json:"motivation"`
	Status         string     `json:"status"`
	SelectionNote  *string    `json:"selection_note"`
	DecidedBy      *uuid.UUID `json:"decided_by"`
	DecidedAt      *time.Time `json:"decided_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

func toApplicationResponse(app *model.Application) ApplicationResponse {
	return ApplicationResponse{
		ID:             app.ID,
		ProjectID:      app.ProjectID,
		PositionID:     app.PositionID,
		PositionName:   app.Position.Name,
		ApplicantID:    app.ApplicantID,
		ApplicantName:  app.Applicant.Name,
		RelevantSkills: []string(app.RelevantSkills),
		PortfolioURL:   app.PortfolioUrl,
		Motivation:     app.Motivation,
		Status:         string(app.Status),
		SelectionNote:  app.SelectionNote,
		DecidedBy:      app.DecidedBy,
		DecidedAt:      app.DecidedAt,
		CreatedAt:      app.CreatedAt,
	}
}
