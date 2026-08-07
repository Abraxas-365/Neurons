package enrollment

import (
	"net/http"

	"github.com/Abraxas-356/neurons/internal/errx"
)

var ErrRegistry = errx.NewRegistry("ENROLLMENT")

var (
	CodeNotFound = ErrRegistry.Register(
		"NOT_FOUND",
		errx.TypeNotFound,
		http.StatusNotFound,
		"Enrollment not found",
	)

	CodeAlreadyEnrolled = ErrRegistry.Register(
		"ALREADY_ENROLLED",
		errx.TypeBusiness,
		http.StatusConflict,
		"The student is already enrolled in this classroom",
	)

	CodeInvalidInput = ErrRegistry.Register(
		"INVALID_INPUT",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Invalid enrollment data",
	)

	// RN-16: withdrawn or pending students cannot move neurons.
	CodeNotActive = ErrRegistry.Register(
		"NOT_ACTIVE",
		errx.TypeBusiness,
		http.StatusConflict,
		"The student is not active in this classroom",
	)

	// RN-04 / MVP criterion 15.
	CodeInsufficientBalance = ErrRegistry.Register(
		"INSUFFICIENT_BALANCE",
		errx.TypeBusiness,
		http.StatusConflict,
		"The student does not have enough neurons",
	)

	CodeForbidden = ErrRegistry.Register(
		"FORBIDDEN",
		errx.TypeAuthorization,
		http.StatusForbidden,
		"You do not have access to this enrollment",
	)

	CodePendingApproval = ErrRegistry.Register(
		"PENDING_APPROVAL",
		errx.TypeBusiness,
		http.StatusAccepted,
		"Your request to join is awaiting the teacher's approval",
	)
)

func ErrNotFound() error            { return ErrRegistry.New(CodeNotFound) }
func ErrAlreadyEnrolled() error     { return ErrRegistry.New(CodeAlreadyEnrolled) }
func ErrNotActive() error           { return ErrRegistry.New(CodeNotActive) }
func ErrInsufficientBalance() error { return ErrRegistry.New(CodeInsufficientBalance) }
func ErrForbidden() error           { return ErrRegistry.New(CodeForbidden) }

func ErrInvalidInput(msg string) error {
	return ErrRegistry.NewWithMessage(CodeInvalidInput, msg)
}
