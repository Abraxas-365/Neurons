package qr

import (
	"net/http"

	"github.com/Abraxas-356/neurons/internal/errx"
)

var ErrRegistry = errx.NewRegistry("QR")

var (
	// RN-13: an expired or unknown code is simply invalid.
	CodeInvalid = ErrRegistry.Register(
		"INVALID",
		errx.TypeValidation,
		http.StatusBadRequest,
		"This QR code is invalid or has expired",
	)

	// RN-12: a code from another classroom must never work here.
	CodeWrongClassroom = ErrRegistry.Register(
		"WRONG_CLASSROOM",
		errx.TypeBusiness,
		http.StatusConflict,
		"This QR code belongs to a different classroom",
	)

	CodeWrongKind = ErrRegistry.Register(
		"WRONG_KIND",
		errx.TypeBusiness,
		http.StatusConflict,
		"This QR code cannot be used here",
	)

	// §11.3: a student may claim a given grant code only once.
	CodeAlreadyClaimed = ErrRegistry.Register(
		"ALREADY_CLAIMED",
		errx.TypeConflict,
		http.StatusConflict,
		"You already claimed this code",
	)

	CodeExhausted = ErrRegistry.Register(
		"EXHAUSTED",
		errx.TypeBusiness,
		http.StatusConflict,
		"This QR code has reached its claim limit",
	)

	CodeInvalidInput = ErrRegistry.Register(
		"INVALID_INPUT",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Invalid QR request",
	)

	CodeInternal = ErrRegistry.Register(
		"INTERNAL",
		errx.TypeInternal,
		http.StatusInternalServerError,
		"Could not process the QR code",
	)
)

func ErrInvalid() error        { return ErrRegistry.New(CodeInvalid) }
func ErrWrongClassroom() error { return ErrRegistry.New(CodeWrongClassroom) }
func ErrWrongKind() error      { return ErrRegistry.New(CodeWrongKind) }
func ErrAlreadyClaimed() error { return ErrRegistry.New(CodeAlreadyClaimed) }
func ErrExhausted() error      { return ErrRegistry.New(CodeExhausted) }

func ErrInvalidInput(msg string) error {
	return ErrRegistry.NewWithMessage(CodeInvalidInput, msg)
}

func ErrInternal(cause error) error {
	return errx.Wrap(cause, "qr store failure", errx.TypeInternal)
}
