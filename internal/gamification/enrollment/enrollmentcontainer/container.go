package enrollmentcontainer

import (
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentapi"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentinfra"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentsrv"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/logx"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

type Deps struct {
	DB         *sqlx.DB
	Classrooms *classroomsrv.ClassroomService
	Users      enrollmentsrv.UserLookup
}

type Container struct {
	EnrollmentRepository enrollment.Repository
	EnrollmentService    *enrollmentsrv.EnrollmentService
	EnrollmentHandlers   *enrollmentapi.EnrollmentHandlers
}

func New(deps Deps) *Container {
	logx.Info("🔧 Initializing Enrollment container...")

	repo := enrollmentinfra.NewPostgresEnrollmentRepository(deps.DB)
	svc := enrollmentsrv.NewEnrollmentService(repo, deps.Classrooms, deps.Users)
	handlers := enrollmentapi.NewEnrollmentHandlers(svc)

	logx.Info("✅ Enrollment container initialized")

	return &Container{
		EnrollmentRepository: repo,
		EnrollmentService:    svc,
		EnrollmentHandlers:   handlers,
	}
}

func (c *Container) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	c.EnrollmentHandlers.RegisterRoutes(router, mw)
}
