package member

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrUnauthorized     = errors.New("authentication required")
	ErrForbidden        = errors.New("you do not have permission to perform this action")
	ErrProjectNotFound  = errors.New("project not found")
	ErrMemberNotFound   = errors.New("project member not found")
	ErrMemberInactive   = errors.New("the project membership is no longer active")
	ErrLastProjectLead  = errors.New("the last project lead cannot leave or be removed")
	ErrUseLeaveEndpoint = errors.New("use the leave project endpoint to leave yourself")
	ErrInvalidID        = errors.New("project ID and member ID must not be empty UUIDs")
)

type Usecase interface {
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

func (u *usecase) Leave(
	ctx context.Context,
	projectID uuid.UUID,
	userID uuid.UUID,
) (*MembershipResponse, error) {
	if userID == uuid.Nil {
		return nil, ErrUnauthorized
	}

	if projectID == uuid.Nil {
		return nil, ErrInvalidID
	}

	member, err := u.repository.Leave(
		ctx,
		projectID,
		userID,
	)
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
	if actorID == uuid.Nil {
		return nil, ErrUnauthorized
	}

	if projectID == uuid.Nil || memberID == uuid.Nil {
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