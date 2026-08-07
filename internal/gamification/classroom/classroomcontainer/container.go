package classroomcontainer

import (
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomapi"
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroominfra"
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/logx"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

// Deps holds the external dependencies this module requires.
type Deps struct {
	DB    *sqlx.DB
	Users classroomsrv.UserLookup
}

// Container exposes only what other modules or cmd/ actually need.
type Container struct {
	ClassroomService  *classroomsrv.ClassroomService
	ClassroomHandlers *classroomapi.ClassroomHandlers
}

// New constructs the entire Classroom dependency graph.
func New(deps Deps) *Container {
	logx.Info("🔧 Initializing Classroom container...")

	repo := classroominfra.NewPostgresClassroomRepository(deps.DB)
	svc := classroomsrv.NewClassroomService(repo, deps.Users)
	handlers := classroomapi.NewClassroomHandlers(svc)

	logx.Info("✅ Classroom container initialized")

	return &Container{
		ClassroomService:  svc,
		ClassroomHandlers: handlers,
	}
}

// RegisterRoutes registers all Classroom HTTP routes on the given router.
func (c *Container) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	c.ClassroomHandlers.RegisterRoutes(router, mw)
}
