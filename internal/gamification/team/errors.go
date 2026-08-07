package team

import (
	"net/http"

	"github.com/Abraxas-356/neurons/internal/errx"
)

var ErrRegistry = errx.NewRegistry("TEAM")

var (
	CodeNotFound = ErrRegistry.Register(
		"NOT_FOUND",
		errx.TypeNotFound,
		http.StatusNotFound,
		"Team not found",
	)

	CodeAlreadyExists = ErrRegistry.Register(
		"ALREADY_EXISTS",
		errx.TypeBusiness,
		http.StatusConflict,
		"A team with that name already exists in this classroom",
	)

	CodeInvalidInput = ErrRegistry.Register(
		"INVALID_INPUT",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Invalid team data",
	)

	CodeNoMembers = ErrRegistry.Register(
		"NO_MEMBERS",
		errx.TypeBusiness,
		http.StatusConflict,
		"The team has no active members",
	)

	CodeNoStudents = ErrRegistry.Register(
		"NO_STUDENTS",
		errx.TypeBusiness,
		http.StatusConflict,
		"There are no active students to distribute",
	)
)

func ErrNotFound() error      { return ErrRegistry.New(CodeNotFound) }
func ErrAlreadyExists() error { return ErrRegistry.New(CodeAlreadyExists) }
func ErrNoMembers() error     { return ErrRegistry.New(CodeNoMembers) }
func ErrNoStudents() error    { return ErrRegistry.New(CodeNoStudents) }

func ErrInvalidInput(msg string) error {
	return ErrRegistry.NewWithMessage(CodeInvalidInput, msg)
}
