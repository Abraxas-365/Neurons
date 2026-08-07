package ledger

import (
	"time"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

// Type is the kind of neuron movement recorded in the ledger (RN-09).
type Type string

const (
	// TypeGrant moves neurons from the classroom vault to a student.
	TypeGrant Type = "GRANT"
	// TypeRedemption moves neurons from a student back to the vault (RN-03).
	TypeRedemption Type = "REDEMPTION"
	// TypeGrantReversal undoes a GRANT (RN-15).
	TypeGrantReversal Type = "GRANT_REVERSAL"
	// TypeRedemptionReversal undoes a REDEMPTION (RN-15).
	TypeRedemptionReversal Type = "REDEMPTION_REVERSAL"
	// TypeVaultTopup refills the classroom vault (decision 15.8).
	TypeVaultTopup Type = "VAULT_TOPUP"
	// TypeAdjustment is a manual correction by a teacher.
	TypeAdjustment Type = "ADJUSTMENT"
)

// IsReversal reports whether the type is itself an undo entry.
func (t Type) IsReversal() bool {
	return t == TypeGrantReversal || t == TypeRedemptionReversal
}

// ReversalType returns the entry type that undoes t.
func (t Type) ReversalType() (Type, bool) {
	switch t {
	case TypeGrant:
		return TypeGrantReversal, true
	case TypeRedemption:
		return TypeRedemptionReversal, true
	default:
		return "", false
	}
}

// Channel records how the movement was initiated (§11 fraud prevention).
type Channel string

const (
	ChannelQR     Channel = "QR"
	ChannelManual Channel = "MANUAL"
	ChannelBulk   Channel = "BULK"
	ChannelSystem Channel = "SYSTEM"
)

// Status marks whether an entry is still in force.
type Status string

const (
	StatusApplied  Status = "APPLIED"
	StatusReversed Status = "REVERSED"
)

// BatchType groups entries produced by one logical teacher action.
type BatchType string

const (
	BatchTeamGrant  BatchType = "TEAM_GRANT"
	BatchMultiGrant BatchType = "MULTI_GRANT"
)

// Transaction is one immutable ledger entry. Rows are never deleted; a mistake
// is undone by writing a compensating reversal entry (RN-15).
type Transaction struct {
	ID          kernel.LedgerID    `json:"id" db:"id"`
	Code        string             `json:"code" db:"code"`
	ClassroomID kernel.ClassroomID `json:"classroom_id" db:"classroom_id"`
	Type        Type               `json:"type" db:"type"`

	EnrollmentID *kernel.EnrollmentID `json:"enrollment_id" db:"enrollment_id"`
	TeamID       *kernel.TeamID       `json:"team_id" db:"team_id"`
	BatchID      *string              `json:"batch_id" db:"batch_id"`

	// Amount is always positive; Type carries the direction.
	Amount int64 `json:"amount" db:"amount"`

	// RN-10: a grant always carries a reason.
	ReasonID   *kernel.ReasonID `json:"reason_id" db:"reason_id"`
	ReasonText *string          `json:"reason_text" db:"reason_text"`
	// RN-11: a return always carries a concept.
	BenefitID   *kernel.BenefitID `json:"benefit_id" db:"benefit_id"`
	BenefitText *string           `json:"benefit_text" db:"benefit_text"`

	// Balance snapshots taken inside the same transaction (§4.6 auditability).
	StudentBalanceBefore *int64 `json:"student_balance_before" db:"student_balance_before"`
	StudentBalanceAfter  *int64 `json:"student_balance_after" db:"student_balance_after"`
	VaultBalanceBefore   *int64 `json:"vault_balance_before" db:"vault_balance_before"`
	VaultBalanceAfter    *int64 `json:"vault_balance_after" db:"vault_balance_after"`

	Channel Channel `json:"channel" db:"channel"`
	Status  Status  `json:"status" db:"status"`

	ReversesTransactionID   *kernel.LedgerID `json:"reverses_transaction_id" db:"reverses_transaction_id"`
	ReversedByTransactionID *kernel.LedgerID `json:"reversed_by_transaction_id" db:"reversed_by_transaction_id"`

	IdempotencyKey *string       `json:"-" db:"idempotency_key"`
	PerformedBy    kernel.UserID `json:"performed_by" db:"performed_by"`
	DeviceInfo     *string       `json:"-" db:"device_info"`
	Notes          *string       `json:"notes" db:"notes"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`

	// Joined display fields.
	StudentName   *string `json:"student_name,omitempty" db:"student_name"`
	TeamName      *string `json:"team_name,omitempty" db:"team_name"`
	PerformerName *string `json:"performer_name,omitempty" db:"performer_name"`
}

// IsCredit reports whether the entry increases the student's balance.
func (t *Transaction) IsCredit() bool {
	return t.Type == TypeGrant || t.Type == TypeRedemptionReversal
}

// CanBeReversed implements RN-15 and decision 15.4: a teacher may undo any
// applied movement at any time, but never an undo of an undo.
func (t *Transaction) CanBeReversed() bool {
	if t.Status != StatusApplied {
		return false
	}
	_, ok := t.Type.ReversalType()
	return ok
}

// Batch groups the entries of a team or multi-student grant (HU-033, HU-052).
type Batch struct {
	ID               string             `json:"id" db:"id"`
	ClassroomID      kernel.ClassroomID `json:"classroom_id" db:"classroom_id"`
	Type             BatchType          `json:"type" db:"type"`
	TeamID           *kernel.TeamID     `json:"team_id" db:"team_id"`
	AmountPerStudent int                `json:"amount_per_student" db:"amount_per_student"`
	RecipientCount   int                `json:"recipient_count" db:"recipient_count"`
	TotalAmount      int64              `json:"total_amount" db:"total_amount"`
	ReasonID         *kernel.ReasonID   `json:"reason_id" db:"reason_id"`
	ReasonText       *string            `json:"reason_text" db:"reason_text"`
	PerformedBy      kernel.UserID      `json:"performed_by" db:"performed_by"`
	CreatedAt        time.Time          `json:"created_at" db:"created_at"`
}

// --- Request DTOs ---

// GrantRequest hands neurons to one or more students (flow A / D, HU-051).
type GrantRequest struct {
	EnrollmentIDs []kernel.EnrollmentID `json:"enrollment_ids"`
	Amount        int64                 `json:"amount"`
	ReasonID      *kernel.ReasonID      `json:"reason_id"`
	ReasonText    *string               `json:"reason_text"`
	Notes         *string               `json:"notes"`
	Channel       Channel               `json:"channel"`
	// IdempotencyKey guards against double submits and duplicate QR scans (§11.3).
	IdempotencyKey *string `json:"idempotency_key"`
	// Confirmed must be true when the amount exceeds the reconfirm threshold (§11.9).
	Confirmed bool `json:"confirmed"`
}

// TeamGrantRequest gives Amount to EACH active member of the team (RN-07).
type TeamGrantRequest struct {
	TeamID         kernel.TeamID    `json:"team_id"`
	Amount         int64            `json:"amount"`
	ReasonID       *kernel.ReasonID `json:"reason_id"`
	ReasonText     *string          `json:"reason_text"`
	Notes          *string          `json:"notes"`
	IdempotencyKey *string          `json:"idempotency_key"`
	Confirmed      bool             `json:"confirmed"`
}

// RedeemRequest returns neurons from a student to the teacher (flow C, RN-03).
type RedeemRequest struct {
	EnrollmentID kernel.EnrollmentID `json:"enrollment_id"`
	BenefitID    *kernel.BenefitID   `json:"benefit_id"`
	BenefitText  *string             `json:"benefit_text"`
	// Amount is required for free-amount benefits and for benefit-less returns.
	Amount         *int    `json:"amount"`
	Notes          *string `json:"notes"`
	Channel        Channel `json:"channel"`
	IdempotencyKey *string `json:"idempotency_key"`
}

// TopupRequest refills the classroom vault (decision 15.8).
type TopupRequest struct {
	Amount int64   `json:"amount"`
	Notes  *string `json:"notes"`
}

// ReverseRequest undoes an applied entry (HU-092).
type ReverseRequest struct {
	Reason *string `json:"reason"`
}

// HistoryFilter drives the ledger queries (HU-084).
type HistoryFilter struct {
	EnrollmentID *kernel.EnrollmentID
	TeamID       *kernel.TeamID
	Type         *Type
	ReasonID     *kernel.ReasonID
	BenefitID    *kernel.BenefitID
	From         *time.Time
	To           *time.Time
}

// --- Response DTOs ---

// GrantResult is what the teacher's screen shows after a grant (HU-051).
type GrantResult struct {
	BatchID      *string       `json:"batch_id,omitempty"`
	Transactions []Transaction `json:"transactions"`
	Recipients   int           `json:"recipients"`
	AmountEach   int64         `json:"amount_each"`
	TotalAmount  int64         `json:"total_amount"`
	VaultBalance int64         `json:"vault_balance"`
}

// ClassroomStats powers the teacher dashboard (§13).
type ClassroomStats struct {
	VaultBalance      int64 `json:"vault_balance" db:"vault_balance"`
	UnlimitedIssuance bool  `json:"unlimited_issuance" db:"unlimited_issuance"`
	TotalGranted      int64 `json:"total_granted" db:"total_granted"`
	TotalRedeemed     int64 `json:"total_redeemed" db:"total_redeemed"`
	// InCirculation is what students currently hold.
	InCirculation  int64 `json:"in_circulation" db:"in_circulation"`
	ActiveStudents int   `json:"active_students" db:"active_students"`
	Transactions   int   `json:"transactions" db:"transactions"`
}

// RankingEntry is one row of the optional public ranking (decision 15.7).
type RankingEntry struct {
	Position      int                 `json:"position"`
	EnrollmentID  kernel.EnrollmentID `json:"enrollment_id"`
	StudentName   string              `json:"student_name"`
	TotalReceived int64               `json:"total_received"`
	MedalCount    int                 `json:"medal_count"`
}

// ReasonUsage powers the "most used reasons" report (§13).
type ReasonUsage struct {
	ReasonID    *kernel.ReasonID `json:"reason_id" db:"reason_id"`
	ReasonName  string           `json:"reason_name" db:"reason_name"`
	Uses        int              `json:"uses" db:"uses"`
	TotalAmount int64            `json:"total_amount" db:"total_amount"`
}
