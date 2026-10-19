package doctors

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	authDomain "hms-backend/internal/modules/auth/domain"
	authRepo "hms-backend/internal/modules/auth/repository"
	deptRepo "hms-backend/internal/modules/departments/repository"
	"hms-backend/internal/modules/doctors/handler"
	"hms-backend/internal/modules/doctors/repository"
	"hms-backend/internal/modules/doctors/service"
)

// Module exposes the pieces other modules need to reuse — notably the
// repository, since Appointments validates/snapshots doctor data (and
// checks availability) without standing up a second connection to the
// same collection.
type Module struct {
	Repository repository.DoctorRepository
}

// Dependencies are the shared, process-wide collaborators the doctors
// module needs. Users and Departments are the already-constructed
// repositories from the auth and departments modules — Doctors reads from
// both but never mounts routes on their behalf.
type Dependencies struct {
	DB           *mongo.Database
	Logger       *zap.Logger
	Audit        audit.Recorder
	Authenticate gin.HandlerFunc
	Users        authRepo.UserRepository
	Departments  deptRepo.DepartmentRepository
}

// RegisterRoutes wires up the doctors module and mounts its routes.
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) Module {
	doctorRepo := repository.NewDoctorRepository(deps.DB)

	doctorSvc := service.NewDoctorService(service.Deps{
		Doctors:     doctorRepo,
		Users:       deps.Users,
		Departments: deps.Departments,
		Audit:       deps.Audit,
		Logger:      deps.Logger,
	})

	h := handler.NewHandler(doctorSvc)

	group := rg.Group("/doctors", deps.Authenticate)
	group.POST("", middleware.RequirePermission(authDomain.PermDoctorManage), h.Create)
	group.GET("", middleware.RequirePermission(authDomain.PermDoctorRead), h.Search)
	group.GET("/:id", middleware.RequirePermission(authDomain.PermDoctorRead), h.GetByID)
	group.GET("/:id/schedule", middleware.RequirePermission(authDomain.PermDoctorRead), h.Schedule)
	group.PATCH("/:id", middleware.RequirePermission(authDomain.PermDoctorManage), h.Update)
	group.DELETE("/:id", middleware.RequirePermission(authDomain.PermDoctorManage), h.Delete)

	return Module{Repository: doctorRepo}
}
