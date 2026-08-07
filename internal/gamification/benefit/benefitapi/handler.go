package benefitapi

import (
	"github.com/Abraxas-356/neurons/internal/gamification/benefit"
	"github.com/Abraxas-356/neurons/internal/gamification/benefit/benefitsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/httpx"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/iam/scopes"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/gofiber/fiber/v2"
)

type BenefitHandlers struct {
	service *benefitsrv.BenefitService
}

func NewBenefitHandlers(service *benefitsrv.BenefitService) *BenefitHandlers {
	return &BenefitHandlers{service: service}
}

func (h *BenefitHandlers) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	cr := router.Group("/classrooms/:classroomId/benefits")
	cr.Get("/", mw.RequireScope(scopes.ScopeCatalogRead), h.List)
	cr.Post("/", mw.RequireScope(scopes.ScopeCatalogWrite), h.Create)

	// Student-facing catalog (HU-061).
	router.Get(
		"/me/classrooms/:classroomId/benefits",
		mw.RequireScope(scopes.ScopeStudentSelf),
		h.ListForStudent,
	)

	g := router.Group("/benefits", mw.RequireScope(scopes.ScopeCatalogWrite))
	g.Put("/:id", h.Update)
	g.Delete("/:id", h.Delete)
}

func (h *BenefitHandlers) Create(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req benefit.CreateBenefitRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	e, err := h.service.Create(c.Context(), classroomID, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(e)
}

func (h *BenefitHandlers) List(c *fiber.Ctx) error {
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

func (h *BenefitHandlers) ListForStudent(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	items, err := h.service.ListForStudent(c.Context(), classroomID, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"items": items})
}

func (h *BenefitHandlers) Update(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id := kernel.NewBenefitID(c.Params("id"))

	var req benefit.UpdateBenefitRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	e, err := h.service.Update(c.Context(), id, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.JSON(e)
}

func (h *BenefitHandlers) Delete(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id := kernel.NewBenefitID(c.Params("id"))

	if err := h.service.Delete(c.Context(), id, actor.UserID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{"message": "Benefit deleted successfully"})
}
