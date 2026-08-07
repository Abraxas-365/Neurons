package teamapi

import (
	"github.com/Abraxas-356/neurons/internal/gamification/httpx"
	"github.com/Abraxas-356/neurons/internal/gamification/team"
	"github.com/Abraxas-356/neurons/internal/gamification/team/teamsrv"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/iam/scopes"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/gofiber/fiber/v2"
)

type TeamHandlers struct {
	service *teamsrv.TeamService
}

func NewTeamHandlers(service *teamsrv.TeamService) *TeamHandlers {
	return &TeamHandlers{service: service}
}

func (h *TeamHandlers) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	cr := router.Group("/classrooms/:classroomId/teams")
	cr.Get("/", mw.RequireScope(scopes.ScopeTeamsRead), h.List)
	cr.Post("/", mw.RequireScope(scopes.ScopeTeamsWrite), h.Create)
	cr.Post("/randomize", mw.RequireScope(scopes.ScopeTeamsWrite), h.Randomize)

	g := router.Group("/teams")
	g.Get("/:id", mw.RequireScope(scopes.ScopeTeamsRead), h.Detail)
	g.Put("/:id", mw.RequireScope(scopes.ScopeTeamsWrite), h.Update)
	g.Delete("/:id", mw.RequireScope(scopes.ScopeTeamsWrite), h.Delete)
	g.Put("/:id/members", mw.RequireScope(scopes.ScopeTeamsWrite), h.SetMembers)
	g.Put("/:id/coordinator", mw.RequireScope(scopes.ScopeTeamsWrite), h.SetCoordinator)

	// Students may view their own team roster.
	router.Get("/me/teams/:id", mw.RequireScope(scopes.ScopeStudentSelf), h.MyTeam)
}

func (h *TeamHandlers) Create(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req team.CreateTeamRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	t, err := h.service.Create(c.Context(), classroomID, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(t.ToResponse())
}

func (h *TeamHandlers) List(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	teams, err := h.service.ListByClassroom(c.Context(), classroomID, actor.UserID)
	if err != nil {
		return err
	}

	items := make([]team.TeamResponse, 0, len(teams))
	for i := range teams {
		items = append(items, teams[i].ToResponse())
	}

	return c.JSON(fiber.Map{"items": items})
}

func (h *TeamHandlers) Detail(c *fiber.Ctx) error {
	actor, id, err := actorAndID(c)
	if err != nil {
		return err
	}

	detail, err := h.service.Detail(c.Context(), id, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(detail)
}

func (h *TeamHandlers) MyTeam(c *fiber.Ctx) error {
	actor, id, err := actorAndID(c)
	if err != nil {
		return err
	}

	detail, err := h.service.MembersForStudent(c.Context(), id, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(detail)
}

func (h *TeamHandlers) Update(c *fiber.Ctx) error {
	actor, id, err := actorAndID(c)
	if err != nil {
		return err
	}

	var req team.UpdateTeamRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	t, err := h.service.Update(c.Context(), id, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.JSON(t.ToResponse())
}

func (h *TeamHandlers) SetMembers(c *fiber.Ctx) error {
	actor, id, err := actorAndID(c)
	if err != nil {
		return err
	}

	var req team.SetMembersRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	members, err := h.service.SetMembers(c.Context(), id, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"members": members})
}

func (h *TeamHandlers) SetCoordinator(c *fiber.Ctx) error {
	actor, id, err := actorAndID(c)
	if err != nil {
		return err
	}

	var req struct {
		EnrollmentID string `json:"enrollment_id"`
	}
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}
	if req.EnrollmentID == "" {
		return httpx.ErrBadRequest("enrollment_id is required")
	}

	if err := h.service.SetCoordinator(c.Context(), id, actor.UserID, kernel.NewEnrollmentID(req.EnrollmentID)); err != nil {
		return err
	}

	return c.JSON(fiber.Map{"message": "Coordinator updated successfully"})
}

func (h *TeamHandlers) Randomize(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req team.RandomizeRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	result, err := h.service.Randomize(c.Context(), classroomID, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func (h *TeamHandlers) Delete(c *fiber.Ctx) error {
	actor, id, err := actorAndID(c)
	if err != nil {
		return err
	}

	if err := h.service.Delete(c.Context(), id, actor.UserID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{"message": "Team deleted successfully"})
}

func actorAndID(c *fiber.Ctx) (httpx.Actor, kernel.TeamID, error) {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return httpx.Actor{}, "", err
	}
	raw := c.Params("id")
	if raw == "" {
		return httpx.Actor{}, "", httpx.ErrBadRequest("id is required")
	}
	return actor, kernel.NewTeamID(raw), nil
}
