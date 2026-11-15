package model

type ProjectStatus string
type PositionStatus string
type ProjectRole string
type MembershipStatus string
type ApplicationStatus string

const (
	ProjectStatusRecruiting ProjectStatus = "recruiting"
	ProjectStatusActive     ProjectStatus = "active"
	ProjectStatusCompleted  ProjectStatus = "completed"
)

const (
	PositionStatusOpen   PositionStatus = "open"
	PositionStatusClosed PositionStatus = "closed"
)

const (
	ProjectRoleLead   ProjectRole = "project_lead"
	ProjectRoleMember ProjectRole = "member"
)

const (
	MembershipStatusActive  MembershipStatus = "active"
	MembershipStatusLeft    MembershipStatus = "left"
	MembershipStatusRemoved MembershipStatus = "removed"
)

const (
	AppStatusSubmitted ApplicationStatus = "submitted"
	AppStatusReviewing ApplicationStatus = "reviewing"
	AppStatusAccepted  ApplicationStatus = "accepted"
	AppStatusRejected  ApplicationStatus = "rejected"
	AppStatusCancelled ApplicationStatus = "cancelled"
)
