package ledgerinfra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Abraxas-356/neurons/internal/errx"
	"github.com/Abraxas-356/neurons/internal/gamification/dbx"
	"github.com/Abraxas-356/neurons/internal/gamification/ledger"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type PostgresLedgerRepository struct {
	db *sqlx.DB
}

func NewPostgresLedgerRepository(db *sqlx.DB) ledger.Repository {
	return &PostgresLedgerRepository{db: db}
}

const txColumns = `id, code, classroom_id, type, enrollment_id, team_id, batch_id, amount,
	reason_id, reason_text, benefit_id, benefit_text,
	student_balance_before, student_balance_after, vault_balance_before, vault_balance_after,
	channel, status, reverses_transaction_id, reversed_by_transaction_id,
	idempotency_key, performed_by, device_info, notes, created_at`

const txInsertValues = `:id, :code, :classroom_id, :type, :enrollment_id, :team_id, :batch_id, :amount,
	:reason_id, :reason_text, :benefit_id, :benefit_text,
	:student_balance_before, :student_balance_after, :vault_balance_before, :vault_balance_after,
	:channel, :status, :reverses_transaction_id, :reversed_by_transaction_id,
	:idempotency_key, :performed_by, :device_info, :notes, :created_at`

// vaultState is the locked snapshot of a classroom vault.
type vaultState struct {
	Balance   int64  `db:"vault_balance"`
	Unlimited bool   `db:"unlimited_issuance"`
	Status    string `db:"status"`
}

// lockVault takes a row lock on the classroom so concurrent grants cannot
// oversell the vault (RN-05).
func lockVault(ctx context.Context, tx *sqlx.Tx, id kernel.ClassroomID) (*vaultState, error) {
	var v vaultState
	query := `SELECT vault_balance, unlimited_issuance, status FROM classrooms WHERE id = $1 FOR UPDATE`
	if err := tx.GetContext(ctx, &v, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ledger.ErrNotFound()
		}
		return nil, errx.Wrap(err, "lock classroom vault", errx.TypeInternal)
	}
	return &v, nil
}

// lockedEnrollment is the locked snapshot of a student's balance.
type lockedEnrollment struct {
	ID      kernel.EnrollmentID `db:"id"`
	Balance int64               `db:"balance"`
	Status  string              `db:"status"`
}

// lockEnrollments locks the recipients in a deterministic order (by id) so two
// concurrent batches can never deadlock each other.
func lockEnrollments(
	ctx context.Context,
	tx *sqlx.Tx,
	classroomID kernel.ClassroomID,
	ids []kernel.EnrollmentID,
) (map[kernel.EnrollmentID]*lockedEnrollment, error) {
	raw := make([]string, 0, len(ids))
	for _, id := range ids {
		raw = append(raw, id.String())
	}

	rows := []lockedEnrollment{}
	query := `SELECT id, balance, status FROM enrollments
	          WHERE classroom_id = $1 AND id = ANY($2)
	          ORDER BY id
	          FOR UPDATE`
	if err := tx.SelectContext(ctx, &rows, query, classroomID, pq.Array(raw)); err != nil {
		return nil, errx.Wrap(err, "lock enrollments", errx.TypeInternal)
	}
	if len(rows) != len(ids) {
		return nil, ledger.ErrNoRecipients()
	}

	locked := make(map[kernel.EnrollmentID]*lockedEnrollment, len(rows))
	for i := range rows {
		if rows[i].Status != "ACTIVE" {
			return nil, ledger.ErrStudentInactive()
		}
		locked[rows[i].ID] = &rows[i]
	}
	return locked, nil
}

// nextCode produces the short human-readable transaction code shown in the UI.
func nextCode(t ledger.Type) string {
	prefix := "TX"
	switch t {
	case ledger.TypeGrant:
		prefix = "GR"
	case ledger.TypeRedemption:
		prefix = "RD"
	case ledger.TypeGrantReversal, ledger.TypeRedemptionReversal:
		prefix = "RV"
	case ledger.TypeVaultTopup:
		prefix = "TU"
	case ledger.TypeAdjustment:
		prefix = "AD"
	}
	return fmt.Sprintf("%s-%s", prefix, strings.ToUpper(uuid.NewString()[:8]))
}

func isDuplicateKey(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

func insertTx(ctx context.Context, tx *sqlx.Tx, t *ledger.Transaction) error {
	query := `INSERT INTO transactions (` + txColumns + `) VALUES (` + txInsertValues + `)`
	if _, err := tx.NamedExecContext(ctx, query, t); err != nil {
		if isDuplicateKey(err) {
			return ledger.ErrDuplicate()
		}
		return errx.Wrap(err, "insert transaction", errx.TypeInternal)
	}
	return nil
}

// Grant moves AmountEach from the vault to every recipient in one database
// transaction. Either all recipients are paid or none are (RN-07, RN-09).
func (r *PostgresLedgerRepository) Grant(ctx context.Context, op ledger.GrantOp) ([]ledger.Transaction, error) {
	if len(op.Recipients) == 0 {
		return nil, ledger.ErrNoRecipients()
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errx.Wrap(err, "begin grant tx", errx.TypeInternal)
	}
	defer tx.Rollback()

	vault, err := lockVault(ctx, tx, op.ClassroomID)
	if err != nil {
		return nil, err
	}
	if vault.Status != "ACTIVE" {
		return nil, ledger.ErrClassroomClosed()
	}

	total := op.AmountEach * int64(len(op.Recipients))
	if !vault.Unlimited && vault.Balance < total {
		return nil, ledger.ErrInsufficientVault()
	}

	locked, err := lockEnrollments(ctx, tx, op.ClassroomID, op.Recipients)
	if err != nil {
		return nil, err
	}

	if op.Batch != nil {
		if err := insertBatch(ctx, tx, op.Batch); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	vaultBefore := vault.Balance
	results := make([]ledger.Transaction, 0, len(op.Recipients))

	for _, id := range op.Recipients {
		e := locked[id]
		studentBefore := e.Balance
		studentAfter := studentBefore + op.AmountEach

		// Vault snapshots are per-entry so the ledger reads as a running balance.
		vaultAfter := vaultBefore
		if !vault.Unlimited {
			vaultAfter = vaultBefore - op.AmountEach
		}

		enrollmentID := id
		var batchID *string
		if op.Batch != nil {
			batchID = &op.Batch.ID
		}

		t := ledger.Transaction{
			ID:                   kernel.NewLedgerID(uuid.NewString()),
			Code:                 nextCode(ledger.TypeGrant),
			ClassroomID:          op.ClassroomID,
			Type:                 ledger.TypeGrant,
			EnrollmentID:         &enrollmentID,
			BatchID:              batchID,
			Amount:               op.AmountEach,
			ReasonID:             op.ReasonID,
			ReasonText:           op.ReasonText,
			StudentBalanceBefore: &studentBefore,
			StudentBalanceAfter:  &studentAfter,
			VaultBalanceBefore:   &vaultBefore,
			VaultBalanceAfter:    &vaultAfter,
			Channel:              op.Channel,
			Status:               ledger.StatusApplied,
			IdempotencyKey:       op.IdempotencyKey,
			PerformedBy:          op.PerformedBy,
			DeviceInfo:           op.DeviceInfo,
			Notes:                op.Notes,
			CreatedAt:            now,
		}
		if op.Batch != nil && op.Batch.TeamID != nil {
			t.TeamID = op.Batch.TeamID
		}

		if err := insertTx(ctx, tx, &t); err != nil {
			return nil, err
		}

		if err := creditStudent(ctx, tx, id, op.AmountEach); err != nil {
			return nil, err
		}

		vaultBefore = vaultAfter
		results = append(results, t)

		// Only the first entry of a batch may carry the idempotency key; the
		// unique index is per classroom.
		op.IdempotencyKey = nil
	}

	if err := debitVault(ctx, tx, op.ClassroomID, total, vault.Unlimited); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errx.Wrap(err, "commit grant tx", errx.TypeInternal)
	}
	return results, nil
}

func creditStudent(ctx context.Context, tx *sqlx.Tx, id kernel.EnrollmentID, amount int64) error {
	query := `UPDATE enrollments
	          SET balance = balance + $2,
	              total_received = total_received + $2,
	              last_activity_at = CURRENT_TIMESTAMP
	          WHERE id = $1`
	if _, err := tx.ExecContext(ctx, query, id, amount); err != nil {
		return errx.Wrap(err, "credit student", errx.TypeInternal)
	}
	return nil
}

// debitVault subtracts from the vault unless issuance is unlimited, in which
// case only the granted counter moves.
func debitVault(ctx context.Context, tx *sqlx.Tx, id kernel.ClassroomID, total int64, unlimited bool) error {
	query := `UPDATE classrooms
	          SET vault_balance = CASE WHEN $3 THEN vault_balance ELSE vault_balance - $2 END,
	              total_granted = total_granted + $2
	          WHERE id = $1`
	if _, err := tx.ExecContext(ctx, query, id, total, unlimited); err != nil {
		return errx.Wrap(err, "debit vault", errx.TypeInternal)
	}
	return nil
}

func insertBatch(ctx context.Context, tx *sqlx.Tx, b *ledger.Batch) error {
	query := `INSERT INTO transaction_batches
	          (id, classroom_id, type, team_id, amount_per_student, recipient_count, total_amount,
	           reason_id, reason_text, performed_by, created_at)
	          VALUES (:id, :classroom_id, :type, :team_id, :amount_per_student, :recipient_count, :total_amount,
	                  :reason_id, :reason_text, :performed_by, :created_at)`
	if _, err := tx.NamedExecContext(ctx, query, b); err != nil {
		return errx.Wrap(err, "insert batch", errx.TypeInternal)
	}
	return nil
}

// Redeem moves neurons from a student back to the vault (RN-03, RN-04).
func (r *PostgresLedgerRepository) Redeem(ctx context.Context, op ledger.RedeemOp) (*ledger.Transaction, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errx.Wrap(err, "begin redeem tx", errx.TypeInternal)
	}
	defer tx.Rollback()

	vault, err := lockVault(ctx, tx, op.ClassroomID)
	if err != nil {
		return nil, err
	}
	if vault.Status != "ACTIVE" {
		return nil, ledger.ErrClassroomClosed()
	}

	locked, err := lockEnrollments(ctx, tx, op.ClassroomID, []kernel.EnrollmentID{op.EnrollmentID})
	if err != nil {
		return nil, err
	}
	e := locked[op.EnrollmentID]

	if e.Balance < op.Amount {
		return nil, ledger.ErrInsufficientBalance()
	}

	studentBefore := e.Balance
	studentAfter := studentBefore - op.Amount
	vaultBefore := vault.Balance
	vaultAfter := vaultBefore + op.Amount

	enrollmentID := op.EnrollmentID
	t := ledger.Transaction{
		ID:                   kernel.NewLedgerID(uuid.NewString()),
		Code:                 nextCode(ledger.TypeRedemption),
		ClassroomID:          op.ClassroomID,
		Type:                 ledger.TypeRedemption,
		EnrollmentID:         &enrollmentID,
		Amount:               op.Amount,
		BenefitID:            op.BenefitID,
		BenefitText:          op.BenefitText,
		StudentBalanceBefore: &studentBefore,
		StudentBalanceAfter:  &studentAfter,
		VaultBalanceBefore:   &vaultBefore,
		VaultBalanceAfter:    &vaultAfter,
		Channel:              op.Channel,
		Status:               ledger.StatusApplied,
		IdempotencyKey:       op.IdempotencyKey,
		PerformedBy:          op.PerformedBy,
		DeviceInfo:           op.DeviceInfo,
		Notes:                op.Notes,
		CreatedAt:            time.Now(),
	}

	if err := insertTx(ctx, tx, &t); err != nil {
		return nil, err
	}

	debit := `UPDATE enrollments
	          SET balance = balance - $2,
	              total_returned = total_returned + $2,
	              last_activity_at = CURRENT_TIMESTAMP
	          WHERE id = $1`
	if _, err := tx.ExecContext(ctx, debit, op.EnrollmentID, op.Amount); err != nil {
		return nil, errx.Wrap(err, "debit student", errx.TypeInternal)
	}

	credit := `UPDATE classrooms
	           SET vault_balance = vault_balance + $2, total_redeemed = total_redeemed + $2
	           WHERE id = $1`
	if _, err := tx.ExecContext(ctx, credit, op.ClassroomID, op.Amount); err != nil {
		return nil, errx.Wrap(err, "credit vault", errx.TypeInternal)
	}

	if op.BenefitID != nil {
		bump := `UPDATE benefits SET uses_count = uses_count + 1 WHERE id = $1`
		if _, err := tx.ExecContext(ctx, bump, *op.BenefitID); err != nil {
			return nil, errx.Wrap(err, "bump benefit uses", errx.TypeInternal)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, errx.Wrap(err, "commit redeem tx", errx.TypeInternal)
	}
	return &t, nil
}

func (r *PostgresLedgerRepository) Topup(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	amount int64,
	notes *string,
	performedBy kernel.UserID,
) (*ledger.Transaction, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errx.Wrap(err, "begin topup tx", errx.TypeInternal)
	}
	defer tx.Rollback()

	vault, err := lockVault(ctx, tx, classroomID)
	if err != nil {
		return nil, err
	}

	vaultBefore := vault.Balance
	vaultAfter := vaultBefore + amount

	t := ledger.Transaction{
		ID:                 kernel.NewLedgerID(uuid.NewString()),
		Code:               nextCode(ledger.TypeVaultTopup),
		ClassroomID:        classroomID,
		Type:               ledger.TypeVaultTopup,
		Amount:             amount,
		VaultBalanceBefore: &vaultBefore,
		VaultBalanceAfter:  &vaultAfter,
		Channel:            ledger.ChannelManual,
		Status:             ledger.StatusApplied,
		PerformedBy:        performedBy,
		Notes:              notes,
		CreatedAt:          time.Now(),
	}

	if err := insertTx(ctx, tx, &t); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE classrooms SET vault_balance = vault_balance + $2 WHERE id = $1`, classroomID, amount); err != nil {
		return nil, errx.Wrap(err, "topup vault", errx.TypeInternal)
	}

	if err := tx.Commit(); err != nil {
		return nil, errx.Wrap(err, "commit topup tx", errx.TypeInternal)
	}
	return &t, nil
}

// Reverse writes the compensating entry and restores both balances (RN-15).
// The original row is kept and marked REVERSED; nothing is ever deleted.
func (r *PostgresLedgerRepository) Reverse(
	ctx context.Context,
	id kernel.LedgerID,
	performedBy kernel.UserID,
	note *string,
) (*ledger.Transaction, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errx.Wrap(err, "begin reverse tx", errx.TypeInternal)
	}
	defer tx.Rollback()

	var orig ledger.Transaction
	if err := tx.GetContext(ctx, &orig, `SELECT `+txColumns+` FROM transactions WHERE id = $1 FOR UPDATE`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ledger.ErrNotFound()
		}
		return nil, errx.Wrap(err, "lock transaction", errx.TypeInternal)
	}
	if orig.Status == ledger.StatusReversed {
		return nil, ledger.ErrAlreadyReversed()
	}
	reversalType, ok := orig.Type.ReversalType()
	if !ok {
		return nil, ledger.ErrNotReversible()
	}
	if orig.EnrollmentID == nil {
		return nil, ledger.ErrNotReversible()
	}

	vault, err := lockVault(ctx, tx, orig.ClassroomID)
	if err != nil {
		return nil, err
	}
	locked, err := lockEnrollmentForReversal(ctx, tx, *orig.EnrollmentID)
	if err != nil {
		return nil, err
	}

	studentBefore := locked.Balance
	vaultBefore := vault.Balance

	var studentAfter, vaultAfter int64
	if orig.Type == ledger.TypeGrant {
		// Undoing a grant takes the neurons back from the student.
		if studentBefore < orig.Amount {
			return nil, ledger.ErrInsufficientBalance()
		}
		studentAfter = studentBefore - orig.Amount
		vaultAfter = vaultBefore
		if !vault.Unlimited {
			vaultAfter = vaultBefore + orig.Amount
		}
		undo := `UPDATE enrollments
		         SET balance = balance - $2, total_received = total_received - $2
		         WHERE id = $1`
		if _, err := tx.ExecContext(ctx, undo, *orig.EnrollmentID, orig.Amount); err != nil {
			return nil, errx.Wrap(err, "undo student credit", errx.TypeInternal)
		}
		undoVault := `UPDATE classrooms
		              SET vault_balance = CASE WHEN $3 THEN vault_balance ELSE vault_balance + $2 END,
		                  total_granted = total_granted - $2
		              WHERE id = $1`
		if _, err := tx.ExecContext(ctx, undoVault, orig.ClassroomID, orig.Amount, vault.Unlimited); err != nil {
			return nil, errx.Wrap(err, "undo vault debit", errx.TypeInternal)
		}
	} else {
		// Undoing a redemption gives the neurons back to the student.
		if !vault.Unlimited && vaultBefore < orig.Amount {
			return nil, ledger.ErrInsufficientVault()
		}
		studentAfter = studentBefore + orig.Amount
		vaultAfter = vaultBefore - orig.Amount
		undo := `UPDATE enrollments
		         SET balance = balance + $2, total_returned = total_returned - $2
		         WHERE id = $1`
		if _, err := tx.ExecContext(ctx, undo, *orig.EnrollmentID, orig.Amount); err != nil {
			return nil, errx.Wrap(err, "undo student debit", errx.TypeInternal)
		}
		undoVault := `UPDATE classrooms
		              SET vault_balance = vault_balance - $2, total_redeemed = total_redeemed - $2
		              WHERE id = $1`
		if _, err := tx.ExecContext(ctx, undoVault, orig.ClassroomID, orig.Amount); err != nil {
			return nil, errx.Wrap(err, "undo vault credit", errx.TypeInternal)
		}
		if orig.BenefitID != nil {
			if _, err := tx.ExecContext(ctx, `UPDATE benefits SET uses_count = GREATEST(uses_count - 1, 0) WHERE id = $1`, *orig.BenefitID); err != nil {
				return nil, errx.Wrap(err, "undo benefit use", errx.TypeInternal)
			}
		}
	}

	reversal := ledger.Transaction{
		ID:                    kernel.NewLedgerID(uuid.NewString()),
		Code:                  nextCode(reversalType),
		ClassroomID:           orig.ClassroomID,
		Type:                  reversalType,
		EnrollmentID:          orig.EnrollmentID,
		TeamID:                orig.TeamID,
		Amount:                orig.Amount,
		ReasonID:              orig.ReasonID,
		ReasonText:            orig.ReasonText,
		BenefitID:             orig.BenefitID,
		BenefitText:           orig.BenefitText,
		StudentBalanceBefore:  &studentBefore,
		StudentBalanceAfter:   &studentAfter,
		VaultBalanceBefore:    &vaultBefore,
		VaultBalanceAfter:     &vaultAfter,
		Channel:               ledger.ChannelSystem,
		Status:                ledger.StatusApplied,
		ReversesTransactionID: &orig.ID,
		PerformedBy:           performedBy,
		Notes:                 note,
		CreatedAt:             time.Now(),
	}
	if err := insertTx(ctx, tx, &reversal); err != nil {
		return nil, err
	}

	mark := `UPDATE transactions SET status = $2, reversed_by_transaction_id = $3 WHERE id = $1`
	if _, err := tx.ExecContext(ctx, mark, orig.ID, ledger.StatusReversed, reversal.ID); err != nil {
		return nil, errx.Wrap(err, "mark transaction reversed", errx.TypeInternal)
	}

	if err := tx.Commit(); err != nil {
		return nil, errx.Wrap(err, "commit reverse tx", errx.TypeInternal)
	}
	return &reversal, nil
}

func lockEnrollmentForReversal(ctx context.Context, tx *sqlx.Tx, id kernel.EnrollmentID) (*lockedEnrollment, error) {
	var e lockedEnrollment
	// RN-16: a withdrawn student keeps their balance, so reversals still apply
	// to them; only the classroom status gates the operation.
	if err := tx.GetContext(ctx, &e, `SELECT id, balance, status FROM enrollments WHERE id = $1 FOR UPDATE`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ledger.ErrNotFound()
		}
		return nil, errx.Wrap(err, "lock enrollment", errx.TypeInternal)
	}
	return &e, nil
}

func (r *PostgresLedgerRepository) GetByID(ctx context.Context, id kernel.LedgerID) (*ledger.Transaction, error) {
	var t ledger.Transaction
	query := `SELECT ` + dbx.Prefixed("t", txColumns) + `,
	                 u.name AS student_name, p.name AS performer_name, tm.name AS team_name
	          FROM transactions t
	          LEFT JOIN enrollments e ON e.id = t.enrollment_id
	          LEFT JOIN users u ON u.id = e.user_id
	          LEFT JOIN users p ON p.id = t.performed_by
	          LEFT JOIN teams tm ON tm.id = t.team_id
	          WHERE t.id = $1`
	if err := r.db.GetContext(ctx, &t, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ledger.ErrNotFound()
		}
		return nil, errx.Wrap(err, "get transaction", errx.TypeInternal)
	}
	return &t, nil
}

func (r *PostgresLedgerRepository) FindByIdempotencyKey(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	key string,
) ([]ledger.Transaction, error) {
	items := []ledger.Transaction{}
	query := `SELECT ` + txColumns + ` FROM transactions
	          WHERE classroom_id = $1 AND idempotency_key = $2`
	if err := r.db.SelectContext(ctx, &items, query, classroomID, key); err != nil {
		return nil, errx.Wrap(err, "find by idempotency key", errx.TypeInternal)
	}
	if len(items) == 0 {
		return nil, nil
	}
	// A batch stores the key on its first entry only; return the whole batch.
	if items[0].BatchID != nil {
		batchItems := []ledger.Transaction{}
		q := `SELECT ` + txColumns + ` FROM transactions WHERE batch_id = $1 ORDER BY created_at`
		if err := r.db.SelectContext(ctx, &batchItems, q, *items[0].BatchID); err != nil {
			return nil, errx.Wrap(err, "load batch transactions", errx.TypeInternal)
		}
		return batchItems, nil
	}
	return items, nil
}

func (r *PostgresLedgerRepository) History(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	filter ledger.HistoryFilter,
	opts kernel.PaginationOptions,
) (kernel.Paginated[ledger.Transaction], error) {
	var empty kernel.Paginated[ledger.Transaction]

	where := []string{"t.classroom_id = :classroom_id"}
	args := map[string]any{"classroom_id": classroomID}

	if filter.EnrollmentID != nil {
		where = append(where, "t.enrollment_id = :enrollment_id")
		args["enrollment_id"] = *filter.EnrollmentID
	}
	if filter.TeamID != nil {
		where = append(where, "t.team_id = :team_id")
		args["team_id"] = *filter.TeamID
	}
	if filter.Type != nil {
		where = append(where, "t.type = :type")
		args["type"] = *filter.Type
	}
	if filter.ReasonID != nil {
		where = append(where, "t.reason_id = :reason_id")
		args["reason_id"] = *filter.ReasonID
	}
	if filter.BenefitID != nil {
		where = append(where, "t.benefit_id = :benefit_id")
		args["benefit_id"] = *filter.BenefitID
	}
	if filter.From != nil {
		where = append(where, "t.created_at >= :from")
		args["from"] = *filter.From
	}
	if filter.To != nil {
		where = append(where, "t.created_at <= :to")
		args["to"] = *filter.To
	}
	clause := " WHERE " + strings.Join(where, " AND ")

	var total int
	countQuery, countArgs, err := sqlx.Named(`SELECT COUNT(*) FROM transactions t`+clause, args)
	if err != nil {
		return empty, errx.Wrap(err, "build count query", errx.TypeInternal)
	}
	if err := r.db.GetContext(ctx, &total, r.db.Rebind(countQuery), countArgs...); err != nil {
		return empty, errx.Wrap(err, "count transactions", errx.TypeInternal)
	}

	args["limit"] = opts.PageSize
	args["offset"] = (opts.Page - 1) * opts.PageSize

	items := []ledger.Transaction{}
	listSQL := `SELECT ` + dbx.Prefixed("t", txColumns) + `,
	                   u.name AS student_name, p.name AS performer_name, tm.name AS team_name
	            FROM transactions t
	            LEFT JOIN enrollments e ON e.id = t.enrollment_id
	            LEFT JOIN users u ON u.id = e.user_id
	            LEFT JOIN users p ON p.id = t.performed_by
	            LEFT JOIN teams tm ON tm.id = t.team_id` + clause + `
	            ORDER BY t.created_at DESC
	            LIMIT :limit OFFSET :offset`
	listQuery, listArgs, err := sqlx.Named(listSQL, args)
	if err != nil {
		return empty, errx.Wrap(err, "build list query", errx.TypeInternal)
	}
	if err := r.db.SelectContext(ctx, &items, r.db.Rebind(listQuery), listArgs...); err != nil {
		return empty, errx.Wrap(err, "list transactions", errx.TypeInternal)
	}

	return kernel.NewPaginated(items, opts.Page, opts.PageSize, total), nil
}

func (r *PostgresLedgerRepository) StudentHistory(
	ctx context.Context,
	enrollmentID kernel.EnrollmentID,
	opts kernel.PaginationOptions,
) (kernel.Paginated[ledger.Transaction], error) {
	var empty kernel.Paginated[ledger.Transaction]

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM transactions WHERE enrollment_id = $1`, enrollmentID); err != nil {
		return empty, errx.Wrap(err, "count student transactions", errx.TypeInternal)
	}

	items := []ledger.Transaction{}
	query := `SELECT ` + dbx.Prefixed("t", txColumns) + `, p.name AS performer_name
	          FROM transactions t
	          LEFT JOIN users p ON p.id = t.performed_by
	          WHERE t.enrollment_id = $1
	          ORDER BY t.created_at DESC
	          LIMIT $2 OFFSET $3`
	if err := r.db.SelectContext(ctx, &items, query, enrollmentID, opts.PageSize, (opts.Page-1)*opts.PageSize); err != nil {
		return empty, errx.Wrap(err, "list student transactions", errx.TypeInternal)
	}

	return kernel.NewPaginated(items, opts.Page, opts.PageSize, total), nil
}

func (r *PostgresLedgerRepository) Stats(
	ctx context.Context,
	classroomID kernel.ClassroomID,
) (*ledger.ClassroomStats, error) {
	var s ledger.ClassroomStats
	query := `SELECT c.vault_balance, c.unlimited_issuance, c.total_granted, c.total_redeemed,
	                 COALESCE((SELECT SUM(balance) FROM enrollments WHERE classroom_id = c.id), 0) AS in_circulation,
	                 (SELECT COUNT(*) FROM enrollments WHERE classroom_id = c.id AND status = 'ACTIVE') AS active_students,
	                 (SELECT COUNT(*) FROM transactions WHERE classroom_id = c.id) AS transactions
	          FROM classrooms c WHERE c.id = $1`
	if err := r.db.GetContext(ctx, &s, query, classroomID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ledger.ErrNotFound()
		}
		return nil, errx.Wrap(err, "classroom stats", errx.TypeInternal)
	}
	return &s, nil
}

func (r *PostgresLedgerRepository) Ranking(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	limit int,
) ([]ledger.RankingEntry, error) {
	rows := []struct {
		EnrollmentID  kernel.EnrollmentID `db:"enrollment_id"`
		StudentName   string              `db:"student_name"`
		TotalReceived int64               `db:"total_received"`
		MedalCount    int                 `db:"medal_count"`
	}{}
	// Ranking is by neurons earned, never by current balance: spending should
	// not push a student down the board.
	query := `SELECT e.id AS enrollment_id, u.name AS student_name, e.total_received,
	                 (SELECT COUNT(*) FROM medal_awards ma WHERE ma.enrollment_id = e.id AND ma.revoked_at IS NULL) AS medal_count
	          FROM enrollments e
	          JOIN users u ON u.id = e.user_id
	          WHERE e.classroom_id = $1 AND e.status = 'ACTIVE'
	          ORDER BY e.total_received DESC, u.name
	          LIMIT $2`
	if err := r.db.SelectContext(ctx, &rows, query, classroomID, limit); err != nil {
		return nil, errx.Wrap(err, "classroom ranking", errx.TypeInternal)
	}

	entries := make([]ledger.RankingEntry, 0, len(rows))
	for i, row := range rows {
		entries = append(entries, ledger.RankingEntry{
			Position:      i + 1,
			EnrollmentID:  row.EnrollmentID,
			StudentName:   row.StudentName,
			TotalReceived: row.TotalReceived,
			MedalCount:    row.MedalCount,
		})
	}
	return entries, nil
}

func (r *PostgresLedgerRepository) ReasonUsage(
	ctx context.Context,
	classroomID kernel.ClassroomID,
) ([]ledger.ReasonUsage, error) {
	items := []ledger.ReasonUsage{}
	query := `SELECT t.reason_id,
	                 COALESCE(rs.name, t.reason_text, 'Sin motivo') AS reason_name,
	                 COUNT(*) AS uses,
	                 SUM(t.amount) AS total_amount
	          FROM transactions t
	          LEFT JOIN reasons rs ON rs.id = t.reason_id
	          WHERE t.classroom_id = $1 AND t.type = 'GRANT' AND t.status = 'APPLIED'
	          GROUP BY t.reason_id, rs.name, t.reason_text
	          ORDER BY uses DESC`
	if err := r.db.SelectContext(ctx, &items, query, classroomID); err != nil {
		return nil, errx.Wrap(err, "reason usage report", errx.TypeInternal)
	}
	return items, nil
}

func (r *PostgresLedgerRepository) GetBatch(
	ctx context.Context,
	batchID string,
) (*ledger.Batch, []ledger.Transaction, error) {
	var b ledger.Batch
	batchQuery := `SELECT id, classroom_id, type, team_id, amount_per_student, recipient_count,
	                      total_amount, reason_id, reason_text, performed_by, created_at
	               FROM transaction_batches WHERE id = $1`
	if err := r.db.GetContext(ctx, &b, batchQuery, batchID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ledger.ErrNotFound()
		}
		return nil, nil, errx.Wrap(err, "get batch", errx.TypeInternal)
	}

	items := []ledger.Transaction{}
	query := `SELECT ` + dbx.Prefixed("t", txColumns) + `, u.name AS student_name
	          FROM transactions t
	          LEFT JOIN enrollments e ON e.id = t.enrollment_id
	          LEFT JOIN users u ON u.id = e.user_id
	          WHERE t.batch_id = $1
	          ORDER BY t.created_at`
	if err := r.db.SelectContext(ctx, &items, query, batchID); err != nil {
		return nil, nil, errx.Wrap(err, "list batch transactions", errx.TypeInternal)
	}

	return &b, items, nil
}
