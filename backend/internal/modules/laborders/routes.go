package laborders

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	authDomain "hms-backend/internal/modules/auth/domain"
	authRepo "hms-backend/internal/modules/auth/repository"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	labTestsRepo "hms-backend/internal/modules/labtests/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"
	"hms-backend/internal/pkg/sequence"

	"hms-backend/internal/modules/laborders/handler"
	"hms-backend/internal/modules/laborders/repository"
	"hms-backend/internal/modules/laborders/service"
)

// Dependencies are the shared, process-wide collaborators the lab orders
// module needs.
type Dependencies struct {
	DB           *mongo.Database
	Logger       *zap.Logger
	Audit        audit.Recorder
	Authenticate gin.HandlerFunc
	Patients     patientsRepo.PatientRepository
	Doctors      doctorsRepo.DoctorRepository
	Tests        labTestsRepo.LabTestRepository
	Users        authRepo.UserRepository
}

// RegisterRoutes wires up the lab orders module and mounts its routes.
// Note the deliberate split: /lab-orders/* uses lab.order.read/write (the
// blueprint's coarse-grained Doctor+LabTech grant), while
// /lab-results/{id}/verify uses the narrower lab.result.verify permission
// (Admin+LabTech only — the REST table explicitly excludes Doctor here,
// unlike the general Lab Orders grant).
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) {
	orderRepo := repository.NewLabOrderRepository(deps.DB)
	resultRepo := repository.NewLabResultRepository(deps.DB)
	sequences := sequence.NewStore(deps.DB)

	orderSvc := service.NewLabOrderService(service.Deps{
		Orders: orderRepo, Results: resultRepo, Tests: deps.Tests,
		Patients: deps.Patients, Doctors: deps.Doctors, Users: deps.Users,
		Sequences: sequences, Audit: deps.Audit, Logger: deps.Logger,
	})

	h := handler.NewHandler(orderSvc)

	orders := rg.Group("/lab-orders", deps.Authenticate)
	orders.POST("", middleware.RequirePermission(authDomain.PermLabOrderWrite), h.Create)
	orders.GET("", middleware.RequirePermission(authDomain.PermLabOrderRead), h.Search)
	orders.GET("/:id", middleware.RequirePermission(authDomain.PermLabOrderRead), h.GetByID)
	orders.GET("/:id/report", middleware.RequirePermission(authDomain.PermLabOrderRead), h.Report)
	orders.POST("/:id/cancel", middleware.RequirePermission(authDomain.PermLabOrderWrite), h.Cancel)
	orders.POST("/:id/result", middleware.RequirePermission(authDomain.PermLabOrderWrite), h.EnterResult)

	results := rg.Group("/lab-results", deps.Authenticate)
	results.PATCH("/:id/verify", middleware.RequirePermission(authDomain.PermLabResultVerify), h.VerifyResult)
}
