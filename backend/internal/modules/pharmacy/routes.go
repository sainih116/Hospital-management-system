package pharmacy

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	authDomain "hms-backend/internal/modules/auth/domain"
	authRepo "hms-backend/internal/modules/auth/repository"
	prescriptionsRepo "hms-backend/internal/modules/prescriptions/repository"

	"hms-backend/internal/modules/pharmacy/handler"
	"hms-backend/internal/modules/pharmacy/repository"
	"hms-backend/internal/modules/pharmacy/service"
)

// Module exposes the pieces other modules need to reuse.
type Module struct {
	Repository repository.MedicineRepository
}

// Dependencies are the shared, process-wide collaborators the pharmacy
// module needs.
type Dependencies struct {
	DB            *mongo.Database
	Logger        *zap.Logger
	Audit         audit.Recorder
	Authenticate  gin.HandlerFunc
	Prescriptions prescriptionsRepo.PrescriptionRepository
	Users         authRepo.UserRepository
}

// RegisterRoutes wires up the pharmacy module and mounts its routes.
// Reads (catalog browsing, stock movement ledger) require pharmacy.read;
// mutating stock (create/update medicine, receive batch, dispense)
// requires pharmacy.dispense, per the RBAC matrix (Admin and Pharmacist
// only).
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) Module {
	medicineRepo := repository.NewMedicineRepository(deps.DB)
	batchRepo := repository.NewBatchRepository(deps.DB)
	movementRepo := repository.NewStockMovementRepository(deps.DB)

	pharmacySvc := service.NewPharmacyService(service.Deps{
		Medicines:     medicineRepo,
		Batches:       batchRepo,
		Movements:     movementRepo,
		Prescriptions: deps.Prescriptions,
		Users:         deps.Users,
		Audit:         deps.Audit,
		Logger:        deps.Logger,
	})

	h := handler.NewHandler(pharmacySvc)

	group := rg.Group("/pharmacy", deps.Authenticate)

	group.GET("/medicines", middleware.RequirePermission(authDomain.PermPharmacyRead), h.SearchMedicines)
	group.POST("/medicines", middleware.RequirePermission(authDomain.PermPharmacyDispense), h.CreateMedicine)
	group.GET("/medicines/:id", middleware.RequirePermission(authDomain.PermPharmacyRead), h.GetMedicine)
	group.PATCH("/medicines/:id", middleware.RequirePermission(authDomain.PermPharmacyDispense), h.UpdateMedicine)

	group.GET("/medicines/:id/batches", middleware.RequirePermission(authDomain.PermPharmacyRead), h.ListBatches)
	group.POST("/medicines/:id/batches", middleware.RequirePermission(authDomain.PermPharmacyDispense), h.ReceiveBatch)
	group.GET("/batches/expiring", middleware.RequirePermission(authDomain.PermPharmacyRead), h.ListExpiringBatches)

	group.POST("/dispense", middleware.RequirePermission(authDomain.PermPharmacyDispense), h.Dispense)
	group.GET("/stock-movements", middleware.RequirePermission(authDomain.PermPharmacyRead), h.SearchStockMovements)

	return Module{Repository: medicineRepo}
}
