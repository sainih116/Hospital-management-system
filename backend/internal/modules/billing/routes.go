package billing

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	authDomain "hms-backend/internal/modules/auth/domain"
	authRepo "hms-backend/internal/modules/auth/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"
	"hms-backend/internal/pkg/sequence"

	"hms-backend/internal/modules/billing/handler"
	"hms-backend/internal/modules/billing/repository"
	"hms-backend/internal/modules/billing/service"
)

// Dependencies are the shared, process-wide collaborators the billing
// module needs.
type Dependencies struct {
	DB           *mongo.Database
	Logger       *zap.Logger
	Audit        audit.Recorder
	Authenticate gin.HandlerFunc
	Patients     patientsRepo.PatientRepository
	Users        authRepo.UserRepository
}

// RegisterRoutes wires up the billing module and mounts its routes.
// Reads use billing.read (Admin + Receptionist, per the seeded RBAC
// matrix); every mutation (create, record payment, cancel) uses the
// narrower billing.manage (Admin only) — the same "coarse read grant,
// narrow write grant" split Lab Orders uses for lab.result.verify.
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) {
	billRepo := repository.NewBillRepository(deps.DB)
	paymentRepo := repository.NewPaymentRepository(deps.DB)
	sequences := sequence.NewStore(deps.DB)

	billingSvc := service.NewBillingService(service.Deps{
		Bills: billRepo, Payments: paymentRepo, Patients: deps.Patients,
		Users: deps.Users, Sequences: sequences, Audit: deps.Audit, Logger: deps.Logger,
	})

	h := handler.NewHandler(billingSvc)

	bills := rg.Group("/billing/bills", deps.Authenticate)
	bills.POST("", middleware.RequirePermission(authDomain.PermBillingManage), h.Create)
	bills.GET("", middleware.RequirePermission(authDomain.PermBillingRead), h.Search)
	bills.GET("/:id", middleware.RequirePermission(authDomain.PermBillingRead), h.GetByID)
	bills.POST("/:id/payments", middleware.RequirePermission(authDomain.PermBillingManage), h.RecordPayment)
	bills.POST("/:id/cancel", middleware.RequirePermission(authDomain.PermBillingManage), h.Cancel)
}
