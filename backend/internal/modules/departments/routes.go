package departments

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	authDomain "hms-backend/internal/modules/auth/domain"
	"hms-backend/internal/modules/departments/handler"
	"hms-backend/internal/modules/departments/repository"
	"hms-backend/internal/modules/departments/service"
)

// Module exposes the pieces other modules need to reuse — notably the
// repository, since Doctors validates/denormalizes department data without
// standing up a second connection to the same collection.
type Module struct {
	Repository repository.DepartmentRepository
	Service    service.DepartmentService
}

// Dependencies are the shared, process-wide collaborators the departments
// module needs.
type Dependencies struct {
	DB           *mongo.Database
	Logger       *zap.Logger
	Audit        audit.Recorder
	Authenticate gin.HandlerFunc
}

// RegisterRoutes wires up the departments module and mounts its routes.
// Reads (list/get) only require an authenticated session (dashboard.view,
// granted to every role) since every module needs department reference
// data for dropdowns; writes require department.manage (Admin only).
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) Module {
	departmentRepo := repository.NewDepartmentRepository(deps.DB)

	departmentSvc := service.NewDepartmentService(service.Deps{
		Departments: departmentRepo,
		Audit:       deps.Audit,
		Logger:      deps.Logger,
	})

	h := handler.NewHandler(departmentSvc)

	group := rg.Group("/departments", deps.Authenticate)
	group.GET("", middleware.RequirePermission(authDomain.PermDashboardView), h.List)
	group.GET("/:id", middleware.RequirePermission(authDomain.PermDashboardView), h.GetByID)
	group.POST("", middleware.RequirePermission(authDomain.PermDepartmentManage), h.Create)
	group.PATCH("/:id", middleware.RequirePermission(authDomain.PermDepartmentManage), h.Update)

	return Module{Repository: departmentRepo, Service: departmentSvc}
}
