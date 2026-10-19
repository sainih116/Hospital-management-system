package auditlogs

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	"hms-backend/internal/middleware"
	authDomain "hms-backend/internal/modules/auth/domain"

	"hms-backend/internal/modules/auditlogs/handler"
	"hms-backend/internal/modules/auditlogs/repository"
	"hms-backend/internal/modules/auditlogs/service"
)

// Dependencies are the shared, process-wide collaborators the audit
// logging module needs.
type Dependencies struct {
	DB           *mongo.Database
	Authenticate gin.HandlerFunc
}

// RegisterRoutes wires up the audit logging module and mounts its routes.
// Gated on settings.manage rather than a bespoke audit.read permission —
// the blueprint's own permission list (Section 3, mirrored in
// auth/domain/permission.go) never names one, and settings.manage is
// already the Admin-only, system-administration-adjacent grant this
// endpoint belongs under.
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) {
	logRepo := repository.NewAuditLogRepository(deps.DB)

	logSvc := service.NewAuditLogService(service.Deps{Logs: logRepo})

	h := handler.NewHandler(logSvc)

	group := rg.Group("/audit-logs", deps.Authenticate)
	group.GET("", middleware.RequirePermission(authDomain.PermSettingsManage), h.Search)
}
