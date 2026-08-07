package ledger

import (
	"net/http"

	"github.com/Abraxas-356/neurons/internal/errx"
)

var ErrRegistry = errx.NewRegistry("LEDGER")

var (
	CodeNotFound = ErrRegistry.Register(
		"NOT_FOUND",
		errx.TypeNotFound,
		http.StatusNotFound,
		"Transaction not found",
	)

	CodeInvalidInput = ErrRegistry.Register(
		"INVALID_INPUT",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Invalid transaction data",
	)

	// RN-08: the amount must be a positive whole number of neurons.
	CodeInvalidAmount = ErrRegistry.Register(
		"INVALID_AMOUNT",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Amount must be a positive whole number",
	)

	// RN-05: the vault cannot issue more than it holds.
	CodeInsufficientVault = ErrRegistry.Register(
		"INSUFFICIENT_VAULT",
		errx.TypeBusiness,
		http.StatusConflict,
		"The classroom vault does not have enough neurons",
	)

	// RN-04: a student balance can never go negative.
	CodeInsufficientBalance = ErrRegistry.Register(
		"INSUFFICIENT_BALANCE",
		errx.TypeBusiness,
		http.StatusConflict,
		"Not enough neurons for this operation",
	)

	// RN-10: grants require a reason.
	CodeReasonRequired = ErrRegistry.Register(
		"REASON_REQUIRED",
		errx.TypeValidation,
		http.StatusBadRequest,
		"A reason is required to grant neurons",
	)

	// RN-11: returns require a concept.
	CodeConceptRequired = ErrRegistry.Register(
		"CONCEPT_REQUIRED",
		errx.TypeValidation,
		http.StatusBadRequest,
		"A benefit or concept is required to return neurons",
	)

	// RN-17: a closed classroom is frozen.
	CodeClassroomClosed = ErrRegistry.Register(
		"CLASSROOM_CLOSED",
		errx.TypeBusiness,
		http.StatusConflict,
		"This classroom no longer accepts transactions",
	)

	// RN-16: pending or withdrawn students cannot transact.
	CodeStudentInactive = ErrRegistry.Register(
		"STUDENT_INACTIVE",
		errx.TypeBusiness,
		http.StatusConflict,
		"This student is not active in the classroom",
	)

	CodeNoRecipients = ErrRegistry.Register(
		"NO_RECIPIENTS",
		errx.TypeValidation,
		http.StatusBadRequest,
		"No valid recipients for this operation",
	)

	// RN-15: reversals cannot be reversed and an entry is undone only once.
	CodeAlreadyReversed = ErrRegistry.Register(
		"ALREADY_REVERSED",
		errx.TypeBusiness,
		http.StatusConflict,
		"This transaction was already reversed",
	)

	CodeNotReversible = ErrRegistry.Register(
		"NOT_REVERSIBLE",
		errx.TypeBusiness,
		http.StatusConflict,
		"This kind of transaction cannot be reversed",
	)

	// §11.9: unusually large grants must be explicitly confirmed.
	CodeConfirmationRequired = ErrRegistry.Register(
		"CONFIRMATION_REQUIRED",
		errx.TypeBusiness,
		http.StatusConflict,
		"This amount is unusually large and must be confirmed",
	)

	// §11.3: the idempotency key was already used for another movement.
	CodeDuplicate = ErrRegistry.Register(
		"DUPLICATE",
		errx.TypeConflict,
		http.StatusConflict,
		"This operation was already processed",
	)

	CodeForbidden = ErrRegistry.Register(
		"FORBIDDEN",
		errx.TypeAuthorization,
		http.StatusForbidden,
		"You cannot access this transaction",
	)
)

func ErrNotFound() error             { return ErrRegistry.New(CodeNotFound) }
func ErrInsufficientVault() error    { return ErrRegistry.New(CodeInsufficientVault) }
func ErrInsufficientBalance() error  { return ErrRegistry.New(CodeInsufficientBalance) }
func ErrReasonRequired() error       { return ErrRegistry.New(CodeReasonRequired) }
func ErrConceptRequired() error      { return ErrRegistry.New(CodeConceptRequired) }
func ErrClassroomClosed() error      { return ErrRegistry.New(CodeClassroomClosed) }
func ErrStudentInactive() error      { return ErrRegistry.New(CodeStudentInactive) }
func ErrNoRecipients() error         { return ErrRegistry.New(CodeNoRecipients) }
func ErrAlreadyReversed() error      { return ErrRegistry.New(CodeAlreadyReversed) }
func ErrNotReversible() error        { return ErrRegistry.New(CodeNotReversible) }
func ErrConfirmationRequired() error { return ErrRegistry.New(CodeConfirmationRequired) }
func ErrDuplicate() error            { return ErrRegistry.New(CodeDuplicate) }
func ErrForbidden() error            { return ErrRegistry.New(CodeForbidden) }

func ErrInvalidAmount(msg string) error {
	return ErrRegistry.NewWithMessage(CodeInvalidAmount, msg)
}

func ErrInvalidInput(msg string) error {
	return ErrRegistry.NewWithMessage(CodeInvalidInput, msg)
}
