package medalapi

import (
	"github.com/Abraxas-356/neurons/internal/gamification/httpx"
	"github.com/Abraxas-356/neurons/internal/gamification/medal"
	"github.com/Abraxas-356/neurons/internal/gamification/medal/medalsrv"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/iam/scopes"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/gofiber/fiber/v2"
)

type MedalHandlers struct {
	service *medalsrv.MedalService
}

func NewMedalHandlers(service *medalsrv.MedalService) *MedalHandlers {
	return &MedalHandlers{service: service}
}

func (h *MedalHandlers) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	cr := router.Group("/classrooms/:classroomId/medals")
	cr.Get("/", mw.RequireScope(scopes.ScopeCatalogRead), h.List)
	cr.Post("/", mw.RequireScope(scopes.ScopeCatalogWrite), h.Create)
	cr.Get("/awards", mw.RequireScope(scopes.ScopeCatalogRead), h.ClassroomAwards)

	g := router.Group("/medals")
	g.Put("/:id", mw.RequireScope(scopes.ScopeCatalogWrite), h.Update)
	g.Delete("/:id", mw.RequireScope(scopes.ScopeCatalogWrite), h.Delete)
	g.Post("/:id/awards", mw.RequireScope(scopes.ScopeMedalsAward), h.Award)

	router.Delete("/medal-awards/:awardId", mw.RequireScope(scopes.ScopeMedalsAward), h.Revoke)

	// Student medal wall (HU-074).
	router.Get("/me/classrooms/:classroomId/medals", mw.RequireScope(scopes.ScopeStudentSelf), h.MyMedals)
}

func (h *MedalHandlers) Create(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req medal.CreateMedalRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	e, err := h.service.Create(c.Context(), classroomID, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(e)
}

func (h *MedalHandlers) List(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	items, err := h.service.List(c.Context(), classroomID, actor.UserID, c.QueryBool("active_only", false))
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"items": items})
}

func (h *MedalHandlers) Update(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id := kernel.NewMedalID(c.Params("id"))

	var req medal.UpdateMedalRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	e, err := h.service.Update(c.Context(), id, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.JSON(e)
}

func (h *MedalHandlers) Delete(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id := kernel.NewMedalID(c.Params("id"))

	if err := h.service.Delete(c.Context(), id, actor.UserID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{"message": "Medal deleted successfully"})
}

func (h *MedalHandlers) Award(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id := kernel.NewMedalID(c.Params("id"))

	var req medal.AwardRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	awards, err := h.service.Award(c.Context(), id, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"awarded": len(awards),
		"items":   awards,
	})
}

func (h *MedalHandlers) Revoke(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}

	if err := h.service.Revoke(c.Context(), c.Params("awardId"), actor.UserID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{"message": "Medal award revoked successfully"})
}

func (h *MedalHandlers) ClassroomAwards(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	items, err := h.service.ClassroomAwards(c.Context(), classroomID, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"items": items})
}

func (h *MedalHandlers) MyMedals(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	items, err := h.service.MyMedals(c.Context(), classroomID, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"items": items})
}
