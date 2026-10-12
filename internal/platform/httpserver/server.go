package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"educore/internal/config"
	"educore/internal/middleware"
	"educore/internal/modules/auth"
	"educore/internal/modules/courses"
	"educore/internal/modules/curricula"
	"educore/internal/modules/faculties"
	"educore/internal/modules/lecturers"
	"educore/internal/modules/students"
	"educore/internal/modules/study_programs"
	"educore/internal/platform/database"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server represents the HTTP server.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
	engine     *gin.Engine
	dbPool     *pgxpool.Pool
}

// New creates and configures a new Server instance with database health integration.
func New(cfg *config.Config, logger *slog.Logger, dbPool *pgxpool.Pool) *Server {
	// Set Gin mode based on app environment
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.AppEnv == "testing" {
		gin.SetMode(gin.TestMode)
	}

	engine := gin.New()
	engine.HandleMethodNotAllowed = true

	// Register global middleware
	engine.Use(middleware.RequestID())
	engine.Use(middleware.RequestLogger(logger))
	engine.Use(middleware.Recovery(logger))
	engine.Use(middleware.BodyLimit(middleware.DefaultMaxBodyBytes))

	// Global 404 Not Found handler
	engine.NoRoute(func(c *gin.Context) {
		reqID := middleware.GetRequestID(c)
		response.RenderError(c, appErrors.ErrRouteNotFound(), reqID)
	})

	// Global 405 Method Not Allowed handler
	engine.NoMethod(func(c *gin.Context) {
		reqID := middleware.GetRequestID(c)
		response.RenderError(c, appErrors.ErrMethodNotAllowed(), reqID)
	})

	// Application health check endpoints
	engine.GET("/health/live", func(c *gin.Context) {
		response.OK(c, "Layanan aplikasi berjalan", gin.H{
			"status": "up",
		}, nil)
	})

	engine.GET("/health/ready", func(c *gin.Context) {
		reqID := middleware.GetRequestID(c)
		dbCheck := "up"

		if dbPool != nil {
			if err := database.CheckHealth(c.Request.Context(), dbPool, cfg.DBHealthTimeout); err != nil {
				dbCheck = "down"
				logger.Warn("readiness check failed: database unavailable",
					slog.String("request_id", reqID),
					slog.Any("error", err),
				)
			}
		} else {
			dbCheck = "uninitialized"
		}

		if dbCheck != "up" {
			response.Error(c, http.StatusServiceUnavailable, appErrors.CodeDependencyUnavailable,
				"Layanan aplikasi belum siap menerima traffic",
				[]response.ErrorDetail{
					{Field: "database", Message: "Koneksi database tidak tersedia"},
				},
				reqID,
			)
			return
		}

		response.OK(c, "Layanan aplikasi siap menerima traffic", gin.H{
			"status": "ready",
			"checks": gin.H{
				"http_server": "up",
				"database":    "up",
			},
		}, nil)
	})

	// REST API v1 routes
	v1 := engine.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			response.OK(c, "EduCore API v1 operational", gin.H{
				"version":     "1.0.0",
				"environment": cfg.AppEnv,
			}, nil)
		})

		// Register Authentication and User Profile routes if database is available
		if dbPool != nil {
			hasher := auth.NewArgon2Hasher(cfg.Argon2Memory, cfg.Argon2Iterations, cfg.Argon2Parallelism, cfg.Argon2SaltLength, cfg.Argon2KeyLength)
			tokenMgr := auth.NewTokenManager(cfg)
			repo := auth.NewRepository(dbPool)
			service := auth.NewService(repo, hasher, tokenMgr, cfg, logger)
			handler := auth.NewHandler(service, logger)

			authMiddleware := middleware.Authenticate(tokenMgr)
			adminMiddleware := middleware.RequireRoles("admin")
			adminStaffMiddleware := middleware.RequireRoles("admin", "akademik")

			auth.RegisterRoutes(v1, handler, authMiddleware, adminMiddleware)

			// Register Faculties routes
			facRepo := faculties.NewRepository(dbPool)
			facService := faculties.NewService(facRepo, logger)
			facHandler := faculties.NewHandler(facService, logger)
			faculties.RegisterRoutes(v1, facHandler, authMiddleware, adminStaffMiddleware)

			// Register Study Programs routes
			spRepo := study_programs.NewRepository(dbPool)
			spService := study_programs.NewService(spRepo, logger)
			spHandler := study_programs.NewHandler(spService, logger)
			study_programs.RegisterRoutes(v1, spHandler, authMiddleware, adminStaffMiddleware)

			// Register Lecturers routes
			lecRepo := lecturers.NewRepository(dbPool)
			lecService := lecturers.NewService(lecRepo, logger)
			lecHandler := lecturers.NewHandler(lecService, logger)
			lecturers.RegisterRoutes(v1, lecHandler, authMiddleware, adminStaffMiddleware)

			// Register Students routes
			studRepo := students.NewRepository(dbPool)
			studService := students.NewService(studRepo, logger)
			studHandler := students.NewHandler(studService, logger)
			students.RegisterRoutes(v1, studHandler, authMiddleware, adminStaffMiddleware)

			// Register Courses routes
			courseRepo := courses.NewRepository(dbPool)
			courseService := courses.NewService(courseRepo, logger)
			courseHandler := courses.NewHandler(courseService, logger)
			courses.RegisterRoutes(v1, courseHandler, authMiddleware, adminStaffMiddleware)

			// Register Curricula routes
			curriRepo := curricula.NewRepository(dbPool)
			curriService := curricula.NewService(curriRepo, logger)
			curriHandler := curricula.NewHandler(curriService, logger)
			curricula.RegisterRoutes(v1, curriHandler, authMiddleware, adminStaffMiddleware)
		}
	}

	addr := fmt.Sprintf(":%d", cfg.AppPort)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	return &Server{
		httpServer: httpServer,
		logger:     logger,
		engine:     engine,
	}
}

// Router returns the Gin engine instance, useful for integration testing.
func (s *Server) Router() *gin.Engine {
	return s.engine
}

// Engine returns the underlying Gin engine instance.
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// Start runs the HTTP server in a goroutine and returns a channel for listening to errors.
func (s *Server) Start() <-chan error {
	errChan := make(chan error, 1)

	go func() {
		s.logger.Info("Starting HTTP server", slog.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
		close(errChan)
	}()

	return errChan
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}
