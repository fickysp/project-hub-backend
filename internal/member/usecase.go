package member

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrUnauthorized      = errors.New("authentication required")
	ErrForbidden         = errors.New("you do not have permission to perform this action")
	ErrProjectNotFound   = errors.New("project not found")
	ErrMemberNotFound    = errors.New("project member not found")
	ErrMemberInactive    = errors.New("the project membership is no longer active")
	ErrLastProjectLead   = errors.New("the last project lead cannot leave or be removed")
	ErrUseLeaveEndpoint  = errors.New("use the leave project endpoint to leave yourself")
	ErrInvalidID         = errors.New("IDs must be valid non-empty UUIDs")
	ErrInvalidPagination = errors.New("page must be between 1 and 1000000 and limit between 1 and 100")
	ErrPositionNotFound  = errors.New("position not found in this project")
	ErrPositionFull      = errors.New("the target position has no available quota")
)

type Usecase interface {
	List(
		ctx context.Context,
		projectID uuid.UUID,
		userID uuid.UUID,
		request ListMembersRequest,
	) (*ListMembersResponse, error)

	UpdatePosition(
		ctx context.Context,
		projectID uuid.UUID,
		memberID uuid.UUID,
		actorID uuid.UUID,
		request UpdateMemberPositionRequest,
	) (*MemberResponse, error)

	Leave(
		ctx context.Context,
		projectID uuid.UUID,
		userID uuid.UUID,
	) (*MembershipResponse, error)

	Remove(
		ctx context.Context,
		projectID uuid.UUID,
		memberID uuid.UUID,
		actorID uuid.UUID,
	) (*MembershipResponse, error)
}

type usecase struct {
	repository Repository
}

func NewUsecase(repository Repository) Usecase {
	return &usecase{repository: repository}
}

func (u *usecase) List(
	ctx context.Context,
	projectID uuid.UUID,
	userID uuid.UUID,
	request ListMembersRequest,
) (*ListMembersResponse, error) {
	if err := validateAccess(projectID, userID); err != nil {
		return nil, err
	}

	if request.Page == 0 {
		request.Page = 1
	}

	if request.Limit == 0 {
		request.Limit = 10
	}

	if request.Page < 1 || request.Page > 1000000 ||
		request.Limit < 1 || request.Limit > 100 {
		return nil, ErrInvalidPagination
	}

	members, total, err := u.repository.List(
		ctx,
		projectID,
		userID,
		request,
	)
	if err != nil {
		return nil, err
	}

	items := make([]MemberResponse, 0, len(members))

	for i := range members {
		items = append(items, toMemberResponse(&members[i]))
	}

	totalPages := total / int64(request.Limit)
	if total%int64(request.Limit) != 0 {
		totalPages++
	}

	return &ListMembersResponse{
		Data: items,
		Pagination: PaginationResponse{
			Page:       request.Page,
			Limit:      request.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (u *usecase) UpdatePosition(
	ctx context.Context,
	projectID uuid.UUID,
	memberID uuid.UUID,
	actorID uuid.UUID,
	request UpdateMemberPositionRequest,
) (*MemberResponse, error) {
	if err := validateAccess(projectID, actorID); err != nil {
		return nil, err
	}

	if memberID == uuid.Nil || request.PositionID == uuid.Nil {
		return nil, ErrInvalidID
	}

	member, err := u.repository.UpdatePosition(
		ctx,
		projectID,
		memberID,
		actorID,
		request.PositionID,
	)
	if err != nil {
		return nil, err
	}

	response := toMemberResponse(member)

	return &response, nil
}

func (u *usecase) Leave(
	ctx context.Context,
	projectID uuid.UUID,
	userID uuid.UUID,
) (*MembershipResponse, error) {
	if err := validateAccess(projectID, userID); err != nil {
		return nil, err
	}

	member, err := u.repository.Leave(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}

	response := toMembershipResponse(member)

	return &response, nil
}

func (u *usecase) Remove(
	ctx context.Context,
	projectID uuid.UUID,
	memberID uuid.UUID,
	actorID uuid.UUID,
) (*MembershipResponse, error) {
	if err := validateAccess(projectID, actorID); err != nil {
		return nil, err
	}

	if memberID == uuid.Nil {
		return nil, ErrInvalidID
	}

	member, err := u.repository.Remove(
		ctx,
		projectID,
		memberID,
		actorID,
	)
	if err != nil {
		return nil, err
	}

	response := toMembershipResponse(member)

	return &response, nil
}

func validateAccess(projectID, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return ErrUnauthorized
	}

	if projectID == uuid.Nil {
		return ErrInvalidID
	}

	return nil
}
