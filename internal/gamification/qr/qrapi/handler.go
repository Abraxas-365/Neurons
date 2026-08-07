package qrapi

import (
	"github.com/Abraxas-356/neurons/internal/gamification/httpx"
	"github.com/Abraxas-356/neurons/internal/gamification/qr"
	"github.com/Abraxas-356/neurons/internal/gamification/qr/qrsrv"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/iam/scopes"
	"github.com/gofiber/fiber/v2"
)

type QRHandlers struct {
	service *qrsrv.QRService
}

func NewQRHandlers(service *qrsrv.QRService) *QRHandlers {
	return &QRHandlers{service: service}
}

func (h *QRHandlers) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	// Teacher side
	cr := router.Group("/classrooms/:classroomId/qr")
	cr.Post("/scan", mw.RequireScope(scopes.ScopeNeuronsGrant), h.Scan)
	cr.Post("/grants", mw.RequireScope(scopes.ScopeNeuronsGrant), h.IssueGrant)

	// Student side
	me := router.Group("/me/classrooms/:classroomId/qr", mw.RequireScope(scopes.ScopeStudentSelf))
	me.Post("/", h.IssueStudent)
	me.Post("/claim", h.Claim)
}

// IssueStudent returns the student's rotating identity code (HU-053).
func (h *QRHandlers) IssueStudent(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	issued, err := h.service.IssueStudent(c.Context(), classroomID, actor.UserID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(issued)
}

// Scan resolves a scanned student code into the grant screen's context.
func (h *QRHandlers) Scan(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req qr.ScanRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	student, err := h.service.Scan(c.Context(), classroomID, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.JSON(student)
}

// IssueGrant prepares a scannable grant for the class (HU-050).
func (h *QRHandlers) IssueGrant(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req qr.IssueGrantRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	issued, err := h.service.IssueGrant(c.Context(), classroomID, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(issued)
}

// Claim redeems a teacher-displayed grant code.
func (h *QRHandlers) Claim(c *fiber.Ctx) error {
	actor, err := httpx.CurrentActor(c)
	if err != nil {
		return err
	}
	classroomID, err := httpx.ClassroomID(c, "classroomId")
	if err != nil {
		return err
	}

	var req qr.ClaimRequest
	if err := httpx.BodyParser(c, &req); err != nil {
		return err
	}

	result, err := h.service.Claim(c.Context(), classroomID, actor.UserID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}
