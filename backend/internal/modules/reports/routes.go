package reports

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	"hms-backend/internal/middleware"
	authDomain "hms-backend/internal/modules/auth/domain"

	"hms-backend/internal/modules/reports/handler"
	"hms-backend/internal/modules/reports/repository"
	"hms-backend/internal/modules/reports/service"
)

// Dependencies are the shared, process-wide collaborators the reporting
// module needs.
type Dependencies struct {
	DB           *mongo.Database
	Authenticate gin.HandlerFunc
}

// RegisterRoutes wires up the reporting module and mounts its routes.
// Every endpoint is gated on report.export (Admin only, per the seeded
// RBAC matrix) — the dashboard this module feeds is an executive/
// operational reporting surface, not general staff-facing data, and
// report.export is the one permission the blueprint's own list already
// names for it.
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) {
	reportsRepo := repository.NewReportsRepository(deps.DB)

	reportsSvc := service.NewReportsService(service.Deps{Reports: reportsRepo})

	h := handler.NewHandler(reportsSvc)

	group := rg.Group("/reports", deps.Authenticate, middleware.RequirePermission(authDomain.PermReportExport))
	group.GET("/summary", h.Summary)
	group.GET("/appointments-by-status", h.AppointmentsByStatus)
	group.GET("/revenue-trend", h.RevenueTrend)
	group.GET("/patient-registrations-trend", h.PatientRegistrationTrend)
	group.GET("/doctor-workload", h.DoctorWorkload)
	group.GET("/lab-test-volume", h.LabTestVolume)
	group.GET("/export", h.Export)
}
