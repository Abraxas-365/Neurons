package classroomapi

import (
	"github.com/Abraxas-356/neurons/internal/gamification/classroom"
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/httpx"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/iam/scopes"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/gofiber/fiber/v2"
)

type ClassroomHandlers struct {
	service *classroomsrv.ClassroomService
}

func NewClassroomHandlers(service *classroomsrv.ClassroomService) *ClassroomHandlers {
	return &ClassroomHandlers{service: service}
}

func (h *ClassroomHandlers) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	g := router.Group("/classrooms")

	g.Post("/", mw.RequireScope(scopes.ScopeClassroomsWrite), h.Create)
	g.Get("/", mw.RequireScope(scopes.ScopeClassroomsRead), h.ListMine)

	// Any authenticated user may preview a classroom from its invite code
	// before deciding to join (HU-022).
	g.Get("/by-code/:code", h.GetByInviteCode)

	g.Get("/:id", mw.RequireScope(scopes.ScopeClassroomsRead), h.GetByID)
	g.Put("/:id", mw.RequireScope(scopes.ScopeClassroomsWrite), h.Update)
	g.Delete("/:id", mw.RequireScope(scopes.ScopeClassroomsDelete), h.Delete)

	// Vault top-ups live in the ledger module: every neuron entering the vault
	// must leave an audit entry (RN-09), which a plain classroom update cannot do.

	g.Get("/:id/teachers", mw.RequireScope(scopes.ScopeClassroomsRead), h.ListTeachers)
	g.Post("/:id/teachers", mw.RequireScope(scopes.ScopeClassroomsWrite), h.AddTeacher)
	g.Delete("/:id/teachers/:userId", mw.RequireScope(scopes.ScopeClassroomsWrite), h.RemoveTeacher)
}

func (h *ClassroomHandlers) Create(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}

	var req classroom.CreateClassroomRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	entity, err := h.service.Create(c.Context(), actor.TenantID, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(entity.ToResponse())
}

func (h *ClassroomHandlers) ListMine(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}

	result, err := h.service.ListForTeacher(c.Context(), actor.TenantID, actor.UserID, httpx.Pagination(c))
	if err != nil {
		return err
	}

	responses := make([]classroom.ClassroomResponse, 0, len(result.Items))
	for i := range result.Items {
		responses = append(responses, result.Items[i].ToResponse())
	}

	return c.JSON(kernel.NewPaginated(responses, result.Page.Number, result.Page.Size, result.Page.Total))
}

func (h *ClassroomHandlers) GetByID(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id, err := httpx.ClassroomID(c, "id")
	if err != nil {
		return err
	}

	entity, _, err := h.service.RequireTeacher(c.Context(), id, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(entity.ToResponse())
}

// GetByInviteCode returns the student-facing preview of a classroom.
func (h *ClassroomHandlers) GetByInviteCode(c *fiber.Ctx) error {
	entity, err := h.service.GetByInviteCode(c.Context(), c.Params("code"))
	if err != nil {
		return err
	}
	return c.JSON(entity.ToStudentView())
}

func (h *ClassroomHandlers) Update(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id, err := httpx.ClassroomID(c, "id")
	if err != nil {
		return err
	}

	var req classroom.UpdateClassroomRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	entity, err := h.service.Update(c.Context(), id, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.JSON(entity.ToResponse())
}

func (h *ClassroomHandlers) ListTeachers(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id, err := httpx.ClassroomID(c, "id")
	if err != nil {
		return err
	}

	teachers, err := h.service.ListTeachers(c.Context(), id, actor.UserID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"items": teachers})
}

func (h *ClassroomHandlers) AddTeacher(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id, err := httpx.ClassroomID(c, "id")
	if err != nil {
		return err
	}

	var req classroom.AddTeacherRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	t, err := h.service.AddTeacher(c.Context(), id, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(t)
}

func (h *ClassroomHandlers) RemoveTeacher(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id, err := httpx.ClassroomID(c, "id")
	if err != nil {
		return err
	}

	target := kernel.NewUserID(c.Params("userId"))
	if err := h.service.RemoveTeacher(c.Context(), id, actor.UserID, target); err != nil {
		return err
	}

	return c.JSON(fiber.Map{"message": "Teacher removed successfully"})
}

func (h *ClassroomHandlers) Delete(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	id, err := httpx.ClassroomID(c, "id")
	if err != nil {
		return err
	}

	if err := h.service.Delete(c.Context(), id, actor.UserID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{"message": "Classroom deleted successfully"})
}
