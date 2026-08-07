package qrsrv

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/ledger"
	"github.com/Abraxas-356/neurons/internal/gamification/ledger/ledgersrv"
	"github.com/Abraxas-356/neurons/internal/gamification/qr"
	"github.com/Abraxas-356/neurons/internal/kernel"
)

// QRService issues and consumes the short-lived codes used in class (flow A).
type QRService struct {
	store       qr.Store
	classrooms  *classroomsrv.ClassroomService
	enrollments *enrollmentsrv.EnrollmentService
	ledger      *ledgersrv.LedgerService
}

func NewQRService(
	store qr.Store,
	classrooms *classroomsrv.ClassroomService,
	enrollments *enrollmentsrv.EnrollmentService,
	ledgerSvc *ledgersrv.LedgerService,
) *QRService {
	return &QRService{store: store, classrooms: classrooms, enrollments: enrollments, ledger: ledgerSvc}
}

// newCode returns an unguessable opaque code. 128 bits of entropy makes brute
// forcing a live token pointless (§11.2).
func newCode() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", qr.ErrInternal(err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// IssueStudent returns the rotating code a student displays in class (HU-053).
func (s *QRService) IssueStudent(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	userID kernel.UserID,
) (*qr.Issued, error) {
	e, _, err := s.enrollments.MyEnrollment(ctx, classroomID, userID)
	if err != nil {
		return nil, err
	}
	if !e.CanTransact() {
		return nil, qr.ErrInvalidInput("your enrollment is not active in this classroom")
	}

	code, err := newCode()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	enrollmentID := e.ID
	owner := userID
	token := &qr.Token{
		Code:         code,
		Kind:         qr.KindStudent,
		ClassroomID:  classroomID,
		EnrollmentID: &enrollmentID,
		UserID:       &owner,
		IssuedAt:     now,
		ExpiresAt:    now.Add(qr.DefaultStudentTTL),
	}
	if err := s.store.Save(ctx, token, qr.DefaultStudentTTL); err != nil {
		return nil, err
	}

	issued := token.ToIssued(now)
	return &issued, nil
}

// Scan resolves a student code for the teacher's scanner (flow A step 2). It
// does not move neurons: the teacher still picks an amount and confirms.
func (s *QRService) Scan(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	req qr.ScanRequest,
) (*qr.ScannedStudent, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}

	token, err := s.resolve(ctx, strings.TrimSpace(req.Code), qr.KindStudent, classroomID)
	if err != nil {
		return nil, err
	}

	e, err := s.enrollments.GetForTeacher(ctx, *token.EnrollmentID, teacherID)
	if err != nil {
		return nil, err
	}

	return &qr.ScannedStudent{
		EnrollmentID: e.ID,
		ClassroomID:  e.ClassroomID,
		StudentName:  e.Name,
		StudentEmail: e.Email,
		Balance:      e.Balance,
		TeamID:       e.TeamID,
		TeamName:     e.TeamName,
		GrantKey:     "qrscan:" + token.Code,
	}, nil
}

// IssueGrant prepares a grant that students claim by scanning (HU-050). The
// neurons are not reserved: each claim is charged to the vault when it happens.
func (s *QRService) IssueGrant(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	req qr.IssueGrantRequest,
) (*qr.Issued, error) {
	c, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID)
	if err != nil {
		return nil, err
	}
	if !c.AcceptsTransactions() {
		return nil, ledger.ErrClassroomClosed()
	}
	if req.Amount <= 0 {
		return nil, qr.ErrInvalidInput("amount must be positive")
	}
	// RN-10: the reason is fixed when the code is issued, so every claim is
	// justified without asking the teacher again.
	if req.ReasonID == nil && strings.TrimSpace(derefStr(req.ReasonText)) == "" {
		return nil, ledger.ErrReasonRequired()
	}

	ttl := time.Duration(req.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = qr.DefaultGrantTTL
	}
	if ttl > qr.MaxGrantTTL {
		ttl = qr.MaxGrantTTL
	}

	code, err := newCode()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	issuer := teacherID
	token := &qr.Token{
		Code:        code,
		Kind:        qr.KindGrant,
		ClassroomID: classroomID,
		Amount:      req.Amount,
		ReasonID:    req.ReasonID,
		ReasonText:  req.ReasonText,
		IssuedBy:    &issuer,
		MaxClaims:   req.MaxClaims,
		IssuedAt:    now,
		ExpiresAt:   now.Add(ttl),
	}
	if err := s.store.Save(ctx, token, ttl); err != nil {
		return nil, err
	}

	issued := token.ToIssued(now)
	return &issued, nil
}

// Claim is the student side of a teacher-displayed grant code. The claim is
// registered atomically before any neurons move, so one student can never be
// paid twice for the same code (§11.3).
func (s *QRService) Claim(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	userID kernel.UserID,
	req qr.ClaimRequest,
) (*ledger.GrantResult, error) {
	token, err := s.resolve(ctx, strings.TrimSpace(req.Code), qr.KindGrant, classroomID)
	if err != nil {
		return nil, err
	}

	e, _, err := s.enrollments.MyEnrollment(ctx, classroomID, userID)
	if err != nil {
		return nil, err
	}
	if !e.CanTransact() {
		return nil, ledger.ErrStudentInactive()
	}

	if token.MaxClaims > 0 {
		claimed, err := s.store.ClaimCount(ctx, token.Code)
		if err != nil {
			return nil, err
		}
		if claimed >= token.MaxClaims {
			return nil, qr.ErrExhausted()
		}
	}

	ttl := token.TTL(time.Now())
	first, err := s.store.ClaimOnce(ctx, token.Code, e.ID.String(), ttl)
	if err != nil {
		return nil, err
	}
	if !first {
		return nil, qr.ErrAlreadyClaimed()
	}

	// The grant is performed on behalf of the issuing teacher: students never
	// have the authority to move neurons themselves (RN-02).
	idempotencyKey := "qr:" + token.Code + ":" + e.ID.String()
	return s.ledger.Grant(ctx, classroomID, *token.IssuedBy, ledger.GrantRequest{
		EnrollmentIDs:  []kernel.EnrollmentID{e.ID},
		Amount:         token.Amount,
		ReasonID:       token.ReasonID,
		ReasonText:     token.ReasonText,
		Channel:        ledger.ChannelQR,
		IdempotencyKey: &idempotencyKey,
		// The teacher already confirmed the amount when issuing the code.
		Confirmed: true,
	}, nil)
}

// resolve loads a token and enforces RN-12 (classroom binding) and RN-13
// (expiry) before anything else happens.
func (s *QRService) resolve(
	ctx context.Context,
	code string,
	kind qr.Kind,
	classroomID kernel.ClassroomID,
) (*qr.Token, error) {
	if code == "" {
		return nil, qr.ErrInvalid()
	}

	token, err := s.store.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	if token.Kind != kind {
		return nil, qr.ErrWrongKind()
	}
	if token.ClassroomID != classroomID {
		return nil, qr.ErrWrongClassroom()
	}
	if token.IsExpired(time.Now()) {
		return nil, qr.ErrInvalid()
	}
	return token, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
