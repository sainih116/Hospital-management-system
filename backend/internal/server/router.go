package server

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/docs"
	"hms-backend/internal/audit"
	"hms-backend/internal/config"
	"hms-backend/internal/handlers/health"
	"hms-backend/internal/middleware"
	appointments "hms-backend/internal/modules/appointments"
	auditlogs "hms-backend/internal/modules/auditlogs"
	auth "hms-backend/internal/modules/auth"
	billing "hms-backend/internal/modules/billing"
	departments "hms-backend/internal/modules/departments"
	doctors "hms-backend/internal/modules/doctors"
	ipd "hms-backend/internal/modules/ipd"
	laborders "hms-backend/internal/modules/laborders"
	labtests "hms-backend/internal/modules/labtests"
	medicalrecords "hms-backend/internal/modules/medicalrecords"
	patients "hms-backend/internal/modules/patients"
	pharmacy "hms-backend/internal/modules/pharmacy"
	prescriptions "hms-backend/internal/modules/prescriptions"
	reports "hms-backend/internal/modules/reports"
	"hms-backend/internal/notifier"

	"github.com/redis/go-redis/v9"
)

// Dependencies bundles everything the router needs to wire up handlers.
type Dependencies struct {
	Config      *config.Config
	Logger      *zap.Logger
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
	RedisClient *redis.Client
	Notifier    notifier.Notifier
	Version     string
}

// NewRouter builds the fully configured Gin engine: global middleware,
// health endpoints, Swagger UI and the versioned API route group that
// later phases attach their module routes to.
func NewRouter(deps Dependencies) *gin.Engine {
	if deps.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(
		middleware.RequestID(),
		middleware.Recovery(deps.Logger),
		middleware.Logger(deps.Logger),
		middleware.CORS(deps.Config.AllowedOrigins),
	)

	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Host = ""

	healthHandler := health.NewHandler(deps.MongoClient, deps.RedisClient, deps.Version)
	router.GET("/health", healthHandler.Live)
	router.GET("/health/ready", healthHandler.Ready)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := router.Group("/api/v1")
	registerV1Routes(v1, deps)

	return router
}

// registerV1Routes is the single attachment point for versioned API module
// routers. Each module returns whatever downstream modules need to reuse
// (most importantly, the auth module's Authenticate middleware).
func registerV1Routes(v1 *gin.RouterGroup, deps Dependencies) {
	// deps.MongoDB is only nil in tests that exercise the health endpoints
	// in isolation; a real server run always provides a live connection.
	if deps.MongoDB == nil {
		return
	}

	authModule := auth.RegisterRoutes(v1, auth.Dependencies{
		DB:       deps.MongoDB,
		Config:   deps.Config,
		Logger:   deps.Logger,
		Notifier: deps.Notifier,
	})

	auditRecorder := audit.NewMongoRecorder(deps.MongoDB)

	patientsModule := patients.RegisterRoutes(v1, patients.Dependencies{
		DB:           deps.MongoDB,
		Logger:       deps.Logger,
		Audit:        auditRecorder,
		Authenticate: authModule.Authenticate,
	})

	departmentsModule := departments.RegisterRoutes(v1, departments.Dependencies{
		DB:           deps.MongoDB,
		Logger:       deps.Logger,
		Audit:        auditRecorder,
		Authenticate: authModule.Authenticate,
	})

	doctorsModule := doctors.RegisterRoutes(v1, doctors.Dependencies{
		DB:           deps.MongoDB,
		Logger:       deps.Logger,
		Audit:        auditRecorder,
		Authenticate: authModule.Authenticate,
		Users:        authModule.Users,
		Departments:  departmentsModule.Repository,
	})

	appointments.RegisterRoutes(v1, appointments.Dependencies{
		DB:           deps.MongoDB,
		Logger:       deps.Logger,
		Audit:        auditRecorder,
		Authenticate: authModule.Authenticate,
		Patients:     patientsModule.Repository,
		Doctors:      doctorsModule.Repository,
	})

	medicalRecordsModule := medicalrecords.RegisterRoutes(v1, medicalrecords.Dependencies{
		DB:           deps.MongoDB,
		Logger:       deps.Logger,
		Audit:        auditRecorder,
		Authenticate: authModule.Authenticate,
		Patients:     patientsModule.Repository,
		Doctors:      doctorsModule.Repository,
	})

	prescriptionsModule := prescriptions.RegisterRoutes(v1, prescriptions.Dependencies{
		DB:            deps.MongoDB,
		Logger:        deps.Logger,
		Audit:         auditRecorder,
		Authenticate:  authModule.Authenticate,
		Patients:      patientsModule.Repository,
		Doctors:       doctorsModule.Repository,
		MedicalRecord: medicalRecordsModule.Repository,
	})

	labTestsModule := labtests.RegisterRoutes(v1, labtests.Dependencies{
		DB:           deps.MongoDB,
		Logger:       deps.Logger,
		Audit:        auditRecorder,
		Authenticate: authModule.Authenticate,
	})

	laborders.RegisterRoutes(v1, laborders.Dependencies{
		DB:           deps.MongoDB,
		Logger:       deps.Logger,
		Audit:        auditRecorder,
		Authenticate: authModule.Authenticate,
		Patients:     patientsModule.Repository,
		Doctors:      doctorsModule.Repository,
		Tests:        labTestsModule.Repository,
		Users:        authModule.Users,
	})

	pharmacy.RegisterRoutes(v1, pharmacy.Dependencies{
		DB:            deps.MongoDB,
		Logger:        deps.Logger,
		Audit:         auditRecorder,
		Authenticate:  authModule.Authenticate,
		Prescriptions: prescriptionsModule.Repository,
		Users:         authModule.Users,
	})

	billing.RegisterRoutes(v1, billing.Dependencies{
		DB:           deps.MongoDB,
		Logger:       deps.Logger,
		Audit:        auditRecorder,
		Authenticate: authModule.Authenticate,
		Patients:     patientsModule.Repository,
		Users:        authModule.Users,
	})

	auditlogs.RegisterRoutes(v1, auditlogs.Dependencies{
		DB:           deps.MongoDB,
		Authenticate: authModule.Authenticate,
	})

	reports.RegisterRoutes(v1, reports.Dependencies{
		DB:           deps.MongoDB,
		Authenticate: authModule.Authenticate,
	})

	ipd.RegisterRoutes(v1, ipd.Dependencies{
		DB:           deps.MongoDB,
		Logger:       deps.Logger,
		Audit:        auditRecorder,
		Authenticate: authModule.Authenticate,
		Patients:     patientsModule.Repository,
		Doctors:      doctorsModule.Repository,
	})
}
