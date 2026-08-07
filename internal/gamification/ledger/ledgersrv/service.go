package ledgersrv

import (
	"context"
	"strings"
	"time"

	"github.com/Abraxas-356/neurons/internal/gamification/benefit/benefitsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/classroom"
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/ledger"
	"github.com/Abraxas-356/neurons/internal/gamification/reason"
	"github.com/Abraxas-356/neurons/internal/gamification/reason/reasonsrv"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/google/uuid"
)

// LedgerService orchestrates every neuron movement: it authorizes the actor,
// validates the business rules and then hands a fully resolved operation to the
// repository, which performs it atomically.
type LedgerService struct {
	repo        ledger.Repository
	classrooms  *classroomsrv.ClassroomService
	enrollments *enrollmentsrv.EnrollmentService
	reasons     *reasonsrv.ReasonService
	benefits    *benefitsrv.BenefitService
}

func NewLedgerService(
	repo ledger.Repository,
	classrooms *classroomsrv.ClassroomService,
	enrollments *enrollmentsrv.EnrollmentService,
	reasons *reasonsrv.ReasonService,
	benefits *benefitsrv.BenefitService,
) *LedgerService {
	return &LedgerService{
		repo:        repo,
		classrooms:  classrooms,
		enrollments: enrollments,
		reasons:     reasons,
		benefits:    benefits,
	}
}

// Grant hands neurons to one or more students (flows A and D).
func (s *LedgerService) Grant(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	req ledger.GrantRequest,
	deviceInfo *string,
) (*ledger.GrantResult, error) {
	c, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID)
	if err != nil {
		return nil, err
	}
	if err := s.checkAmount(c, req.Amount, req.Confirmed); err != nil {
		return nil, err
	}
	if len(req.EnrollmentIDs) == 0 {
		return nil, ledger.ErrNoRecipients()
	}

	if done, err := s.replay(ctx, classroomID, req.IdempotencyKey); done != nil || err != nil {
		return done, err
	}

	reasonID, reasonText, err := s.resolveReason(ctx, c, req.ReasonID, req.ReasonText, reason.ScopeIndividual)
	if err != nil {
		return nil, err
	}

	recipients, err := s.enrollments.ListActiveByIDs(ctx, classroomID, req.EnrollmentIDs)
	if err != nil {
		return nil, err
	}
	if len(recipients) != len(req.EnrollmentIDs) {
		return nil, ledger.ErrStudentInactive()
	}

	channel := req.Channel
	if channel == "" {
		channel = ledger.ChannelManual
	}

	op := ledger.GrantOp{
		ClassroomID:    classroomID,
		Recipients:     idsOf(recipients),
		AmountEach:     req.Amount,
		ReasonID:       reasonID,
		ReasonText:     reasonText,
		Notes:          req.Notes,
		Channel:        channel,
		IdempotencyKey: req.IdempotencyKey,
		PerformedBy:    teacherID,
		DeviceInfo:     deviceInfo,
	}
	if len(recipients) > 1 {
		op.Batch = s.newBatch(classroomID, ledger.BatchMultiGrant, nil, req.Amount, len(recipients), reasonID, reasonText, teacherID)
	}

	txs, err := s.repo.Grant(ctx, op)
	if err != nil {
		return nil, err
	}
	return s.grantResult(ctx, classroomID, txs, req.Amount, len(recipients), op.Batch)
}

// GrantToTeam gives the amount to EACH active member of the team (RN-07).
func (s *LedgerService) GrantToTeam(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	req ledger.TeamGrantRequest,
	deviceInfo *string,
) (*ledger.GrantResult, error) {
	c, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID)
	if err != nil {
		return nil, err
	}
	if err := s.checkAmount(c, req.Amount, req.Confirmed); err != nil {
		return nil, err
	}

	if done, err := s.replay(ctx, classroomID, req.IdempotencyKey); done != nil || err != nil {
		return done, err
	}

	reasonID, reasonText, err := s.resolveReason(ctx, c, req.ReasonID, req.ReasonText, reason.ScopeTeam)
	if err != nil {
		return nil, err
	}

	members, err := s.enrollments.ListActiveByTeam(ctx, req.TeamID)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, ledger.ErrNoRecipients()
	}
	// RN-01: the team must belong to this classroom.
	for i := range members {
		if members[i].ClassroomID != classroomID {
			return nil, ledger.ErrForbidden()
		}
	}

	teamID := req.TeamID
	batch := s.newBatch(classroomID, ledger.BatchTeamGrant, &teamID, req.Amount, len(members), reasonID, reasonText, teacherID)

	txs, err := s.repo.Grant(ctx, ledger.GrantOp{
		ClassroomID:    classroomID,
		Recipients:     idsOf(members),
		AmountEach:     req.Amount,
		ReasonID:       reasonID,
		ReasonText:     reasonText,
		Notes:          req.Notes,
		Channel:        ledger.ChannelManual,
		Batch:          batch,
		IdempotencyKey: req.IdempotencyKey,
		PerformedBy:    teacherID,
		DeviceInfo:     deviceInfo,
	})
	if err != nil {
		return nil, err
	}
	return s.grantResult(ctx, classroomID, txs, req.Amount, len(members), batch)
}

// Redeem receives neurons back from a student (flow C, RN-03). Only a teacher
// can complete a return: neurons always come back to the teacher, never move
// between students (RN-02).
func (s *LedgerService) Redeem(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	req ledger.RedeemRequest,
	deviceInfo *string,
) (*ledger.Transaction, error) {
	c, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID)
	if err != nil {
		return nil, err
	}
	if !c.AcceptsTransactions() {
		return nil, ledger.ErrClassroomClosed()
	}

	if req.IdempotencyKey != nil {
		existing, err := s.repo.FindByIdempotencyKey(ctx, classroomID, *req.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if len(existing) > 0 {
			return &existing[0], nil
		}
	}

	e, err := s.enrollments.GetForTeacher(ctx, req.EnrollmentID, teacherID)
	if err != nil {
		return nil, err
	}
	if !e.CanTransact() {
		return nil, ledger.ErrStudentInactive()
	}

	amount, benefitID, benefitText, err := s.resolveRedemption(ctx, c, e, req)
	if err != nil {
		return nil, err
	}
	// RN-04: never let a student go negative. The repository re-checks this
	// under a row lock; this is the friendly early exit.
	if !e.CanAfford(amount) {
		return nil, ledger.ErrInsufficientBalance()
	}

	channel := req.Channel
	if channel == "" {
		channel = ledger.ChannelManual
	}

	return s.repo.Redeem(ctx, ledger.RedeemOp{
		ClassroomID:    classroomID,
		EnrollmentID:   req.EnrollmentID,
		Amount:         amount,
		BenefitID:      benefitID,
		BenefitText:    benefitText,
		Notes:          req.Notes,
		Channel:        channel,
		IdempotencyKey: req.IdempotencyKey,
		PerformedBy:    teacherID,
		DeviceInfo:     deviceInfo,
	})
}

// resolveRedemption applies RN-11: a return always carries a concept, either a
// catalog benefit or free text when the classroom allows it.
func (s *LedgerService) resolveRedemption(
	ctx context.Context,
	c *classroom.Classroom,
	e *enrollment.Enrollment,
	req ledger.RedeemRequest,
) (int64, *kernel.BenefitID, *string, error) {
	if req.BenefitID != nil {
		b, cost, err := s.benefits.ResolveForRedemption(ctx, *req.BenefitID, e, req.Amount)
		if err != nil {
			return 0, nil, nil, err
		}
		name := b.Name
		return int64(cost), &b.ID, &name, nil
	}

	if !c.AllowFreeRedemption {
		return 0, nil, nil, ledger.ErrConceptRequired()
	}
	text := strings.TrimSpace(deref(req.BenefitText))
	if text == "" {
		return 0, nil, nil, ledger.ErrConceptRequired()
	}
	if req.Amount == nil || *req.Amount <= 0 {
		return 0, nil, nil, ledger.ErrInvalidAmount("amount is required")
	}
	return int64(*req.Amount), nil, &text, nil
}

// Topup refills the vault (decision 15.8). Only the classroom owner may do it.
func (s *LedgerService) Topup(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	req ledger.TopupRequest,
) (*ledger.Transaction, error) {
	if _, err := s.classrooms.RequireOwner(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, ledger.ErrInvalidAmount("amount must be positive")
	}
	return s.repo.Topup(ctx, classroomID, req.Amount, req.Notes, teacherID)
}

// Reverse undoes an applied movement (HU-092, decision 15.4).
func (s *LedgerService) Reverse(
	ctx context.Context,
	id kernel.LedgerID,
	teacherID kernel.UserID,
	req ledger.ReverseRequest,
) (*ledger.Transaction, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.classrooms.RequireTeacher(ctx, t.ClassroomID, teacherID); err != nil {
		return nil, err
	}
	if !t.CanBeReversed() {
		if t.Status == ledger.StatusReversed {
			return nil, ledger.ErrAlreadyReversed()
		}
		return nil, ledger.ErrNotReversible()
	}
	return s.repo.Reverse(ctx, id, teacherID, req.Reason)
}

// Get returns one ledger entry for a teacher of its classroom.
func (s *LedgerService) Get(
	ctx context.Context,
	id kernel.LedgerID,
	teacherID kernel.UserID,
) (*ledger.Transaction, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.classrooms.RequireTeacher(ctx, t.ClassroomID, teacherID); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *LedgerService) History(ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	filter ledger.HistoryFilter,
	opts kernel.PaginationOptions,
) (kernel.Paginated[ledger.Transaction], error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return kernel.Paginated[ledger.Transaction]{}, err
	}
	return s.repo.History(ctx, classroomID, filter, opts)
}

// MyHistory is the student's own movement list (HU-082). Students only ever
// see their own data (§10.4).
func (s *LedgerService) MyHistory(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	userID kernel.UserID,
	opts kernel.PaginationOptions,
) (kernel.Paginated[ledger.Transaction], error) {
	e, _, err := s.enrollments.MyEnrollment(ctx, classroomID, userID)
	if err != nil {
		return kernel.Paginated[ledger.Transaction]{}, err
	}
	return s.repo.StudentHistory(ctx, e.ID, opts)
}

func (s *LedgerService) StudentHistory(
	ctx context.Context,
	enrollmentID kernel.EnrollmentID,
	teacherID kernel.UserID,
	opts kernel.PaginationOptions,
) (kernel.Paginated[ledger.Transaction], error) {
	if _, err := s.enrollments.GetForTeacher(ctx, enrollmentID, teacherID); err != nil {
		return kernel.Paginated[ledger.Transaction]{}, err
	}
	return s.repo.StudentHistory(ctx, enrollmentID, opts)
}

func (s *LedgerService) Stats(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
) (*ledger.ClassroomStats, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}
	return s.repo.Stats(ctx, classroomID)
}

func (s *LedgerService) ReasonUsage(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
) ([]ledger.ReasonUsage, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}
	return s.repo.ReasonUsage(ctx, classroomID)
}

// Ranking is visible to students only when the teacher published it
// (decision 15.7).
func (s *LedgerService) Ranking(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	userID kernel.UserID,
	limit int,
) ([]ledger.RankingEntry, error) {
	c, err := s.classrooms.GetByID(ctx, classroomID)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, userID); err != nil {
		if !c.RankingPublic {
			return nil, ledger.ErrForbidden()
		}
		// A student may only see the ranking of a classroom they belong to.
		if _, _, err := s.enrollments.MyEnrollment(ctx, classroomID, userID); err != nil {
			return nil, err
		}
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.Ranking(ctx, classroomID, limit)
}

func (s *LedgerService) GetBatch(
	ctx context.Context,
	batchID string,
	teacherID kernel.UserID,
) (*ledger.Batch, []ledger.Transaction, error) {
	b, items, err := s.repo.GetBatch(ctx, batchID)
	if err != nil {
		return nil, nil, err
	}
	if _, _, err := s.classrooms.RequireTeacher(ctx, b.ClassroomID, teacherID); err != nil {
		return nil, nil, err
	}
	return b, items, nil
}

// --- helpers ---

// checkAmount enforces RN-08, RN-17 and the §11.9 reconfirmation guard.
func (s *LedgerService) checkAmount(c *classroom.Classroom, amount int64, confirmed bool) error {
	if !c.AcceptsTransactions() {
		return ledger.ErrClassroomClosed()
	}
	if amount <= 0 {
		return ledger.ErrInvalidAmount("amount must be a positive whole number")
	}
	if c.NeedsReconfirmation(amount) && !confirmed {
		return ledger.ErrConfirmationRequired()
	}
	return nil
}

// resolveReason enforces RN-10: a grant always carries a reason, either from
// the catalog or as free text.
func (s *LedgerService) resolveReason(
	ctx context.Context,
	c *classroom.Classroom,
	reasonID *kernel.ReasonID,
	reasonText *string,
	scope reason.Scope,
) (*kernel.ReasonID, *string, error) {
	if reasonID != nil {
		r, err := s.reasons.GetForGrant(ctx, *reasonID, c.ID, scope)
		if err != nil {
			return nil, nil, err
		}
		name := r.Name
		return &r.ID, &name, nil
	}

	text := strings.TrimSpace(deref(reasonText))
	if text == "" {
		return nil, nil, ledger.ErrReasonRequired()
	}
	return nil, &text, nil
}

// replay returns the original result when an idempotency key was already used,
// so a double submit or a re-scanned QR never pays twice (§11.3).
func (s *LedgerService) replay(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	key *string,
) (*ledger.GrantResult, error) {
	if key == nil || *key == "" {
		return nil, nil
	}
	existing, err := s.repo.FindByIdempotencyKey(ctx, classroomID, *key)
	if err != nil {
		return nil, err
	}
	if len(existing) == 0 {
		return nil, nil
	}

	stats, err := s.repo.Stats(ctx, classroomID)
	if err != nil {
		return nil, err
	}
	return &ledger.GrantResult{
		BatchID:      existing[0].BatchID,
		Transactions: existing,
		Recipients:   len(existing),
		AmountEach:   existing[0].Amount,
		TotalAmount:  existing[0].Amount * int64(len(existing)),
		VaultBalance: stats.VaultBalance,
	}, nil
}

func (s *LedgerService) grantResult(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	txs []ledger.Transaction,
	amountEach int64,
	recipients int,
	batch *ledger.Batch,
) (*ledger.GrantResult, error) {
	stats, err := s.repo.Stats(ctx, classroomID)
	if err != nil {
		return nil, err
	}
	var batchID *string
	if batch != nil {
		batchID = &batch.ID
	}
	return &ledger.GrantResult{
		BatchID:      batchID,
		Transactions: txs,
		Recipients:   recipients,
		AmountEach:   amountEach,
		TotalAmount:  amountEach * int64(recipients),
		VaultBalance: stats.VaultBalance,
	}, nil
}

func (s *LedgerService) newBatch(
	classroomID kernel.ClassroomID,
	batchType ledger.BatchType,
	teamID *kernel.TeamID,
	amount int64,
	recipients int,
	reasonID *kernel.ReasonID,
	reasonText *string,
	teacherID kernel.UserID,
) *ledger.Batch {
	return &ledger.Batch{
		ID:               uuid.NewString(),
		ClassroomID:      classroomID,
		Type:             batchType,
		TeamID:           teamID,
		AmountPerStudent: int(amount),
		RecipientCount:   recipients,
		TotalAmount:      amount * int64(recipients),
		ReasonID:         reasonID,
		ReasonText:       reasonText,
		PerformedBy:      teacherID,
		CreatedAt:        time.Now(),
	}
}

func idsOf(items []enrollment.Enrollment) []kernel.EnrollmentID {
	ids := make([]kernel.EnrollmentID, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	return ids
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
