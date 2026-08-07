package ledger

import (
	"context"

	"github.com/Abraxas-356/neurons/internal/gamification/enrollment"
	"github.com/Abraxas-356/neurons/internal/kernel"
)

// GrantOp is the fully validated instruction the repository executes atomically.
// The service resolves reasons, recipients and limits; the repository only
// performs the money movement and writes the audit rows.
type GrantOp struct {
	ClassroomID kernel.ClassroomID
	// Recipients are active enrollments; each one receives AmountEach (RN-07).
	Recipients []kernel.EnrollmentID
	AmountEach int64
	ReasonID   *kernel.ReasonID
	ReasonText *string
	Notes      *string
	Channel    Channel
	// Batch is set when more than one recipient is involved.
	Batch          *Batch
	IdempotencyKey *string
	PerformedBy    kernel.UserID
	DeviceInfo     *string
}

// RedeemOp returns neurons from one student to the classroom vault.
type RedeemOp struct {
	ClassroomID    kernel.ClassroomID
	EnrollmentID   kernel.EnrollmentID
	Amount         int64
	BenefitID      *kernel.BenefitID
	BenefitText    *string
	Notes          *string
	Channel        Channel
	IdempotencyKey *string
	PerformedBy    kernel.UserID
	DeviceInfo     *string
}

// Repository owns every neuron movement. All balance mutations happen here,
// inside a single database transaction with row locks, so the vault, the
// student balance and the ledger can never drift apart (RN-09).
type Repository interface {
	// Grant debits the vault and credits each recipient atomically. It fails
	// with ErrInsufficientVault when the vault cannot cover the total (RN-05).
	Grant(ctx context.Context, op GrantOp) ([]Transaction, error)

	// Redeem debits the student and credits the vault atomically. It fails with
	// ErrInsufficientBalance when the student cannot cover it (RN-04).
	Redeem(ctx context.Context, op RedeemOp) (*Transaction, error)

	// Topup adds neurons to the vault and records the movement (decision 15.8).
	Topup(ctx context.Context, classroomID kernel.ClassroomID, amount int64, notes *string, performedBy kernel.UserID) (*Transaction, error)

	// Reverse writes the compensating entry for an applied transaction and
	// restores both balances atomically (RN-15).
	Reverse(ctx context.Context, id kernel.LedgerID, performedBy kernel.UserID, note *string) (*Transaction, error)

	GetByID(ctx context.Context, id kernel.LedgerID) (*Transaction, error)
	// FindByIdempotencyKey lets the service return the original result instead
	// of duplicating a movement (§11.3).
	FindByIdempotencyKey(ctx context.Context, classroomID kernel.ClassroomID, key string) ([]Transaction, error)

	History(ctx context.Context, classroomID kernel.ClassroomID, filter HistoryFilter, opts kernel.PaginationOptions) (kernel.Paginated[Transaction], error)
	StudentHistory(ctx context.Context, enrollmentID kernel.EnrollmentID, opts kernel.PaginationOptions) (kernel.Paginated[Transaction], error)

	Stats(ctx context.Context, classroomID kernel.ClassroomID) (*ClassroomStats, error)
	Ranking(ctx context.Context, classroomID kernel.ClassroomID, limit int) ([]RankingEntry, error)
	ReasonUsage(ctx context.Context, classroomID kernel.ClassroomID) ([]ReasonUsage, error)

	// GetBatch returns the grouped view of a team or multi-student grant.
	GetBatch(ctx context.Context, batchID string) (*Batch, []Transaction, error)
}

// EnrollmentReader is the slice of the enrollment module the ledger needs to
// validate recipients without depending on its service.
type EnrollmentReader interface {
	GetByID(ctx context.Context, id kernel.EnrollmentID) (*enrollment.Enrollment, error)
	ListActiveByTeam(ctx context.Context, teamID kernel.TeamID) ([]enrollment.Enrollment, error)
	ListActiveByIDs(ctx context.Context, classroomID kernel.ClassroomID, ids []kernel.EnrollmentID) ([]enrollment.Enrollment, error)
}
