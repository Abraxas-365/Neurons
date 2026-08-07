package benefit

import (
	"net/http"

	"github.com/Abraxas-356/neurons/internal/errx"
)

var ErrRegistry = errx.NewRegistry("BENEFIT")

var (
	CodeNotFound = ErrRegistry.Register(
		"NOT_FOUND",
		errx.TypeNotFound,
		http.StatusNotFound,
		"Benefit not found",
	)

	CodeInvalidInput = ErrRegistry.Register(
		"INVALID_INPUT",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Invalid benefit data",
	)

	// HU-042: a deactivated benefit accepts no new requests.
	CodeUnavailable = ErrRegistry.Register(
		"UNAVAILABLE",
		errx.TypeBusiness,
		http.StatusConflict,
		"This benefit is not available right now",
	)

	CodeQuotaExhausted = ErrRegistry.Register(
		"QUOTA_EXHAUSTED",
		errx.TypeBusiness,
		http.StatusConflict,
		"This benefit has reached its usage limit",
	)

	CodeStudentQuotaExhausted = ErrRegistry.Register(
		"STUDENT_QUOTA_EXHAUSTED",
		errx.TypeBusiness,
		http.StatusConflict,
		"You have already used this benefit the maximum number of times",
	)
)

func ErrNotFound() error              { return ErrRegistry.New(CodeNotFound) }
func ErrUnavailable() error           { return ErrRegistry.New(CodeUnavailable) }
func ErrQuotaExhausted() error        { return ErrRegistry.New(CodeQuotaExhausted) }
func ErrStudentQuotaExhausted() error { return ErrRegistry.New(CodeStudentQuotaExhausted) }

func ErrInvalidInput(msg string) error {
	return ErrRegistry.NewWithMessage(CodeInvalidInput, msg)
}
