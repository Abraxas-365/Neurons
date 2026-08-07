package medal

import (
	"net/http"

	"github.com/Abraxas-356/neurons/internal/errx"
)

var ErrRegistry = errx.NewRegistry("MEDAL")

var (
	CodeNotFound = ErrRegistry.Register(
		"NOT_FOUND",
		errx.TypeNotFound,
		http.StatusNotFound,
		"Medal not found",
	)

	CodeAwardNotFound = ErrRegistry.Register(
		"AWARD_NOT_FOUND",
		errx.TypeNotFound,
		http.StatusNotFound,
		"Medal award not found",
	)

	CodeInvalidInput = ErrRegistry.Register(
		"INVALID_INPUT",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Invalid medal data",
	)

	CodeUnavailable = ErrRegistry.Register(
		"UNAVAILABLE",
		errx.TypeBusiness,
		http.StatusConflict,
		"This medal is not available right now",
	)

	CodeQuotaExhausted = ErrRegistry.Register(
		"QUOTA_EXHAUSTED",
		errx.TypeBusiness,
		http.StatusConflict,
		"This medal has reached its award limit",
	)

	// Decision 15.5: non-repeatable medals may only be awarded once per target.
	CodeAlreadyAwarded = ErrRegistry.Register(
		"ALREADY_AWARDED",
		errx.TypeBusiness,
		http.StatusConflict,
		"This medal was already awarded to that recipient",
	)

	CodeAlreadyRevoked = ErrRegistry.Register(
		"ALREADY_REVOKED",
		errx.TypeBusiness,
		http.StatusConflict,
		"This medal award was already revoked",
	)

	CodeWrongType = ErrRegistry.Register(
		"WRONG_TYPE",
		errx.TypeBusiness,
		http.StatusBadRequest,
		"This medal cannot be awarded to that kind of recipient",
	)
)

func ErrNotFound() error       { return ErrRegistry.New(CodeNotFound) }
func ErrAwardNotFound() error  { return ErrRegistry.New(CodeAwardNotFound) }
func ErrUnavailable() error    { return ErrRegistry.New(CodeUnavailable) }
func ErrQuotaExhausted() error { return ErrRegistry.New(CodeQuotaExhausted) }
func ErrAlreadyAwarded() error { return ErrRegistry.New(CodeAlreadyAwarded) }
func ErrAlreadyRevoked() error { return ErrRegistry.New(CodeAlreadyRevoked) }
func ErrWrongType() error      { return ErrRegistry.New(CodeWrongType) }

func ErrInvalidInput(msg string) error {
	return ErrRegistry.NewWithMessage(CodeInvalidInput, msg)
}
