package teamcontainer

import (
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/team"
	"github.com/Abraxas-356/neurons/internal/gamification/team/teamapi"
	"github.com/Abraxas-356/neurons/internal/gamification/team/teaminfra"
	"github.com/Abraxas-356/neurons/internal/gamification/team/teamsrv"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/logx"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

type Deps struct {
	DB          *sqlx.DB
	Classrooms  *classroomsrv.ClassroomService
	Enrollments *enrollmentsrv.EnrollmentService
	Roster      enrollment.Repository
}

type Container struct {
	TeamRepository team.Repository
	TeamService    *teamsrv.TeamService
	TeamHandlers   *teamapi.TeamHandlers
}

func New(deps Deps) *Container {
	logx.Info("🔧 Initializing Team container...")

	repo := teaminfra.NewPostgresTeamRepository(deps.DB)
	svc := teamsrv.NewTeamService(repo, deps.Classrooms, deps.Enrollments, deps.Roster)
	handlers := teamapi.NewTeamHandlers(svc)

	logx.Info("✅ Team container initialized")

	return &Container{
		TeamRepository: repo,
		TeamService:    svc,
		TeamHandlers:   handlers,
	}
}

func (c *Container) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	c.TeamHandlers.RegisterRoutes(router, mw)
}
