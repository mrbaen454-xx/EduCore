package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"educore/internal/config"
	"educore/internal/middleware"
	appErrors "educore/internal/shared/errors"
	"educore/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// Server represents the HTTP server.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
	engine     *gin.Engine
}

// New creates and configures a new Server instance.
func New(cfg *config.Config, logger *slog.Logger) *Server {
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
		// Readiness check currently verifies HTTP server readiness; DB check will be integrated in Phase 04
		response.OK(c, "Layanan aplikasi siap menerima traffic", gin.H{
			"status": "ready",
			"checks": gin.H{
				"http_server": "up",
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
