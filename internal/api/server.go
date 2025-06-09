package api

import (
	"context"
	"net/http"
	"time"

	"aeswibon.com/github/gitopsctl/internal/api/app"
	"aeswibon.com/github/gitopsctl/internal/api/cluster"
	"aeswibon.com/github/gitopsctl/internal/controller"
	appcore "aeswibon.com/github/gitopsctl/internal/core/app"
	clustercore "aeswibon.com/github/gitopsctl/internal/core/cluster"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// Server represents the API server.
// It holds the Echo instance, logger, applications store, and controller reference.
type Server struct {
	// e is the Echo instance used for handling HTTP requests.
	e *echo.Echo
	// logger is the zap.Logger instance used for logging.
	logger *zap.Logger
	// apps is the reference to the applications store, which holds registered applications.
	apps *appcore.Applications
	// clusters is the reference to the clusters store, which holds registered Kubernetes clusters.
	clusters *clustercore.Clusters
	// controller is the reference to the main controller that manages application synchronization.
	controller *controller.Controller
}

// NewServer creates a new API server instance.
// It initializes the Echo instance, sets up middleware, and registers routes.
func NewServer(logger *zap.Logger, apps *appcore.Applications, clusters *clustercore.Clusters, ctrl *controller.Controller) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Validator = NewCustomValidator()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339_nano}","id":"${id}","remote_ip":"${remote_ip}",` +
			`"host":"${host}","method":"${method}","uri":"${uri}","status":${status}, "latency":"${latency_human}"` +
			`,"bytes_in":${bytes_in},"bytes_out":${bytes_out}}` + "\n",
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	s := &Server{
		e:          e,
		logger:     logger,
		apps:       apps,
		clusters:   clusters,
		controller: ctrl,
	}

	s.registerRoutes()
	return s
}

// RegisterRoutes defines all API endpoints.
// It sets up the routes for managing applications, health checks, and other API functionalities.
func (s *Server) registerRoutes() {
	v1 := s.e.Group("/api/v1")

	appHandler := app.NewHandler(s.logger, s.apps, s.clusters, s.controller)
	clusterHandler := cluster.NewHandler(s.logger, s.clusters, s.apps, s.controller)

	app.RegisterRoutes(v1, appHandler)
	cluster.RegisterRoutes(v1, clusterHandler)

	s.e.GET("/health", s.HealthCheck)

}

// Echo returns the Echo instance used by the server.
// This is useful for accessing Echo-specific methods or configurations outside the server struct.
func (s *Server) Echo() *echo.Echo {
	return s.e
}

// Start starts the HTTP server.
// It binds the server to the specified address and begins listening for incoming requests.
func (s *Server) Start(address string) error {
	s.logger.Info("Starting API server", zap.String("address", address))
	return s.e.Start(address)
}

// Stop stops the HTTP server.
// It gracefully shuts down the server, allowing ongoing requests to complete.
// This method can be called from the controller or directly via an API endpoint.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down API server...")
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.e.Shutdown(timeoutCtx)
}

// HealthCheck is a simple endpoint to check if the API server is running.
// It responds with a 200 OK status and a simple message.
// This is useful for monitoring and health checks in production environments.
func (s *Server) HealthCheck(c echo.Context) error {
	// Simple health check: just respond with 200 OK
	return c.String(http.StatusOK, "OK")
}
