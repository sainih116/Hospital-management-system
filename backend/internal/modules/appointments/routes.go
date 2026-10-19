package appointments

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	authDomain "hms-backend/internal/modules/auth/domain"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"

	"hms-backend/internal/modules/appointments/handler"
	"hms-backend/internal/modules/appointments/repository"
	"hms-backend/internal/modules/appointments/service"
)

// Dependencies are the shared, process-wide collaborators the appointments
// module needs. Patients and Doctors are the already-constructed
// repositories from their own modules — Appointments reads from both but
// never mounts routes on their behalf.
type Dependencies struct {
	DB           *mongo.Database
	Logger       *zap.Logger
	Audit        audit.Recorder
	Authenticate gin.HandlerFunc
	Patients     patientsRepo.PatientRepository
	Doctors      doctorsRepo.DoctorRepository
}

// RegisterRoutes wires up the appointments module and mounts its routes.
//
// ChangeStatus is intentionally gated on the *view* permission rather than
// *manage*: a Doctor (view-only per the RBAC matrix) still needs to hit
// this endpoint to progress their own consultations through
// IN_PROGRESS/COMPLETED/NO_SHOW. The finer-grained "manage vs. own-and-
// clinical-only" distinction is enforced inside the service — see
// service.authorizeStatusChange.
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) {
	appointmentRepo := repository.NewAppointmentRepository(deps.DB)

	appointmentSvc := service.NewAppointmentService(service.Deps{
		Appointments: appointmentRepo,
		Patients:     deps.Patients,
		Doctors:      deps.Doctors,
		Audit:        deps.Audit,
		Logger:       deps.Logger,
	})

	h := handler.NewHandler(appointmentSvc)

	group := rg.Group("/appointments", deps.Authenticate)
	group.POST("", middleware.RequirePermission(authDomain.PermAppointmentManage), h.Book)
	group.GET("", middleware.RequirePermission(authDomain.PermAppointmentView), h.Search)
	group.GET("/:id", middleware.RequirePermission(authDomain.PermAppointmentView), h.GetByID)
	group.PATCH("/:id/status", middleware.RequirePermission(authDomain.PermAppointmentView), h.ChangeStatus)
	group.POST("/:id/reschedule", middleware.RequirePermission(authDomain.PermAppointmentManage), h.Reschedule)
}
