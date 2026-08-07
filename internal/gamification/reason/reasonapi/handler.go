package reasonapi

import (
	"github.com/Abraxas-356/neurons/internal/gamification/httpx"
	"github.com/Abraxas-356/neurons/internal/gamification/reason"
	"github.com/Abraxas-356/neurons/internal/gamification/reason/reasonsrv"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/iam/scopes"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/gofiber/fiber/v2"
)

type ReasonHandlers struct {
	service *reasonsrv.ReasonService
}

func NewReasonHandlers(service *reasonsrv.ReasonService) *ReasonHandlers {
	return &ReasonHandlers{service: service}
}

func (h *ReasonHandlers) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	cr := router.Group("/classrooms/:classroomId/reasons")
	cr.Get("/", mw.RequireScope(scopes.ScopeCatalogRead), h.List)
	cr.Post("/", mw.RequireScope(scopes.ScopeCatalogWrite), h.Create)

	g := router.Group("/reasons", mw.RequireScope(scopes.ScopeCatalogWrite))
	g.Put("/:id", h.Update)
	g.Delete("/:id", h.Delete)
}

func (h *ReasonHandlers) Create(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req reason.CreateReasonRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	e, err := h.service.Create(c.Context(), classroomID, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(e)
}

func (h *ReasonHandlers) List(c *fiber.Ctx) error {
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

func (h *ReasonHandlers) Update(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id := kernel.NewReasonID(c.Params("id"))

	var req reason.UpdateReasonRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	e, err := h.service.Update(c.Context(), id, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.JSON(e)
}

func (h *ReasonHandlers) Delete(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id := kernel.NewReasonID(c.Params("id"))

	if err := h.service.Delete(c.Context(), id, actor.UserID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{"message": "Reason deleted successfully"})
}
