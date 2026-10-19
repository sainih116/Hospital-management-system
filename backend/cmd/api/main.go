// Command api is the entry point for the HMS backend HTTP server.
//
// @title                       Hospital Management System API
// @version                     1.0
// @description                 REST API for the Hospital Management System (Auth, Patients, Doctors, Appointments, EMR, Laboratory, Pharmacy, Billing, IPD/Wards, Reporting).
// @termsOfService               http://swagger.io/terms/
// @contact.name                HMS Engineering
// @license.name                Proprietary
// @BasePath                    /api/v1
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and the JWT access token.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"hms-backend/internal/audit"
	"hms-backend/internal/config"
	"hms-backend/internal/database"
	"hms-backend/internal/logger"
	appointmentsRepo "hms-backend/internal/modules/appointments/repository"
	authRepo "hms-backend/internal/modules/auth/repository"
	billingRepo "hms-backend/internal/modules/billing/repository"
	departmentsRepo "hms-backend/internal/modules/departments/repository"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	ipdRepo "hms-backend/internal/modules/ipd/repository"
	laborders "hms-backend/internal/modules/laborders/repository"
	labtestsRepo "hms-backend/internal/modules/labtests/repository"
	medicalRecordsRepo "hms-backend/internal/modules/medicalrecords/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"
	pharmacyRepo "hms-backend/internal/modules/pharmacy/repository"
	prescriptionsRepo "hms-backend/internal/modules/prescriptions/repository"
	"hms-backend/internal/notifier"
	"hms-backend/internal/server"
)

// version is the build version, overridable at build time via:
//
//	go build -ldflags "-X main.version=1.2.3"
var version = "dev"

func main() {
	if err := run(); err != nil {
		os.Stderr.WriteString("fatal: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log, err := logger.New(cfg.Env, cfg.LogLevel)
	if err != nil {
		return err
	}
	defer func() { _ = log.Sync() }()

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mongoClient, mongoDB, err := database.ConnectMongo(rootCtx, cfg.Mongo)
	if err != nil {
		log.Fatal("failed to connect to mongodb", zap.Error(err))
	}
	defer func() {
		if err := database.DisconnectMongo(context.Background(), mongoClient); err != nil {
			log.Error("failed to disconnect mongodb", zap.Error(err))
		}
	}()

	if err := authRepo.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure auth module indexes", zap.Error(err))
	}
	if err := patientsRepo.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure patients module indexes", zap.Error(err))
	}
	if err := departmentsRepo.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure departments module indexes", zap.Error(err))
	}
	if err := doctorsRepo.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure doctors module indexes", zap.Error(err))
	}
	if err := appointmentsRepo.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure appointments module indexes", zap.Error(err))
	}
	if err := medicalRecordsRepo.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure medical records module indexes", zap.Error(err))
	}
	if err := labtestsRepo.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure lab tests module indexes", zap.Error(err))
	}
	if err := laborders.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure lab orders module indexes", zap.Error(err))
	}
	if err := prescriptionsRepo.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure prescriptions module indexes", zap.Error(err))
	}
	if err := pharmacyRepo.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure pharmacy module indexes", zap.Error(err))
	}
	if err := billingRepo.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure billing module indexes", zap.Error(err))
	}
	if err := audit.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure audit log indexes", zap.Error(err))
	}
	if err := ipdRepo.EnsureIndexes(rootCtx, mongoDB); err != nil {
		log.Fatal("failed to ensure IPD module indexes", zap.Error(err))
	}

	redisClient, err := database.ConnectRedis(rootCtx, cfg.Redis)
	if err != nil {
		log.Warn("redis unavailable, continuing without cache layer", zap.Error(err))
		redisClient = nil
	}
	defer func() {
		if err := database.DisconnectRedis(redisClient); err != nil {
			log.Error("failed to disconnect redis", zap.Error(err))
		}
	}()

	router := server.NewRouter(server.Dependencies{
		Config:      cfg,
		Logger:      log,
		MongoClient: mongoClient,
		MongoDB:     mongoDB,
		RedisClient: redisClient,
		Notifier:    notifier.NewLogNotifier(log),
		Version:     version,
	})

	httpServer := server.New(":"+cfg.Port, router)

	serverErrCh := make(chan error, 1)
	go func() {
		log.Info("http server starting",
			zap.String("addr", httpServer.Addr),
			zap.String("env", cfg.Env),
			zap.String("version", version),
		)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	select {
	case <-rootCtx.Done():
		log.Info("shutdown signal received")
	case err := <-serverErrCh:
		if err != nil {
			return err
		}
	}

	log.Info("shutting down http server", zap.Duration("timeout", cfg.ShutdownTimeout))
	if err := server.Shutdown(context.Background(), httpServer, cfg.ShutdownTimeout); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
		return err
	}

	log.Info("shutdown complete")
	return nil
}
