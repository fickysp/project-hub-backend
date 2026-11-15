package role

import (
	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/google/uuid"
)

type UpdateMemberRoleRequest struct {
	Role model.ProjectRole `json:"role" binding:"required,oneof=project_lead member"`
}

type MemberRoleResponse struct {
	ID         uuid.UUID              `json:"id"`
	ProjectID  uuid.UUID              `json:"project_id"`
	UserID     uuid.UUID              `json:"user_id"`
	PositionID *uuid.UUID             `json:"position_id"`
	Role       model.ProjectRole      `json:"role"`
	Status     model.MembershipStatus `json:"status"`
}

func toMemberRoleResponse(member *model.ProjectMember) MemberRoleResponse {
	return MemberRoleResponse{
		ID:         member.ID,
		ProjectID:  member.ProjectID,
		UserID:     member.UserID,
		PositionID: member.PositionID,
		Role:       member.Role,
		Status:     member.Status,
	}
}
