package enrollmentapi

import (
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/httpx"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/iam/scopes"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/gofiber/fiber/v2"
)

type EnrollmentHandlers struct {
	service *enrollmentsrv.EnrollmentService
}

func NewEnrollmentHandlers(service *enrollmentsrv.EnrollmentService) *EnrollmentHandlers {
	return &EnrollmentHandlers{service: service}
}

func (h *EnrollmentHandlers) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	// --- Student self-service ---
	me := router.Group("/me", mw.RequireScope(scopes.ScopeStudentSelf))
	me.Post("/classrooms/join", h.JoinByCode)
	me.Get("/classrooms", h.MyClassrooms)
	me.Get("/classrooms/:classroomId/enrollment", h.MyEnrollment)

	// --- Teacher roster management ---
	cr := router.Group("/classrooms/:classroomId")
	cr.Get("/students", mw.RequireScope(scopes.ScopeEnrollmentsRead), h.Roster)
	cr.Post("/students", mw.RequireScope(scopes.ScopeEnrollmentsWrite), h.InviteStudents)

	en := router.Group("/enrollments", mw.RequireScope(scopes.ScopeEnrollmentsWrite))
	en.Put("/:id", h.Update)
	en.Post("/:id/approve", h.Approve)
	en.Post("/:id/withdraw", h.Withdraw)
	en.Put("/:id/team", h.SetTeam)
}

func (h *EnrollmentHandlers) JoinByCode(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}

	var req enrollment.JoinByCodeRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	e, err := h.service.JoinByCode(c.Context(), actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"enrollment_id": e.ID,
		"classroom_id":  e.ClassroomID,
		"status":        e.Status,
		"balance":       e.Balance,
	})
}

func (h *EnrollmentHandlers) MyClassrooms(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}

	items, err := h.service.MyClassrooms(c.Context(), actor.TenantID, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"items": items})
}

func (h *EnrollmentHandlers) MyEnrollment(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	e, room, err := h.service.MyEnrollment(c.Context(), classroomID, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"enrollment_id":  e.ID,
		"classroom_id":   e.ClassroomID,
		"classroom_name": room.Name,
		"classroom_open": room.Status == "ACTIVE",
		"section":        room.Section,
		"term":           room.Term,
		"icon":           room.Icon,
		"balance":        e.Balance,
		"total_received": e.TotalReceived,
		"total_returned": e.TotalReturned,
		"team_id":        e.TeamID,
		"team_name":      e.TeamName,
		"status":         e.Status,
	})
}

func (h *EnrollmentHandlers) Roster(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	filter := enrollment.RosterFilter{
		Search:  c.Query("search"),
		SortBy:  c.Query("sort_by"),
		SortDir: c.Query("sort_dir"),
	}
	if raw := c.Query("status"); raw != "" {
		status := enrollment.Status(raw)
		filter.Status = &status
	}
	if raw := c.Query("team_id"); raw != "" {
		teamID := kernel.NewTeamID(raw)
		filter.TeamID = &teamID
	}

	result, err := h.service.Roster(c.Context(), classroomID, actor.UserID, filter, httpx.Pagination(c))
	if err != nil {
		return err
	}

	entries := make([]enrollment.RosterEntry, 0, len(result.Items))
	for i := range result.Items {
		entries = append(entries, result.Items[i].ToRosterEntry())
	}

	return c.JSON(kernel.NewPaginated(entries, result.Page.Number, result.Page.Size, result.Page.Total))
}

func (h *EnrollmentHandlers) InviteStudents(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req enrollment.InviteStudentsRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	results, err := h.service.InviteStudents(c.Context(), classroomID, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"results": results})
}

func (h *EnrollmentHandlers) Update(c *fiber.Ctx) error {
	actor, id, err := actorAndID(c)
	if err != nil {
		return err
	}

	var req enrollment.UpdateEnrollmentRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	e, err := h.service.Update(c.Context(), id, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.JSON(e.ToRosterEntry())
}

func (h *EnrollmentHandlers) Approve(c *fiber.Ctx) error {
	actor, id, err := actorAndID(c)
	if err != nil {
		return err
	}

	e, err := h.service.Approve(c.Context(), id, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(e.ToRosterEntry())
}

func (h *EnrollmentHandlers) Withdraw(c *fiber.Ctx) error {
	actor, id, err := actorAndID(c)
	if err != nil {
		return err
	}

	e, err := h.service.Withdraw(c.Context(), id, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(e.ToRosterEntry())
}

func (h *EnrollmentHandlers) SetTeam(c *fiber.Ctx) error {
	actor, id, err := actorAndID(c)
	if err != nil {
		return err
	}

	var req struct {
		TeamID *string `json:"team_id"`
	}
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	var teamID *kernel.TeamID
	if req.TeamID != nil && *req.TeamID != "" {
		id := kernel.NewTeamID(*req.TeamID)
		teamID = &id
	}

	if err := h.service.SetTeam(c.Context(), id, actor.UserID, teamID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{"message": "Team updated successfully"})
}

func actorAndID(c *fiber.Ctx) (httpx.Actor, kernel.EnrollmentID, error) {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return httpx.Actor{}, "", err
	}
	raw := c.Params("id")
	if raw == "" {
		return httpx.Actor{}, "", httpx.ErrBadRequest("id is required")
	}
	return actor, kernel.NewEnrollmentID(raw), nil
}
