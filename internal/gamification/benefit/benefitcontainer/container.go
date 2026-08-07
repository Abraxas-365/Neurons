package benefitcontainer

import (
	"github.com/Abraxas-356/neurons/internal/gamification/benefit"
	"github.com/Abraxas-356/neurons/internal/gamification/benefit/benefitapi"
	"github.com/Abraxas-356/neurons/internal/gamification/benefit/benefitinfra"
	"github.com/Abraxas-356/neurons/internal/gamification/benefit/benefitsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentsrv"
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
	BenefitRepository benefit.Repository
	BenefitService    *benefitsrv.BenefitService
	BenefitHandlers   *benefitapi.BenefitHandlers
}

func New(deps Deps) *Container {
	logx.Info("🔧 Initializing Benefit container...")

	repo := benefitinfra.NewPostgresBenefitRepository(deps.DB)
	svc := benefitsrv.NewBenefitService(repo, deps.Classrooms, deps.Enrollments)
	handlers := benefitapi.NewBenefitHandlers(svc)

	logx.Info("✅ Benefit container initialized")

	return &Container{
		BenefitRepository: repo,
		BenefitService:    svc,
		BenefitHandlers:   handlers,
	}
}

func (c *Container) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	c.BenefitHandlers.RegisterRoutes(router, mw)
}
