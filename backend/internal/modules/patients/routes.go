package patients

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/auth/domain"
	"hms-backend/internal/modules/patients/handler"
	"hms-backend/internal/modules/patients/repository"
	"hms-backend/internal/modules/patients/service"
	"hms-backend/internal/pkg/sequence"
)

// Module exposes the pieces other modules need to reuse — notably the
// repository, since Appointments validates/snapshots patient data without
// standing up a second connection to the same collection.
type Module struct {
	Repository repository.PatientRepository
}

// Dependencies are the shared, process-wide collaborators the patients
// module needs, supplied by the composition root (internal/server).
type Dependencies struct {
	DB           *mongo.Database
	Logger       *zap.Logger
	Audit        audit.Recorder
	Authenticate gin.HandlerFunc
}

// RegisterRoutes wires up the patients module's repository, service and
// handler, and mounts its routes under the given group.
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) Module {
	patientRepo := repository.NewPatientRepository(deps.DB)
	sequences := sequence.NewStore(deps.DB)

	patientSvc := service.NewPatientService(service.Deps{
		Patients:  patientRepo,
		Sequences: sequences,
		Audit:     deps.Audit,
		Logger:    deps.Logger,
	})

	h := handler.NewHandler(patientSvc)

	group := rg.Group("/patients", deps.Authenticate)
	group.POST("", middleware.RequirePermission(domain.PermPatientCreate), h.Create)
	group.GET("", middleware.RequirePermission(domain.PermPatientRead), h.Search)
	group.GET("/:id", middleware.RequirePermission(domain.PermPatientRead), h.GetByID)
	group.PATCH("/:id", middleware.RequirePermission(domain.PermPatientUpdate), h.Update)
	group.DELETE("/:id", middleware.RequirePermission(domain.PermPatientDelete), h.Delete)

	return Module{Repository: patientRepo}
}
