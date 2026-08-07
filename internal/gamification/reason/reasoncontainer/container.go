package reasoncontainer

import (
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/reason"
	"github.com/Abraxas-356/neurons/internal/gamification/reason/reasonapi"
	"github.com/Abraxas-356/neurons/internal/gamification/reason/reasoninfra"
	"github.com/Abraxas-356/neurons/internal/gamification/reason/reasonsrv"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/logx"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

type Deps struct {
	DB         *sqlx.DB
	Classrooms *classroomsrv.ClassroomService
}

type Container struct {
	ReasonRepository reason.Repository
	ReasonService    *reasonsrv.ReasonService
	ReasonHandlers   *reasonapi.ReasonHandlers
}

func New(deps Deps) *Container {
	logx.Info("🔧 Initializing Reason container...")

	repo := reasoninfra.NewPostgresReasonRepository(deps.DB)
	svc := reasonsrv.NewReasonService(repo, deps.Classrooms)
	handlers := reasonapi.NewReasonHandlers(svc)

	logx.Info("✅ Reason container initialized")

	return &Container{
		ReasonRepository: repo,
		ReasonService:    svc,
		ReasonHandlers:   handlers,
	}
}

func (c *Container) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	c.ReasonHandlers.RegisterRoutes(router, mw)
}
