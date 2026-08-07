package reason

import (
	"net/http"

	"github.com/Abraxas-356/neurons/internal/errx"
)

var ErrRegistry = errx.NewRegistry("REASON")

var (
	CodeNotFound = ErrRegistry.Register(
		"NOT_FOUND",
		errx.TypeNotFound,
		http.StatusNotFound,
		"Reason not found",
	)

	CodeInvalidInput = ErrRegistry.Register(
		"INVALID_INPUT",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Invalid reason data",
	)

	CodeInactive = ErrRegistry.Register(
		"INACTIVE",
		errx.TypeBusiness,
		http.StatusConflict,
		"This reason is no longer active",
	)

	CodeWrongScope = ErrRegistry.Register(
		"WRONG_SCOPE",
		errx.TypeBusiness,
		http.StatusConflict,
		"This reason cannot be used for that kind of grant",
	)
)

func ErrNotFound() error   { return ErrRegistry.New(CodeNotFound) }
func ErrInactive() error   { return ErrRegistry.New(CodeInactive) }
func ErrWrongScope() error { return ErrRegistry.New(CodeWrongScope) }

func ErrInvalidInput(msg string) error {
	return ErrRegistry.NewWithMessage(CodeInvalidInput, msg)
}
