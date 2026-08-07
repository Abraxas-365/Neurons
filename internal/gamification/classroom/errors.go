package classroom

import (
	"net/http"

	"github.com/Abraxas-356/neurons/internal/errx"
)

var ErrRegistry = errx.NewRegistry("CLASSROOM")

var (
	CodeClassroomNotFound = ErrRegistry.Register(
		"NOT_FOUND",
		errx.TypeNotFound,
		http.StatusNotFound,
		"Classroom not found",
	)

	CodeClassroomAlreadyExists = ErrRegistry.Register(
		"ALREADY_EXISTS",
		errx.TypeBusiness,
		http.StatusConflict,
		"Classroom already exists",
	)

	CodeInvalidInput = ErrRegistry.Register(
		"INVALID_INPUT",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Invalid classroom data",
	)

	// RN-17: no transactions once the classroom is closed or archived.
	CodeClassroomNotActive = ErrRegistry.Register(
		"NOT_ACTIVE",
		errx.TypeBusiness,
		http.StatusConflict,
		"Classroom is not active",
	)

	CodeNotTeacher = ErrRegistry.Register(
		"NOT_TEACHER",
		errx.TypeAuthorization,
		http.StatusForbidden,
		"You are not a teacher of this classroom",
	)

	CodeNotOwner = ErrRegistry.Register(
		"NOT_OWNER",
		errx.TypeAuthorization,
		http.StatusForbidden,
		"Only the classroom owner can perform this action",
	)

	CodeTeacherAlreadyAdded = ErrRegistry.Register(
		"TEACHER_ALREADY_ADDED",
		errx.TypeBusiness,
		http.StatusConflict,
		"This user is already a teacher of the classroom",
	)

	CodeCannotRemoveOwner = ErrRegistry.Register(
		"CANNOT_REMOVE_OWNER",
		errx.TypeBusiness,
		http.StatusConflict,
		"The classroom owner cannot be removed",
	)

	CodeInviteCodeInvalid = ErrRegistry.Register(
		"INVITE_CODE_INVALID",
		errx.TypeNotFound,
		http.StatusNotFound,
		"Invite code is not valid",
	)

	// RN-05: the vault cannot go below zero.
	CodeInsufficientVault = ErrRegistry.Register(
		"INSUFFICIENT_VAULT",
		errx.TypeBusiness,
		http.StatusConflict,
		"The classroom vault does not have enough neurons",
	)
)

func ErrClassroomNotFound() error      { return ErrRegistry.New(CodeClassroomNotFound) }
func ErrClassroomAlreadyExists() error { return ErrRegistry.New(CodeClassroomAlreadyExists) }
func ErrClassroomNotActive() error     { return ErrRegistry.New(CodeClassroomNotActive) }
func ErrNotTeacher() error             { return ErrRegistry.New(CodeNotTeacher) }
func ErrNotOwner() error               { return ErrRegistry.New(CodeNotOwner) }
func ErrTeacherAlreadyAdded() error    { return ErrRegistry.New(CodeTeacherAlreadyAdded) }
func ErrCannotRemoveOwner() error      { return ErrRegistry.New(CodeCannotRemoveOwner) }
func ErrInviteCodeInvalid() error      { return ErrRegistry.New(CodeInviteCodeInvalid) }
func ErrInsufficientVault() error      { return ErrRegistry.New(CodeInsufficientVault) }

func ErrInvalidInput(msg string) error {
	return ErrRegistry.NewWithMessage(CodeInvalidInput, msg)
}
