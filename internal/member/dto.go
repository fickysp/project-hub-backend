package member

import (
	"time"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/google/uuid"
)

type MembershipResponse struct {
	ID         uuid.UUID              `json:"id"`
	ProjectID  uuid.UUID              `json:"project_id"`
	UserID     uuid.UUID              `json:"user_id"`
	PositionID *uuid.UUID             `json:"position_id"`
	Role       model.ProjectRole      `json:"role"`
	Status     model.MembershipStatus `json:"status"`
	EndedAt    *time.Time             `json:"ended_at"`
}

func toMembershipResponse(member *model.ProjectMember) MembershipResponse {
	return MembershipResponse{
		ID:         member.ID,
		ProjectID:  member.ProjectID,
		UserID:     member.UserID,
		PositionID: member.PositionID,
		Role:       member.Role,
		Status:     member.Status,
		EndedAt:    member.EndedAt,
	}
}