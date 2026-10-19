package medicalrecords

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	authDomain "hms-backend/internal/modules/auth/domain"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"

	"hms-backend/internal/modules/medicalrecords/handler"
	"hms-backend/internal/modules/medicalrecords/repository"
	"hms-backend/internal/modules/medicalrecords/service"
)

// Module exposes the pieces other modules need to reuse — notably the
// repository, since Prescriptions validates an optional medicalRecordId
// link without standing up a second connection to the same collection.
type Module struct {
	Repository repository.MedicalRecordRepository
}

// Dependencies are the shared, process-wide collaborators the medical
// records module needs.
type Dependencies struct {
	DB           *mongo.Database
	Logger       *zap.Logger
	Audit        audit.Recorder
	Authenticate gin.HandlerFunc
	Patients     patientsRepo.PatientRepository
	Doctors      doctorsRepo.DoctorRepository
}

// RegisterRoutes wires up the medical records module and mounts its routes.
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) Module {
	recordRepo := repository.NewMedicalRecordRepository(deps.DB)

	recordSvc := service.NewMedicalRecordService(service.Deps{
		Records:  recordRepo,
		Patients: deps.Patients,
		Doctors:  deps.Doctors,
		Audit:    deps.Audit,
		Logger:   deps.Logger,
	})

	h := handler.NewHandler(recordSvc)

	group := rg.Group("/medical-records", deps.Authenticate)
	group.POST("", middleware.RequirePermission(authDomain.PermMedicalRecordWrite), h.Create)
	group.GET("", middleware.RequirePermission(authDomain.PermMedicalRecordRead), h.Search)
	group.GET("/:id", middleware.RequirePermission(authDomain.PermMedicalRecordRead), h.GetByID)
	group.PATCH("/:id", middleware.RequirePermission(authDomain.PermMedicalRecordWrite), h.Update)

	return Module{Repository: recordRepo}
}
