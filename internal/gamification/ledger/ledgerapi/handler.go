package ledgerapi

import (
	"time"

	"github.com/Abraxas-356/neurons/internal/gamification/httpx"
	"github.com/Abraxas-356/neurons/internal/gamification/ledger"
	"github.com/Abraxas-356/neurons/internal/gamification/ledger/ledgersrv"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/iam/scopes"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/gofiber/fiber/v2"
)

type LedgerHandlers struct {
	service *ledgersrv.LedgerService
}

func NewLedgerHandlers(service *ledgersrv.LedgerService) *LedgerHandlers {
	return &LedgerHandlers{service: service}
}

func (h *LedgerHandlers) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	cr := router.Group("/classrooms/:classroomId")

	// Movements
	cr.Post("/grants", mw.RequireScope(scopes.ScopeNeuronsGrant), h.Grant)
	cr.Post("/team-grants", mw.RequireScope(scopes.ScopeNeuronsGrant), h.GrantToTeam)
	cr.Post("/redemptions", mw.RequireScope(scopes.ScopeNeuronsRedeem), h.Redeem)
	cr.Post("/vault/topup", mw.RequireScope(scopes.ScopeNeuronsGrant), h.Topup)

	// Ledger and reports
	cr.Get("/transactions", mw.RequireScope(scopes.ScopeNeuronsRead), h.History)
	cr.Get("/stats", mw.RequireScope(scopes.ScopeNeuronsRead), h.Stats)
	cr.Get("/reports/reasons", mw.RequireScope(scopes.ScopeNeuronsRead), h.ReasonUsage)
	cr.Get("/ranking", mw.RequireAnyScope(scopes.ScopeNeuronsRead, scopes.ScopeStudentSelf), h.Ranking)

	router.Get("/enrollments/:enrollmentId/transactions", mw.RequireScope(scopes.ScopeNeuronsRead), h.StudentHistory)
	router.Get("/transactions/:id", mw.RequireScope(scopes.ScopeNeuronsRead), h.GetByID)
	router.Post("/transactions/:id/reversal", mw.RequireScope(scopes.ScopeNeuronsVoid), h.Reverse)
	router.Get("/batches/:batchId", mw.RequireScope(scopes.ScopeNeuronsRead), h.GetBatch)

	// Student self-service (§10.4: own data only)
	router.Get("/me/classrooms/:classroomId/transactions", mw.RequireScope(scopes.ScopeStudentSelf), h.MyHistory)
}

func (h *LedgerHandlers) Grant(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req ledger.GrantRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	result, err := h.service.Grant(c.Context(), classroomID, actor.UserID, req, deviceInfo(c))
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *LedgerHandlers) GrantToTeam(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req ledger.TeamGrantRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	result, err := h.service.GrantToTeam(c.Context(), classroomID, actor.UserID, req, deviceInfo(c))
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *LedgerHandlers) Redeem(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req ledger.RedeemRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	t, err := h.service.Redeem(c.Context(), classroomID, actor.UserID, req, deviceInfo(c))
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(t)
}

func (h *LedgerHandlers) Topup(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req ledger.TopupRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	t, err := h.service.Topup(c.Context(), classroomID, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(t)
}

func (h *LedgerHandlers) History(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	filter, err := parseFilter(c)
	if err != nil {
		return err
	}

	page, err := h.service.History(c.Context(), classroomID, actor.UserID, filter, httpx.Pagination(c))
	if err != nil {
		return err
	}

	return c.JSON(page)
}

func (h *LedgerHandlers) MyHistory(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	page, err := h.service.MyHistory(c.Context(), classroomID, actor.UserID, httpx.Pagination(c))
	if err != nil {
		return err
	}

	return c.JSON(page)
}

func (h *LedgerHandlers) StudentHistory(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	enrollmentID := kernel.NewEnrollmentID(c.Params("enrollmentId"))

	page, err := h.service.StudentHistory(c.Context(), enrollmentID, actor.UserID, httpx.Pagination(c))
	if err != nil {
		return err
	}

	return c.JSON(page)
}

func (h *LedgerHandlers) GetByID(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}

	t, err := h.service.Get(c.Context(), kernel.NewLedgerID(c.Params("id")), actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(t)
}

func (h *LedgerHandlers) Reverse(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}

	var req ledger.ReverseRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	t, err := h.service.Reverse(c.Context(), kernel.NewLedgerID(c.Params("id")), actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(t)
}

func (h *LedgerHandlers) Stats(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	stats, err := h.service.Stats(c.Context(), classroomID, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(stats)
}

func (h *LedgerHandlers) ReasonUsage(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	items, err := h.service.ReasonUsage(c.Context(), classroomID, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"items": items})
}

func (h *LedgerHandlers) Ranking(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	items, err := h.service.Ranking(c.Context(), classroomID, actor.UserID, c.QueryInt("limit", 20))
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"items": items})
}

func (h *LedgerHandlers) GetBatch(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}

	batch, items, err := h.service.GetBatch(c.Context(), c.Params("batchId"), actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"batch": batch, "transactions": items})
}

// --- helpers ---

// deviceInfo records the caller's user agent for the audit trail (§11.6).
func deviceInfo(c *fiber.Ctx) *string {
	ua := c.Get(fiber.HeaderUserAgent)
	if ua == "" {
		return nil
	}
	if len(ua) > 500 {
		ua = ua[:500]
	}
	return &ua
}

func parseFilter(c *fiber.Ctx) (ledger.HistoryFilter, error) {
	var f ledger.HistoryFilter

	if v := c.Query("enrollment_id"); v != "" {
		id := kernel.NewEnrollmentID(v)
		f.EnrollmentID = &id
	}
	if v := c.Query("team_id"); v != "" {
		id := kernel.NewTeamID(v)
		f.TeamID = &id
	}
	if v := c.Query("type"); v != "" {
		t := ledger.Type(v)
		f.Type = &t
	}
	if v := c.Query("reason_id"); v != "" {
		id := kernel.NewReasonID(v)
		f.ReasonID = &id
	}
	if v := c.Query("benefit_id"); v != "" {
		id := kernel.NewBenefitID(v)
		f.BenefitID = &id
	}
	if v := c.Query("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, httpx.ErrBadRequest("from must be an RFC3339 timestamp")
		}
		f.From = &t
	}
	if v := c.Query("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, httpx.ErrBadRequest("to must be an RFC3339 timestamp")
		}
		f.To = &t
	}

	return f, nil
}
