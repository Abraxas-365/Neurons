package medalcontainer

import (
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/medal"
	"github.com/Abraxas-356/neurons/internal/gamification/medal/medalapi"
	"github.com/Abraxas-356/neurons/internal/gamification/medal/medalinfra"
	"github.com/Abraxas-356/neurons/internal/gamification/medal/medalsrv"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/logx"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

type Deps struct {
	DB          *sqlx.DB
	Classrooms  *classroomsrv.ClassroomService
	Enrollments *enrollmentsrv.EnrollmentService
}

type Container struct {
	MedalRepository medal.Repository
	MedalService    *medalsrv.MedalService
	MedalHandlers   *medalapi.MedalHandlers
}

func New(deps Deps) *Container {
	logx.Info("🔧 Initializing Medal container...")

	repo := medalinfra.NewPostgresMedalRepository(deps.DB)
	svc := medalsrv.NewMedalService(repo, deps.Classrooms, deps.Enrollments)
	handlers := medalapi.NewMedalHandlers(svc)

	logx.Info("✅ Medal container initialized")

	return &Container{
		MedalRepository: repo,
		MedalService:    svc,
		MedalHandlers:   handlers,
	}
}

func (c *Container) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	c.MedalHandlers.RegisterRoutes(router, mw)
}
