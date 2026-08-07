package ledgerinfra_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Abraxas-356/neurons/internal/gamification/ledger"
	"github.com/Abraxas-356/neurons/internal/gamification/ledger/ledgerinfra"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// dsn points at the local docker-compose database. The test is skipped when it
// is not reachable so `go test ./...` stays green without docker.
func dsn() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://neurons:supersecret@localhost:5490/neuronsdb?sslmode=disable"
}

func openDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Connect("postgres", dsn())
	if err != nil {
		t.Skipf("database not reachable: %v", err)
	}
	return db
}

// fixture creates a throwaway tenant, teacher, classroom and two students.
type fixture struct {
	classroomID kernel.ClassroomID
	teacherID   kernel.UserID
	students    []kernel.EnrollmentID
}

func newFixture(t *testing.T, db *sqlx.DB, vault int64, unlimited bool) fixture {
	t.Helper()
	ctx := context.Background()
	suffix := uuid.NewString()

	tenantID := "t-" + suffix
	if _, err := db.ExecContext(ctx,
		`INSERT INTO tenants (id, company_name, status) VALUES ($1, $2, 'ACTIVE')`,
		tenantID, "Test "+suffix); err != nil {
		t.Fatalf("insert tenant: %v", err)
	}

	teacherID := "u-teacher-" + suffix
	insertUser(t, db, teacherID, tenantID, "teacher-"+suffix+"@test.local")

	classroomID := "c-" + suffix
	if _, err := db.ExecContext(ctx,
		`INSERT INTO classrooms (id, tenant_id, name, invite_code, status, unlimited_issuance, vault_balance, created_by)
		 VALUES ($1, $2, 'Test Course', $3, 'ACTIVE', $4, $5, $6)`,
		classroomID, tenantID, "IC"+suffix[:6], unlimited, vault, teacherID); err != nil {
		t.Fatalf("insert classroom: %v", err)
	}

	f := fixture{
		classroomID: kernel.NewClassroomID(classroomID),
		teacherID:   kernel.NewUserID(teacherID),
	}

	for i := 0; i < 2; i++ {
		userID := "u-student-" + suffix + "-" + uuid.NewString()[:4]
		insertUser(t, db, userID, tenantID, userID+"@test.local")

		enrollmentID := "e-" + uuid.NewString()
		if _, err := db.ExecContext(ctx,
			`INSERT INTO enrollments (id, classroom_id, user_id, status) VALUES ($1, $2, $3, 'ACTIVE')`,
			enrollmentID, classroomID, userID); err != nil {
			t.Fatalf("insert enrollment: %v", err)
		}
		f.students = append(f.students, kernel.NewEnrollmentID(enrollmentID))
	}

	t.Cleanup(func() {
		db.Exec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	})
	return f
}

func insertUser(t *testing.T, db *sqlx.DB, id, tenantID, email string) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO users (id, tenant_id, email, name, status, oauth_provider, oauth_provider_id)
		 VALUES ($1, $2, $3, $4, 'ACTIVE', 'test', $1)`,
		id, tenantID, email, "Test User"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

func balanceOf(t *testing.T, db *sqlx.DB, id kernel.EnrollmentID) int64 {
	t.Helper()
	var b int64
	if err := db.Get(&b, `SELECT balance FROM enrollments WHERE id = $1`, id); err != nil {
		t.Fatalf("read balance: %v", err)
	}
	return b
}

func vaultOf(t *testing.T, db *sqlx.DB, id kernel.ClassroomID) int64 {
	t.Helper()
	var b int64
	if err := db.Get(&b, `SELECT vault_balance FROM classrooms WHERE id = $1`, id); err != nil {
		t.Fatalf("read vault: %v", err)
	}
	return b
}

func reasonText(s string) *string { return &s }

// TestGrantMovesNeuronsFromVaultToStudents covers RN-05, RN-07 and RN-09.
func TestGrantMovesNeuronsFromVaultToStudents(t *testing.T) {
	db := openDB(t)
	repo := ledgerinfra.NewPostgresLedgerRepository(db)
	f := newFixture(t, db, 100, false)
	ctx := context.Background()

	batchID := uuid.NewString()
	txs, err := repo.Grant(ctx, ledger.GrantOp{
		ClassroomID: f.classroomID,
		Recipients:  f.students,
		AmountEach:  5,
		ReasonText:  reasonText("Participación"),
		Channel:     ledger.ChannelManual,
		Batch: &ledger.Batch{
			ID:               batchID,
			ClassroomID:      f.classroomID,
			Type:             ledger.BatchMultiGrant,
			AmountPerStudent: 5,
			RecipientCount:   len(f.students),
			TotalAmount:      10,
			ReasonText:       reasonText("Participación"),
			PerformedBy:      f.teacherID,
			CreatedAt:        time.Now(),
		},
		PerformedBy: f.teacherID,
	})
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txs))
	}

	// RN-07: each member receives the full amount, not a split.
	for _, id := range f.students {
		if got := balanceOf(t, db, id); got != 5 {
			t.Errorf("student %s balance = %d, want 5", id, got)
		}
	}
	if got := vaultOf(t, db, f.classroomID); got != 90 {
		t.Errorf("vault = %d, want 90", got)
	}
}

// TestGrantRejectsOverdraft covers RN-05.
func TestGrantRejectsOverdraft(t *testing.T) {
	db := openDB(t)
	repo := ledgerinfra.NewPostgresLedgerRepository(db)
	f := newFixture(t, db, 5, false)

	_, err := repo.Grant(context.Background(), ledger.GrantOp{
		ClassroomID: f.classroomID,
		Recipients:  f.students,
		AmountEach:  10,
		ReasonText:  reasonText("Demasiado"),
		Channel:     ledger.ChannelManual,
		PerformedBy: f.teacherID,
	})
	if err == nil {
		t.Fatal("expected insufficient vault error")
	}
	if got := vaultOf(t, db, f.classroomID); got != 5 {
		t.Errorf("vault changed on failed grant: %d", got)
	}
}

// TestUnlimitedIssuanceDoesNotDrainVault covers decision 15.1.
func TestUnlimitedIssuanceDoesNotDrainVault(t *testing.T) {
	db := openDB(t)
	repo := ledgerinfra.NewPostgresLedgerRepository(db)
	f := newFixture(t, db, 0, true)

	if _, err := repo.Grant(context.Background(), ledger.GrantOp{
		ClassroomID: f.classroomID,
		Recipients:  f.students[:1],
		AmountEach:  1000,
		ReasonText:  reasonText("Ilimitado"),
		Channel:     ledger.ChannelManual,
		PerformedBy: f.teacherID,
	}); err != nil {
		t.Fatalf("grant: %v", err)
	}
	if got := balanceOf(t, db, f.students[0]); got != 1000 {
		t.Errorf("balance = %d, want 1000", got)
	}
	if got := vaultOf(t, db, f.classroomID); got != 0 {
		t.Errorf("unlimited vault should stay at 0, got %d", got)
	}
}

// TestRedeemReturnsNeuronsToVault covers RN-03 and RN-04.
func TestRedeemReturnsNeuronsToVault(t *testing.T) {
	db := openDB(t)
	repo := ledgerinfra.NewPostgresLedgerRepository(db)
	f := newFixture(t, db, 100, false)
	ctx := context.Background()

	if _, err := repo.Grant(ctx, ledger.GrantOp{
		ClassroomID: f.classroomID,
		Recipients:  f.students[:1],
		AmountEach:  10,
		ReasonText:  reasonText("Inicial"),
		Channel:     ledger.ChannelManual,
		PerformedBy: f.teacherID,
	}); err != nil {
		t.Fatalf("grant: %v", err)
	}

	if _, err := repo.Redeem(ctx, ledger.RedeemOp{
		ClassroomID:  f.classroomID,
		EnrollmentID: f.students[0],
		Amount:       4,
		BenefitText:  reasonText("Punto extra"),
		Channel:      ledger.ChannelManual,
		PerformedBy:  f.teacherID,
	}); err != nil {
		t.Fatalf("redeem: %v", err)
	}

	if got := balanceOf(t, db, f.students[0]); got != 6 {
		t.Errorf("balance = %d, want 6", got)
	}
	if got := vaultOf(t, db, f.classroomID); got != 94 {
		t.Errorf("vault = %d, want 94", got)
	}

	// RN-04: a student can never return more than they hold.
	if _, err := repo.Redeem(ctx, ledger.RedeemOp{
		ClassroomID:  f.classroomID,
		EnrollmentID: f.students[0],
		Amount:       99,
		BenefitText:  reasonText("Demasiado"),
		Channel:      ledger.ChannelManual,
		PerformedBy:  f.teacherID,
	}); err == nil {
		t.Fatal("expected insufficient balance error")
	}
	if got := balanceOf(t, db, f.students[0]); got != 6 {
		t.Errorf("balance changed on failed redeem: %d", got)
	}
}

// TestReverseRestoresBothBalances covers RN-15 and decision 15.4.
func TestReverseRestoresBothBalances(t *testing.T) {
	db := openDB(t)
	repo := ledgerinfra.NewPostgresLedgerRepository(db)
	f := newFixture(t, db, 100, false)
	ctx := context.Background()

	txs, err := repo.Grant(ctx, ledger.GrantOp{
		ClassroomID: f.classroomID,
		Recipients:  f.students[:1],
		AmountEach:  7,
		ReasonText:  reasonText("Error"),
		Channel:     ledger.ChannelManual,
		PerformedBy: f.teacherID,
	})
	if err != nil {
		t.Fatalf("grant: %v", err)
	}

	if _, err := repo.Reverse(ctx, txs[0].ID, f.teacherID, reasonText("Otorgado por error")); err != nil {
		t.Fatalf("reverse: %v", err)
	}

	if got := balanceOf(t, db, f.students[0]); got != 0 {
		t.Errorf("balance = %d, want 0", got)
	}
	if got := vaultOf(t, db, f.classroomID); got != 100 {
		t.Errorf("vault = %d, want 100", got)
	}

	// The original row survives, marked REVERSED (RN-15: nothing is deleted).
	orig, err := repo.GetByID(ctx, txs[0].ID)
	if err != nil {
		t.Fatalf("get original: %v", err)
	}
	if orig.Status != ledger.StatusReversed {
		t.Errorf("status = %s, want REVERSED", orig.Status)
	}
	if orig.ReversedByTransactionID == nil {
		t.Error("original should link to its reversal")
	}

	// A reversal can never be reversed twice.
	if _, err := repo.Reverse(ctx, txs[0].ID, f.teacherID, nil); err == nil {
		t.Fatal("expected already-reversed error")
	}
}

// TestIdempotencyKeyPreventsDoublePay covers §11.3.
func TestIdempotencyKeyPreventsDoublePay(t *testing.T) {
	db := openDB(t)
	repo := ledgerinfra.NewPostgresLedgerRepository(db)
	f := newFixture(t, db, 100, false)
	ctx := context.Background()

	key := "qr:" + uuid.NewString()
	op := ledger.GrantOp{
		ClassroomID:    f.classroomID,
		Recipients:     f.students[:1],
		AmountEach:     3,
		ReasonText:     reasonText("QR"),
		Channel:        ledger.ChannelQR,
		IdempotencyKey: &key,
		PerformedBy:    f.teacherID,
	}

	if _, err := repo.Grant(ctx, op); err != nil {
		t.Fatalf("first grant: %v", err)
	}
	if _, err := repo.Grant(ctx, op); err == nil {
		t.Fatal("expected duplicate error on replayed key")
	}
	if got := balanceOf(t, db, f.students[0]); got != 3 {
		t.Errorf("balance = %d, want 3 (paid exactly once)", got)
	}

	found, err := repo.FindByIdempotencyKey(ctx, f.classroomID, key)
	if err != nil {
		t.Fatalf("find by key: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("expected 1 transaction for key, got %d", len(found))
	}
}

// TestConcurrentGrantsNeverOverdrawVault proves the FOR UPDATE lock in RN-09
// holds: 20 racing teachers may only spend what the vault actually contains.
func TestConcurrentGrantsNeverOverdrawVault(t *testing.T) {
	db := openDB(t)
	repo := ledgerinfra.NewPostgresLedgerRepository(db)
	f := newFixture(t, db, 50, false)
	ctx := context.Background()

	const attempts = 20
	var wg sync.WaitGroup
	var granted int64
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := repo.Grant(ctx, ledger.GrantOp{
				ClassroomID: f.classroomID,
				Recipients:  f.students[:1],
				AmountEach:  10,
				ReasonText:  reasonText("Carrera"),
				Channel:     ledger.ChannelManual,
				PerformedBy: f.teacherID,
			}); err == nil {
				atomic.AddInt64(&granted, 1)
			}
		}()
	}
	wg.Wait()

	// The vault held 50 and each grant costs 10, so exactly 5 may succeed.
	if granted != 5 {
		t.Errorf("%d grants succeeded, want exactly 5", granted)
	}
	if got := vaultOf(t, db, f.classroomID); got != 0 {
		t.Errorf("vault = %d, want 0", got)
	}
	if got := balanceOf(t, db, f.students[0]); got != 50 {
		t.Errorf("student balance = %d, want 50", got)
	}
}

// TestStatsAndHistory exercises the reporting queries end to end.
func TestStatsAndHistory(t *testing.T) {
	db := openDB(t)
	repo := ledgerinfra.NewPostgresLedgerRepository(db)
	f := newFixture(t, db, 100, false)
	ctx := context.Background()

	if _, err := repo.Grant(ctx, ledger.GrantOp{
		ClassroomID: f.classroomID,
		Recipients:  f.students,
		AmountEach:  5,
		ReasonText:  reasonText("Participación"),
		Channel:     ledger.ChannelManual,
		PerformedBy: f.teacherID,
	}); err != nil {
		t.Fatalf("grant: %v", err)
	}

	stats, err := repo.Stats(ctx, f.classroomID)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.InCirculation != 10 {
		t.Errorf("in circulation = %d, want 10", stats.InCirculation)
	}
	if stats.ActiveStudents != 2 {
		t.Errorf("active students = %d, want 2", stats.ActiveStudents)
	}
	if stats.TotalGranted != 10 {
		t.Errorf("total granted = %d, want 10", stats.TotalGranted)
	}

	page, err := repo.History(ctx, f.classroomID, ledger.HistoryFilter{}, kernel.PaginationOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if page.Page.Total != 2 {
		t.Errorf("history total = %d, want 2", page.Page.Total)
	}

	ranking, err := repo.Ranking(ctx, f.classroomID, 10)
	if err != nil {
		t.Fatalf("ranking: %v", err)
	}
	if len(ranking) != 2 || ranking[0].TotalReceived != 5 {
		t.Errorf("unexpected ranking: %+v", ranking)
	}

	usage, err := repo.ReasonUsage(ctx, f.classroomID)
	if err != nil {
		t.Fatalf("reason usage: %v", err)
	}
	if len(usage) != 1 || usage[0].Uses != 2 {
		t.Errorf("unexpected reason usage: %+v", usage)
	}
}
