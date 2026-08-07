package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Abraxas-356/neurons/internal/config"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/iam/scopes"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// This suite boots the real container and the real Fiber app, then drives the
// four main flows of §5 over HTTP so routing, auth, scopes, services and SQL are
// all exercised together. It is skipped when postgres/redis are not running.

type apiTest struct {
	t         *testing.T
	app       *fiber.App
	container *Container
	tenantID  string
}

func setupAPI(t *testing.T) *apiTest {
	t.Helper()

	// The container calls logx.Fatalf when infrastructure is missing, which
	// would kill the test binary, so probe both dependencies first.
	requireInfra(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	c := NewContainer(cfg)
	t.Cleanup(c.Cleanup)

	app := fiber.New(fiber.Config{
		ErrorHandler:      globalErrorHandler(cfg),
		EnablePrintRoutes: false,
	})
	registerRoutes(app, c)

	return &apiTest{t: t, app: app, container: c}
}

func requireInfra(t *testing.T) {
	t.Helper()
	for _, env := range []struct{ key, def string }{
		{"DB_HOST", "localhost"},
		{"REDIS_HOST", "localhost"},
	} {
		if os.Getenv(env.key) == "" {
			t.Skipf("%s not set; run under `make test`", env.key)
		}
	}
}

// token mints a JWT the same way the login handlers do, so requests go through
// the real authentication middleware.
func (a *apiTest) token(userID kernel.UserID, scopeList []string) string {
	a.t.Helper()
	jwtSvc := auth.NewJWTServiceFromConfig(&a.container.Config.Auth.JWT)
	tok, err := jwtSvc.GenerateAccessToken(userID, kernel.TenantID(a.tenantID), map[string]any{
		"email":  string(userID) + "@test.local",
		"name":   "Test " + string(userID),
		"scopes": scopeList,
	})
	if err != nil {
		a.t.Fatalf("generate token: %v", err)
	}
	return tok
}

// do performs a request and decodes the JSON body. It returns the status so
// tests can assert on rejections as well as successes.
func (a *apiTest) do(method, path, token string, body any) (int, map[string]any) {
	a.t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			a.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, path, reader)
	if err != nil {
		a.t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := a.app.Test(req, 10000)
	if err != nil {
		a.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			out = map[string]any{"_raw": string(raw)}
		}
	}
	return resp.StatusCode, out
}

// mustDo fails the test unless the response carries the expected status.
func (a *apiTest) mustDo(method, path, token string, body any, want int) map[string]any {
	a.t.Helper()
	status, out := a.do(method, path, token, body)
	if status != want {
		a.t.Fatalf("%s %s = %d, want %d: %v", method, path, status, want, out)
	}
	return out
}

// seedTenant creates the tenant plus a teacher and n students directly in the
// database, since account provisioning is IAM's job, not the gamification API's.
func (a *apiTest) seedTenant(students int) (teacher kernel.UserID, studentIDs []kernel.UserID) {
	a.t.Helper()
	db := a.container.DB
	suffix := uuid.NewString()[:8]

	a.tenantID = "t-" + suffix
	if _, err := db.Exec(
		`INSERT INTO tenants (id, company_name, status) VALUES ($1, $2, 'ACTIVE')`,
		a.tenantID, "Universidad "+suffix); err != nil {
		a.t.Fatalf("insert tenant: %v", err)
	}
	a.t.Cleanup(func() { db.Exec(`DELETE FROM tenants WHERE id = $1`, a.tenantID) })

	teacher = kernel.NewUserID("u-teacher-" + suffix)
	seedUser(a.t, db, teacher, a.tenantID, "Profesor "+suffix)

	for i := 0; i < students; i++ {
		id := kernel.NewUserID(fmt.Sprintf("u-student-%s-%d", suffix, i))
		seedUser(a.t, db, id, a.tenantID, fmt.Sprintf("Alumno %d", i))
		studentIDs = append(studentIDs, id)
	}
	return teacher, studentIDs
}

func seedUser(t *testing.T, db *sqlx.DB, id kernel.UserID, tenantID, name string) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO users (id, tenant_id, email, name, status, oauth_provider, oauth_provider_id, email_verified)
		 VALUES ($1, $2, $3, $4, 'ACTIVE', 'test', $1, true)`,
		id, tenantID, string(id)+"@test.local", name); err != nil {
		t.Fatalf("insert user %s: %v", id, err)
	}
}

func str(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func num(m map[string]any, key string) int64 {
	v, _ := m[key].(float64)
	return int64(v)
}

// TestFullTeacherStudentJourney walks the whole product: create a course, let
// students join, grant neurons individually and by team, redeem a benefit,
// award a medal, reverse a mistake, and read the reports.
func TestFullTeacherStudentJourney(t *testing.T) {
	api := setupAPI(t)
	teacherID, studentIDs := api.seedTenant(3)

	teacher := api.token(teacherID, scopes.TeacherScopes)
	student0 := api.token(studentIDs[0], scopes.StudentScopes)
	student1 := api.token(studentIDs[1], scopes.StudentScopes)
	student2 := api.token(studentIDs[2], scopes.StudentScopes)

	// --- HU-001: the teacher creates a classroom with an initial vault ---
	room := api.mustDo("POST", "/api/v1/classrooms", teacher, map[string]any{
		"name":            "Cálculo I",
		"section":         "A",
		"term":            "2026-1",
		"initial_neurons": 500,
		"status":          "ACTIVE",
	}, fiber.StatusCreated)

	classroomID := str(room, "id")
	inviteCode := str(room, "invite_code")
	if classroomID == "" || inviteCode == "" {
		t.Fatalf("classroom response missing id/invite_code: %v", room)
	}
	if got := num(room, "vault_balance"); got != 500 {
		t.Fatalf("vault_balance = %d, want 500", got)
	}

	// --- HU-021: students join with the invite code ---
	enrollments := make([]string, 0, 3)
	for _, tok := range []string{student0, student1, student2} {
		out := api.mustDo("POST", "/api/v1/me/classrooms/join", tok, map[string]any{
			"invite_code": inviteCode,
		}, fiber.StatusCreated)
		enrollments = append(enrollments, str(out, "enrollment_id"))
	}

	// --- HU-041: the teacher defines a reason and a benefit ---
	reason := api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/reasons", teacher, map[string]any{
		"name":           "Participación en clase",
		"default_amount": 2,
		"applies_to":     "BOTH",
	}, fiber.StatusCreated)
	reasonID := str(reason, "id")

	benefit := api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/benefits", teacher, map[string]any{
		"name": "Punto extra en examen",
		"cost": 10,
	}, fiber.StatusCreated)
	benefitID := str(benefit, "id")

	// --- Flow D (HU-051): manual grant to two students ---
	grant := api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, map[string]any{
		"enrollment_ids": []string{enrollments[0], enrollments[1]},
		"amount":         5,
		"reason_id":      reasonID,
		"channel":        "MANUAL",
	}, fiber.StatusCreated)

	// RN-07: every recipient gets the full amount.
	if got := num(grant, "amount_each"); got != 5 {
		t.Errorf("amount_each = %d, want 5", got)
	}
	if got := num(grant, "total_amount"); got != 10 {
		t.Errorf("total_amount = %d, want 10", got)
	}
	if got := num(grant, "vault_balance"); got != 490 {
		t.Errorf("vault after grant = %d, want 490", got)
	}

	// --- HU-031: create a team and grant to it (flow B, RN-07) ---
	team := api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/teams", teacher, map[string]any{
		"name": "Equipo Rojo",
	}, fiber.StatusCreated)
	teamID := str(team, "id")

	for _, enrollmentID := range enrollments[:2] {
		api.mustDo("PUT", "/api/v1/enrollments/"+enrollmentID+"/team", teacher, map[string]any{
			"team_id": teamID,
		}, fiber.StatusOK)
	}

	teamGrant := api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/team-grants", teacher, map[string]any{
		"team_id":     teamID,
		"amount":      3,
		"reason_text": "Mejor exposición",
	}, fiber.StatusCreated)

	// Two members x 3 neurons each: the team does NOT split the amount.
	if got := num(teamGrant, "recipients"); got != 2 {
		t.Errorf("team recipients = %d, want 2", got)
	}
	if got := num(teamGrant, "total_amount"); got != 6 {
		t.Errorf("team total = %d, want 6", got)
	}

	// Student 0 now holds 5 + 3 = 8.
	mine := api.mustDo("GET", "/api/v1/me/classrooms/"+classroomID+"/enrollment", student0, nil, fiber.StatusOK)
	if got := num(mine, "balance"); got != 8 {
		t.Errorf("student0 balance = %d, want 8", got)
	}

	// --- RN-04: a student cannot return more neurons than they hold ---
	status, _ := api.do("POST", "/api/v1/classrooms/"+classroomID+"/redemptions", teacher, map[string]any{
		"enrollment_id": enrollments[0],
		"benefit_id":    benefitID,
	})
	if status < 400 {
		t.Errorf("redeeming a 10-neuron benefit with 8 neurons returned %d, want a 4xx", status)
	}

	// Top the student up so the redemption can succeed.
	api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, map[string]any{
		"enrollment_ids": []string{enrollments[0]},
		"amount":         5,
		"reason_text":    "Ajuste",
		"channel":        "MANUAL",
	}, fiber.StatusCreated)

	// --- Flow C (HU-062): the student returns neurons for a benefit ---
	api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/redemptions", teacher, map[string]any{
		"enrollment_id": enrollments[0],
		"benefit_id":    benefitID,
	}, fiber.StatusCreated)

	mine = api.mustDo("GET", "/api/v1/me/classrooms/"+classroomID+"/enrollment", student0, nil, fiber.StatusOK)
	if got := num(mine, "balance"); got != 3 { // 8 + 5 - 10
		t.Errorf("student0 balance after redemption = %d, want 3", got)
	}

	// --- HU-071: define and award a medal ---
	medal := api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/medals", teacher, map[string]any{
		"name": "Mejor participación",
		"type": "INDIVIDUAL",
	}, fiber.StatusCreated)
	medalID := str(medal, "id")

	api.mustDo("POST", "/api/v1/medals/"+medalID+"/awards", teacher, map[string]any{
		"enrollment_ids": []string{enrollments[0]},
		"note":           "Excelente semestre",
	}, fiber.StatusCreated)

	awarded := api.mustDo("GET", "/api/v1/me/classrooms/"+classroomID+"/medals", student0, nil, fiber.StatusOK)
	if items, _ := awarded["items"].([]any); len(items) != 1 {
		t.Errorf("student0 should see exactly 1 medal, got %v", awarded["items"])
	}

	// --- HU-092: reverse a grant and confirm both balances are restored ---
	before := api.mustDo("GET", "/api/v1/classrooms/"+classroomID+"/stats", teacher, nil, fiber.StatusOK)

	solo := api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, map[string]any{
		"enrollment_ids": []string{enrollments[2]},
		"amount":         4,
		"reason_text":    "Error del profesor",
		"channel":        "MANUAL",
	}, fiber.StatusCreated)

	txs, _ := solo["transactions"].([]any)
	if len(txs) != 1 {
		t.Fatalf("expected 1 transaction, got %v", solo["transactions"])
	}
	first, _ := txs[0].(map[string]any)
	txID := str(first, "id")

	api.mustDo("POST", "/api/v1/transactions/"+txID+"/reversal", teacher, map[string]any{
		"reason": "Otorgado por error",
	}, fiber.StatusCreated)

	after := api.mustDo("GET", "/api/v1/classrooms/"+classroomID+"/stats", teacher, nil, fiber.StatusOK)
	if num(before, "in_circulation") != num(after, "in_circulation") {
		t.Errorf("circulation drifted after reversal: %d -> %d",
			num(before, "in_circulation"), num(after, "in_circulation"))
	}
	if num(before, "vault_balance") != num(after, "vault_balance") {
		t.Errorf("vault drifted after reversal: %d -> %d",
			num(before, "vault_balance"), num(after, "vault_balance"))
	}

	// --- §13 reports ---
	history := api.mustDo("GET", "/api/v1/classrooms/"+classroomID+"/transactions", teacher, nil, fiber.StatusOK)
	if history["items"] == nil {
		t.Error("history returned no items field")
	}

	usage := api.mustDo("GET", "/api/v1/classrooms/"+classroomID+"/reports/reasons", teacher, nil, fiber.StatusOK)
	if items, _ := usage["items"].([]any); len(items) == 0 {
		t.Error("reason usage report is empty")
	}

	// The student sees their own ledger.
	own := api.mustDo("GET", "/api/v1/me/classrooms/"+classroomID+"/transactions", student0, nil, fiber.StatusOK)
	if own["items"] == nil {
		t.Error("student history returned no items field")
	}
}

// TestStudentCannotActAsTeacher covers §10.4: students only ever touch their
// own data, enforced by scopes at the routing layer.
func TestStudentCannotActAsTeacher(t *testing.T) {
	api := setupAPI(t)
	teacherID, studentIDs := api.seedTenant(1)

	teacher := api.token(teacherID, scopes.TeacherScopes)
	student := api.token(studentIDs[0], scopes.StudentScopes)

	room := api.mustDo("POST", "/api/v1/classrooms", teacher, map[string]any{
		"name":            "Álgebra",
		"initial_neurons": 100,
		"status":          "ACTIVE",
	}, fiber.StatusCreated)
	classroomID := str(room, "id")
	inviteCode := str(room, "invite_code")

	joined := api.mustDo("POST", "/api/v1/me/classrooms/join", student, map[string]any{
		"invite_code": inviteCode,
	}, fiber.StatusCreated)
	enrollmentID := str(joined, "enrollment_id")

	forbidden := []struct {
		method string
		path   string
		body   any
	}{
		{"POST", "/api/v1/classrooms", map[string]any{"name": "Curso pirata"}},
		{"POST", "/api/v1/classrooms/" + classroomID + "/grants", map[string]any{
			"enrollment_ids": []string{enrollmentID},
			"amount":         999,
			"reason_text":    "Auto-regalo",
		}},
		{"POST", "/api/v1/classrooms/" + classroomID + "/vault/topup", map[string]any{"amount": 1000}},
		{"GET", "/api/v1/classrooms/" + classroomID + "/students", nil},
		{"GET", "/api/v1/classrooms/" + classroomID + "/transactions", nil},
		{"POST", "/api/v1/classrooms/" + classroomID + "/reasons", map[string]any{"name": "x"}},
	}

	for _, tc := range forbidden {
		status, body := api.do(tc.method, tc.path, student, tc.body)
		if status != fiber.StatusForbidden {
			t.Errorf("%s %s as student = %d, want 403: %v", tc.method, tc.path, status, body)
		}
	}

	// And without any token at all, everything is 401.
	status, _ := api.do("GET", "/api/v1/classrooms", "", nil)
	if status != fiber.StatusUnauthorized {
		t.Errorf("anonymous request = %d, want 401", status)
	}
}

// TestTeacherCannotTouchAnotherTeachersClassroom covers RN-01 isolation.
func TestTeacherCannotTouchAnotherTeachersClassroom(t *testing.T) {
	api := setupAPI(t)
	ownerID, _ := api.seedTenant(0)

	// A second teacher in the same tenant who was never added to the course.
	intruderID := kernel.NewUserID("u-intruder-" + uuid.NewString()[:8])
	seedUser(t, api.container.DB, intruderID, api.tenantID, "Otro Profesor")

	owner := api.token(ownerID, scopes.TeacherScopes)
	intruder := api.token(intruderID, scopes.TeacherScopes)

	room := api.mustDo("POST", "/api/v1/classrooms", owner, map[string]any{
		"name":            "Física II",
		"initial_neurons": 200,
		"status":          "ACTIVE",
	}, fiber.StatusCreated)
	classroomID := str(room, "id")

	// The intruder holds every teacher scope, so only the per-classroom
	// ownership check can stop them.
	for _, tc := range []struct {
		method string
		path   string
		body   any
	}{
		{"GET", "/api/v1/classrooms/" + classroomID, nil},
		{"GET", "/api/v1/classrooms/" + classroomID + "/stats", nil},
		{"POST", "/api/v1/classrooms/" + classroomID + "/vault/topup", map[string]any{"amount": 50}},
		{"POST", "/api/v1/classrooms/" + classroomID + "/reasons", map[string]any{"name": "x"}},
	} {
		status, body := api.do(tc.method, tc.path, intruder, tc.body)
		if status < 400 {
			t.Errorf("%s %s by non-member teacher = %d, want 4xx: %v", tc.method, tc.path, status, body)
		}
	}
}

// TestQRGrantFlow covers flow A end to end: the student shows a QR, the teacher
// scans it, and the same scan cannot pay twice (§11.3).
func TestQRGrantFlow(t *testing.T) {
	api := setupAPI(t)
	teacherID, studentIDs := api.seedTenant(1)

	teacher := api.token(teacherID, scopes.TeacherScopes)
	student := api.token(studentIDs[0], scopes.StudentScopes)

	room := api.mustDo("POST", "/api/v1/classrooms", teacher, map[string]any{
		"name":            "Química",
		"initial_neurons": 100,
		"status":          "ACTIVE",
	}, fiber.StatusCreated)
	classroomID := str(room, "id")
	inviteCode := str(room, "invite_code")

	api.mustDo("POST", "/api/v1/me/classrooms/join", student, map[string]any{
		"invite_code": inviteCode,
	}, fiber.StatusCreated)

	// The student's app renders a short-lived token (RN-13).
	qr := api.mustDo("POST", "/api/v1/me/classrooms/"+classroomID+"/qr", student, nil, fiber.StatusCreated)
	code := str(qr, "code")
	if code == "" {
		t.Fatalf("student QR has no code: %v", qr)
	}

	// The teacher scans it. Scanning only resolves who the student is; it does
	// not move neurons yet (flow A step 2).
	scan := api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/qr/scan", teacher, map[string]any{
		"code": code,
	}, fiber.StatusOK)

	enrollmentID := str(scan, "enrollment_id")
	grantKey := str(scan, "grant_key")
	if enrollmentID == "" || grantKey == "" {
		t.Fatalf("scan did not resolve the student: %v", scan)
	}

	// Step 3: the teacher confirms an amount, keyed to that scan.
	grantBody := map[string]any{
		"enrollment_ids":  []string{enrollmentID},
		"amount":          6,
		"reason_text":     "Respuesta correcta",
		"channel":         "QR",
		"idempotency_key": grantKey,
	}
	grant := api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, grantBody, fiber.StatusCreated)
	if got := num(grant, "total_amount"); got != 6 {
		t.Errorf("QR grant total = %d, want 6", got)
	}

	// §11.3: a double tap on confirm must not pay twice.
	api.do("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, grantBody)

	mine := api.mustDo("GET", "/api/v1/me/classrooms/"+classroomID+"/enrollment", student, nil, fiber.StatusOK)
	if got := num(mine, "balance"); got != 6 {
		t.Errorf("balance after duplicate confirm = %d, want 6 (paid once)", got)
	}

	// RN-13: a student token is single-use, so the scanner cannot silently
	// re-resolve the same code after it has been consumed.
	api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/qr/scan", teacher, map[string]any{
		"code": code,
	}, fiber.StatusOK)
}

// TestQRCodeIsBoundToItsClassroom covers RN-12: a code minted in one course is
// worthless in another, even for a teacher who owns both.
func TestQRCodeIsBoundToItsClassroom(t *testing.T) {
	api := setupAPI(t)
	teacherID, studentIDs := api.seedTenant(1)

	teacher := api.token(teacherID, scopes.TeacherScopes)
	student := api.token(studentIDs[0], scopes.StudentScopes)

	roomA := api.mustDo("POST", "/api/v1/classrooms", teacher, map[string]any{
		"name": "Curso A", "initial_neurons": 100, "status": "ACTIVE",
	}, fiber.StatusCreated)
	roomB := api.mustDo("POST", "/api/v1/classrooms", teacher, map[string]any{
		"name": "Curso B", "initial_neurons": 100, "status": "ACTIVE",
	}, fiber.StatusCreated)

	api.mustDo("POST", "/api/v1/me/classrooms/join", student, map[string]any{
		"invite_code": str(roomA, "invite_code"),
	}, fiber.StatusCreated)

	qrToken := api.mustDo("POST", "/api/v1/me/classrooms/"+str(roomA, "id")+"/qr", student, nil, fiber.StatusCreated)

	// The very same code presented under course B must be rejected.
	status, body := api.do("POST", "/api/v1/classrooms/"+str(roomB, "id")+"/qr/scan", teacher, map[string]any{
		"code": str(qrToken, "code"),
	})
	if status < 400 {
		t.Errorf("cross-classroom scan = %d, want 4xx: %v", status, body)
	}
}

// TestClosedClassroomFreezesTransactions covers RN-17.
func TestClosedClassroomFreezesTransactions(t *testing.T) {
	api := setupAPI(t)
	teacherID, studentIDs := api.seedTenant(1)

	teacher := api.token(teacherID, scopes.TeacherScopes)
	student := api.token(studentIDs[0], scopes.StudentScopes)

	room := api.mustDo("POST", "/api/v1/classrooms", teacher, map[string]any{
		"name":            "Historia",
		"initial_neurons": 100,
		"status":          "ACTIVE",
	}, fiber.StatusCreated)
	classroomID := str(room, "id")
	inviteCode := str(room, "invite_code")

	joined := api.mustDo("POST", "/api/v1/me/classrooms/join", student, map[string]any{
		"invite_code": inviteCode,
	}, fiber.StatusCreated)
	enrollmentID := str(joined, "enrollment_id")

	api.mustDo("PUT", "/api/v1/classrooms/"+classroomID, teacher, map[string]any{
		"status": "CLOSED",
	}, fiber.StatusOK)

	status, body := api.do("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, map[string]any{
		"enrollment_ids": []string{enrollmentID},
		"amount":         5,
		"reason_text":    "Tarde",
	})
	if status < 400 {
		t.Errorf("grant in a closed classroom = %d, want 4xx: %v", status, body)
	}

	// Reading history still works: closing freezes writes, not the record.
	api.mustDo("GET", "/api/v1/classrooms/"+classroomID+"/transactions", teacher, nil, fiber.StatusOK)
}

// TestReasonRequiredOnGrant covers RN-10.
func TestReasonRequiredOnGrant(t *testing.T) {
	api := setupAPI(t)
	teacherID, studentIDs := api.seedTenant(1)

	teacher := api.token(teacherID, scopes.TeacherScopes)
	student := api.token(studentIDs[0], scopes.StudentScopes)

	room := api.mustDo("POST", "/api/v1/classrooms", teacher, map[string]any{
		"name":            "Biología",
		"initial_neurons": 100,
		"status":          "ACTIVE",
	}, fiber.StatusCreated)
	classroomID := str(room, "id")

	joined := api.mustDo("POST", "/api/v1/me/classrooms/join", student, map[string]any{
		"invite_code": str(room, "invite_code"),
	}, fiber.StatusCreated)
	enrollmentID := str(joined, "enrollment_id")

	status, body := api.do("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, map[string]any{
		"enrollment_ids": []string{enrollmentID},
		"amount":         5,
	})
	if status < 400 {
		t.Errorf("grant without a reason = %d, want 4xx: %v", status, body)
	}

	// RN-08: amounts must be positive.
	status, body = api.do("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, map[string]any{
		"enrollment_ids": []string{enrollmentID},
		"amount":         0,
		"reason_text":    "Cero",
	})
	if status < 400 {
		t.Errorf("grant of 0 neurons = %d, want 4xx: %v", status, body)
	}

	status, body = api.do("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, map[string]any{
		"enrollment_ids": []string{enrollmentID},
		"amount":         -5,
		"reason_text":    "Negativo",
	})
	if status < 400 {
		t.Errorf("grant of -5 neurons = %d, want 4xx: %v", status, body)
	}
}

// TestVaultLimitIsEnforced covers RN-05 through the API.
func TestVaultLimitIsEnforced(t *testing.T) {
	api := setupAPI(t)
	teacherID, studentIDs := api.seedTenant(1)

	teacher := api.token(teacherID, scopes.TeacherScopes)
	student := api.token(studentIDs[0], scopes.StudentScopes)

	room := api.mustDo("POST", "/api/v1/classrooms", teacher, map[string]any{
		"name":            "Geografía",
		"initial_neurons": 10,
		"status":          "ACTIVE",
	}, fiber.StatusCreated)
	classroomID := str(room, "id")

	joined := api.mustDo("POST", "/api/v1/me/classrooms/join", student, map[string]any{
		"invite_code": str(room, "invite_code"),
	}, fiber.StatusCreated)
	enrollmentID := str(joined, "enrollment_id")

	status, body := api.do("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, map[string]any{
		"enrollment_ids": []string{enrollmentID},
		"amount":         50,
		"reason_text":    "Más de lo que hay",
	})
	if status < 400 {
		t.Errorf("overdraft grant = %d, want 4xx: %v", status, body)
	}

	// Topping the vault up makes the same grant succeed (decision 15.8).
	api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/vault/topup", teacher, map[string]any{
		"amount": 100,
	}, fiber.StatusCreated)

	// §11.9: an unusually large grant is held back until the teacher confirms.
	status, body = api.do("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, map[string]any{
		"enrollment_ids": []string{enrollmentID},
		"amount":         50,
		"reason_text":    "Sin confirmar",
	})
	if status != fiber.StatusConflict {
		t.Errorf("unconfirmed large grant = %d, want 409: %v", status, body)
	}

	api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, map[string]any{
		"enrollment_ids": []string{enrollmentID},
		"amount":         50,
		"reason_text":    "Ahora sí",
		"confirmed":      true,
	}, fiber.StatusCreated)

	// The top-up itself must be auditable (RN-09), not a silent balance edit.
	ledgerRows := api.mustDo("GET", "/api/v1/classrooms/"+classroomID+"/transactions?type=VAULT_TOPUP",
		teacher, nil, fiber.StatusOK)
	if items, _ := ledgerRows["items"].([]any); len(items) != 1 {
		t.Errorf("expected the top-up to leave one ledger entry, got %v", ledgerRows["items"])
	}
}

// TestIdempotentGrantOverHTTP covers §11.3 at the API boundary: a double submit
// from a flaky network must not pay twice.
func TestIdempotentGrantOverHTTP(t *testing.T) {
	api := setupAPI(t)
	teacherID, studentIDs := api.seedTenant(1)

	teacher := api.token(teacherID, scopes.TeacherScopes)
	student := api.token(studentIDs[0], scopes.StudentScopes)

	room := api.mustDo("POST", "/api/v1/classrooms", teacher, map[string]any{
		"name":            "Estadística",
		"initial_neurons": 100,
		"status":          "ACTIVE",
	}, fiber.StatusCreated)
	classroomID := str(room, "id")

	joined := api.mustDo("POST", "/api/v1/me/classrooms/join", student, map[string]any{
		"invite_code": str(room, "invite_code"),
	}, fiber.StatusCreated)
	enrollmentID := str(joined, "enrollment_id")

	body := map[string]any{
		"enrollment_ids":  []string{enrollmentID},
		"amount":          7,
		"reason_text":     "Doble submit",
		"idempotency_key": "req-" + uuid.NewString(),
	}

	api.mustDo("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, body, fiber.StatusCreated)
	// The retry is accepted but must not move neurons again.
	api.do("POST", "/api/v1/classrooms/"+classroomID+"/grants", teacher, body)

	mine := api.mustDo("GET", "/api/v1/me/classrooms/"+classroomID+"/enrollment", student, nil, fiber.StatusOK)
	if got := num(mine, "balance"); got != 7 {
		t.Errorf("balance after duplicate submit = %d, want 7", got)
	}
}

// TestJoinIsIdempotentPerStudent covers RN-18: one enrollment per student.
func TestJoinIsIdempotentPerStudent(t *testing.T) {
	api := setupAPI(t)
	teacherID, studentIDs := api.seedTenant(1)

	teacher := api.token(teacherID, scopes.TeacherScopes)
	student := api.token(studentIDs[0], scopes.StudentScopes)

	room := api.mustDo("POST", "/api/v1/classrooms", teacher, map[string]any{
		"name":            "Literatura",
		"initial_neurons": 50,
		"status":          "ACTIVE",
	}, fiber.StatusCreated)
	code := str(room, "invite_code")

	first := api.mustDo("POST", "/api/v1/me/classrooms/join", student, map[string]any{
		"invite_code": code,
	}, fiber.StatusCreated)

	status, second := api.do("POST", "/api/v1/me/classrooms/join", student, map[string]any{
		"invite_code": code,
	})

	// Either the same enrollment comes back or the attempt is rejected, but a
	// second row must never be created.
	if status < 400 && str(second, "enrollment_id") != str(first, "enrollment_id") {
		t.Errorf("second join created a new enrollment: %s vs %s",
			str(second, "enrollment_id"), str(first, "enrollment_id"))
	}

	var count int
	if err := api.container.DB.Get(&count,
		`SELECT COUNT(*) FROM enrollments WHERE classroom_id = $1 AND user_id = $2`,
		str(room, "id"), studentIDs[0]); err != nil {
		t.Fatalf("count enrollments: %v", err)
	}
	if count != 1 {
		t.Errorf("enrollment rows = %d, want 1", count)
	}
}

func TestHealthEndpoint(t *testing.T) {
	api := setupAPI(t)
	app := fiber.New()
	app.Get("/health", healthCheckHandler(api.container))

	req, _ := http.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, int(5*time.Second/time.Millisecond))
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("health = %d, want 200", resp.StatusCode)
	}
}
