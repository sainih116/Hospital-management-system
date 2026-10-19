package labtests

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/internal/audit"
	"hms-backend/internal/middleware"
	authDomain "hms-backend/internal/modules/auth/domain"
	"hms-backend/internal/modules/labtests/handler"
	"hms-backend/internal/modules/labtests/repository"
	"hms-backend/internal/modules/labtests/service"
)

// Module exposes the pieces other modules need to reuse — notably the
// repository, since LabOrders validates/snapshots test data without
// standing up a second connection to the same collection.
type Module struct {
	Repository repository.LabTestRepository
}

// Dependencies are the shared, process-wide collaborators the lab tests
// module needs.
type Dependencies struct {
	DB           *mongo.Database
	Logger       *zap.Logger
	Audit        audit.Recorder
	Authenticate gin.HandlerFunc
}

// RegisterRoutes wires up the lab tests module and mounts its routes.
// Reads only require an authenticated session (dashboard.view, granted to
// every role) since doctors/lab techs both need the catalog for dropdowns;
// writes require labtest.manage (Admin only).
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) Module {
	testRepo := repository.NewLabTestRepository(deps.DB)

	testSvc := service.NewLabTestService(service.Deps{Tests: testRepo, Audit: deps.Audit, Logger: deps.Logger})

	h := handler.NewHandler(testSvc)

	group := rg.Group("/lab-tests", deps.Authenticate)
	group.GET("", middleware.RequirePermission(authDomain.PermDashboardView), h.List)
	group.GET("/:id", middleware.RequirePermission(authDomain.PermDashboardView), h.GetByID)
	group.POST("", middleware.RequirePermission(authDomain.PermLabTestManage), h.Create)
	group.PATCH("/:id", middleware.RequirePermission(authDomain.PermLabTestManage), h.Update)

	return Module{Repository: testRepo}
}
