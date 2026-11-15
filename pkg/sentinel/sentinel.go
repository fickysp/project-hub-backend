package sentinel

import "errors"

var (
	ErrValidate                 = "Data tidak valid, cek kembali data anda"
	ErrInternalServerError      = errors.New("Internal Server Error")
	ErrInvalidCredentials       = errors.New("Email or Password Incorrect")
	ErrEmailIsUsed              = errors.New("Email has already been used")
	ErrUserNotFound             = errors.New("User not found")
	ErrUnauthorized             = errors.New("User Unauthorized")
	ErrProjectNotFound          = errors.New("Project not found")
	ErrForbidden                = errors.New("Only an active project lead can perform this action")
	ErrPendingApplications      = errors.New("Resolve pending applications before completing the project")
	ErrInvalidSkills            = errors.New("One or more skills do not exist")
	ErrApplicationNotFound      = errors.New("Application not found")
	ErrPositionNotFound         = errors.New("Position not found")
	ErrInvalidApplicationStatus = errors.New("Application status does not allow this action")
	ErrPositionFull             = errors.New("Position quota is full")
	ErrAlreadyProjectMember     = errors.New("Applicant is already an active project member")
	ErrProjectCompleted         = errors.New("Completed projects cannot process applications")
)
