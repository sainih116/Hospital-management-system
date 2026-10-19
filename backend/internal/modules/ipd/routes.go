package ipd

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	authDomain "hms-backend/internal/modules/auth/domain"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"

	"hms-backend/internal/modules/ipd/handler"
	"hms-backend/internal/modules/ipd/repository"
	"hms-backend/internal/modules/ipd/service"
)

// Dependencies are the shared, process-wide collaborators the IPD module
// needs.
type Dependencies struct {
	DB           *mongo.Database
	Logger       *zap.Logger
	Audit        audit.Recorder
	Authenticate gin.HandlerFunc
	Patients     patientsRepo.PatientRepository
	Doctors      doctorsRepo.DoctorRepository
}

// RegisterRoutes wires up the IPD module and mounts its routes. Reuses
// the ipd.read/ipd.write permissions already seeded in Phase 1 (Admin,
// Doctor and Nurse have both; Receptionist has ipd.read only) — no new
// permission codes were needed since the blueprint's own RBAC matrix
// already named these for the IPD/Wards row.
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) {
	wardRepo := repository.NewWardRepository(deps.DB)
	bedRepo := repository.NewBedRepository(deps.DB)
	admissionRepo := repository.NewAdmissionRepository(deps.DB)

	ipdSvc := service.NewIPDService(service.Deps{
		Wards: wardRepo, Beds: bedRepo, Admissions: admissionRepo,
		Patients: deps.Patients, Doctors: deps.Doctors, Audit: deps.Audit, Logger: deps.Logger,
	})

	h := handler.NewHandler(ipdSvc)

	group := rg.Group("/ipd", deps.Authenticate)

	group.GET("/wards", middleware.RequirePermission(authDomain.PermIPDRead), h.ListWards)
	group.POST("/wards", middleware.RequirePermission(authDomain.PermIPDWrite), h.CreateWard)
	group.PATCH("/wards/:id", middleware.RequirePermission(authDomain.PermIPDWrite), h.UpdateWard)

	group.GET("/beds", middleware.RequirePermission(authDomain.PermIPDRead), h.ListBeds)
	group.POST("/beds", middleware.RequirePermission(authDomain.PermIPDWrite), h.CreateBed)
	group.PATCH("/beds/:id", middleware.RequirePermission(authDomain.PermIPDWrite), h.UpdateBedStatus)

	group.GET("/admissions", middleware.RequirePermission(authDomain.PermIPDRead), h.SearchAdmissions)
	group.GET("/admissions/:id", middleware.RequirePermission(authDomain.PermIPDRead), h.GetAdmission)
	group.POST("/admissions", middleware.RequirePermission(authDomain.PermIPDWrite), h.AdmitPatient)
	group.POST("/admissions/:id/discharge", middleware.RequirePermission(authDomain.PermIPDWrite), h.DischargePatient)
}
