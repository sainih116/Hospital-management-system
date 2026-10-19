package prescriptions

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	authDomain "hms-backend/internal/modules/auth/domain"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	medicalRecordsRepo "hms-backend/internal/modules/medicalrecords/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"

	"hms-backend/internal/modules/prescriptions/handler"
	"hms-backend/internal/modules/prescriptions/repository"
	"hms-backend/internal/modules/prescriptions/service"
)

// Module exposes the pieces other modules need to reuse — notably the
// repository, since Pharmacy looks up and updates prescription status
// when dispensing without standing up a second connection to the same
// collection.
type Module struct {
	Repository repository.PrescriptionRepository
}

// Dependencies are the shared, process-wide collaborators the
// prescriptions module needs.
type Dependencies struct {
	DB            *mongo.Database
	Logger        *zap.Logger
	Audit         audit.Recorder
	Authenticate  gin.HandlerFunc
	Patients      patientsRepo.PatientRepository
	Doctors       doctorsRepo.DoctorRepository
	MedicalRecord medicalRecordsRepo.MedicalRecordRepository
}

// RegisterRoutes wires up the prescriptions module and mounts its routes.
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) Module {
	prescriptionRepo := repository.NewPrescriptionRepository(deps.DB)

	prescriptionSvc := service.NewPrescriptionService(service.Deps{
		Prescriptions: prescriptionRepo,
		Patients:      deps.Patients,
		Doctors:       deps.Doctors,
		MedicalRecord: deps.MedicalRecord,
		Audit:         deps.Audit,
		Logger:        deps.Logger,
	})

	h := handler.NewHandler(prescriptionSvc)

	group := rg.Group("/prescriptions", deps.Authenticate)
	group.POST("", middleware.RequirePermission(authDomain.PermPrescriptionWrite), h.Create)
	group.GET("", middleware.RequirePermission(authDomain.PermPrescriptionRead), h.Search)
	group.GET("/:id", middleware.RequirePermission(authDomain.PermPrescriptionRead), h.GetByID)
	group.PATCH("/:id/status", middleware.RequirePermission(authDomain.PermPrescriptionWrite), h.ChangeStatus)

	return Module{Repository: prescriptionRepo}
}
