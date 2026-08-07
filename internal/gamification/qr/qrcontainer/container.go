package qrcontainer

import (
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/ledger/ledgersrv"
	"github.com/Abraxas-356/neurons/internal/gamification/qr"
	"github.com/Abraxas-356/neurons/internal/gamification/qr/qrapi"
	"github.com/Abraxas-356/neurons/internal/gamification/qr/qrinfra"
	"github.com/Abraxas-356/neurons/internal/gamification/qr/qrsrv"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/logx"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type Deps struct {
	Redis       *redis.Client
	Classrooms  *classroomsrv.ClassroomService
	Enrollments *enrollmentsrv.EnrollmentService
	Ledger      *ledgersrv.LedgerService
}

type Container struct {
	QRStore    qr.Store
	QRService  *qrsrv.QRService
	QRHandlers *qrapi.QRHandlers
}

func New(deps Deps) *Container {
	logx.Info("🔧 Initializing QR container...")

	store := qrinfra.NewRedisQRStore(deps.Redis)
	svc := qrsrv.NewQRService(store, deps.Classrooms, deps.Enrollments, deps.Ledger)
	handlers := qrapi.NewQRHandlers(svc)

	logx.Info("✅ QR container initialized")

	return &Container{
		QRStore:    store,
		QRService:  svc,
		QRHandlers: handlers,
	}
}

func (c *Container) RegisterRoutes(router fiber.Router, mw *auth.UnifiedAuthMiddleware) {
	c.QRHandlers.RegisterRoutes(router, mw)
}
