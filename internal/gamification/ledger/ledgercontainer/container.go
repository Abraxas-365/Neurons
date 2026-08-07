package ledgercontainer

import (
	"github.com/Abraxas-356/neurons/internal/gamification/benefit/benefitsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/ledger"
	"github.com/Abraxas-356/neurons/internal/gamification/ledger/ledgerapi"
	"github.com/Abraxas-356/neurons/internal/gamification/ledger/ledgerinfra"
	"github.com/Abraxas-356/neurons/internal/gamification/ledger/ledgersrv"
	"github.com/Abraxas-356/neurons/internal/gamification/reason/reasonsrv"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/logx"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

type Deps struct {
	DB          *sqlx.DB
	Classrooms  *classroomsrv.ClassroomService
	Enrollments *enrollmentsrv.EnrollmentService
	Reasons     *reasonsrv.ReasonService
	Benefits    *benefitsrv.BenefitService
}

type Container struct {
	LedgerRepository ledger.Repository
	LedgerService    *ledgersrv.LedgerService
	LedgerHandlers   *ledgerapi.LedgerHandlers
}

func New(deps Deps) *Container {
	logx.Info("🔧 Initializing Ledger container...")

	repo := ledgerinfra.NewPostgresLedgerRepository(deps.DB)
	svc := ledgersrv.NewLedgerService(repo, deps.Classrooms, deps.Enrollments, deps.Reasons, deps.Benefits)
	handlers := ledgerapi.NewLedgerHandlers(svc)

	logx.Info("✅ Ledger container initialized")

	return &Container{
		LedgerRepository: repo,
		LedgerService:    svc,
		LedgerHandlers:   handlers,
	}
}

func (c *Container) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	c.LedgerHandlers.RegisterRoutes(router, mw)
}
