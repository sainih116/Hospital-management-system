package auth

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"hms-backend/internal/config"
	"hms-backend/internal/middleware"
	"hms-backend/internal/modules/auth/domain"
	"hms-backend/internal/modules/auth/handler"
	"hms-backend/internal/modules/auth/repository"
	"hms-backend/internal/modules/auth/service"
	"hms-backend/internal/notifier"
	"hms-backend/internal/security"
)

// Module bundles everything RegisterRoutes builds, so other modules
// (Patients, Doctors, Appointments, ...) can reuse the same Authenticate
// middleware, permission constants and user repository without
// re-deriving or duplicating them.
type Module struct {
	Authenticate gin.HandlerFunc
	Service      service.AuthService
	Users        repository.UserRepository
}

// Dependencies are the shared, process-wide collaborators the auth module
// needs, supplied by the composition root (internal/server).
type Dependencies struct {
	DB       *mongo.Database
	Config   *config.Config
	Logger   *zap.Logger
	Notifier notifier.Notifier
}

// RegisterRoutes wires up the auth module's repositories, services and
// handlers, mounts its routes under the given group, and returns the
// pieces (notably the Authenticate middleware) that later phases attach to
// their own route groups.
func RegisterRoutes(rg *gin.RouterGroup, deps Dependencies) Module {
	tokenService := security.NewTokenService(deps.Config.JWT)

	userRepo := repository.NewUserRepository(deps.DB)
	roleRepo := repository.NewRoleRepository(deps.DB)
	refreshTokenRepo := repository.NewRefreshTokenRepository(deps.DB)
	resetTokenRepo := repository.NewPasswordResetRepository(deps.DB)

	authSvc := service.NewAuthService(service.Deps{
		Users:         userRepo,
		Roles:         roleRepo,
		RefreshTokens: refreshTokenRepo,
		ResetTokens:   resetTokenRepo,
		Tokens:        tokenService,
		Notifier:      deps.Notifier,
		Logger:        deps.Logger,
		FrontendURL:   deps.Config.FrontendBaseURL,
	})

	h := handler.NewHandler(authSvc)

	authenticate := middleware.Authenticate(tokenService, userRepo.IsActiveUser)

	strictRateLimit := middleware.RateLimit(10, 5)

	group := rg.Group("/auth")
	group.POST("/login", strictRateLimit, h.Login)
	group.POST("/refresh", h.Refresh)
	group.POST("/logout", h.Logout)
	group.POST("/password/forgot", strictRateLimit, h.ForgotPassword)
	group.POST("/password/reset", strictRateLimit, h.ResetPassword)
	group.GET("/me", authenticate, h.Me)
	group.POST("/register", authenticate, middleware.RequirePermission(domain.PermUserManage), h.Register)

	rg.GET("/users", authenticate, middleware.RequirePermission(domain.PermUserManage), h.ListUsers)

	return Module{
		Authenticate: authenticate,
		Service:      authSvc,
		Users:        userRepo,
	}
}
