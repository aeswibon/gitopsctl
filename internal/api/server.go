package api

import (
	"fmt"
	"net/http"
	"time"

	"aeswibon.com/github/gitopsctl/internal/common"
	"aeswibon.com/github/gitopsctl/internal/controller"
	"aeswibon.com/github/gitopsctl/internal/core/app"
	"github.com/go-playground/validator/v10" // For validation
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// CustomValidator holds the go-playground validator instance.
//
// It implements the echo.Validator interface to integrate with Echo's validation system.
type CustomValidator struct {
	validator *validator.Validate
}

// Validate validates the input struct.
//
// It uses the go-playground validator to check the struct fields based on tags.
// If validation fails, it returns an HTTP error with status 400 Bad Request.
func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

// Server represents the API server.
//
// It holds the Echo instance, logger, applications store, and controller reference.
type Server struct {
	// e is the Echo instance used for handling HTTP requests.
	e *echo.Echo
	// logger is the zap.Logger instance used for logging.
	logger *zap.Logger
	// apps is the reference to the applications store, which holds registered applications.
	apps *app.Applications
	// controller is the reference to the main controller that manages application synchronization.
	controller *controller.Controller
}

// NewServer creates a new API server instance.
//
// It initializes the Echo instance, sets up middleware, and registers routes.
func NewServer(logger *zap.Logger, applications *app.Applications, ctrl *controller.Controller) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Register custom validation for Git URLs
	v := validator.New()
	v.RegisterValidation("giturl", func(fl validator.FieldLevel) bool {
		return common.IsValidGitURL(fl.Field().String())
	})
	v.RegisterValidation("path", func(fl validator.FieldLevel) bool {
		return common.IsValidRepoPath(fl.Field().String())
	})

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339_nano}","id":"${id}","remote_ip":"${remote_ip}",` +
			`"host":"${host}","method":"${method}","uri":"${uri}","status":${status}, "latency":"${latency_human}"` +
			`,"bytes_in":${bytes_in},"bytes_out":${bytes_out}}` + "\n",
	}))
	e.Use(middleware.Recover()) // Recover from panics
	e.Use(middleware.CORS())    // Enable CORS for potential UI

	s := &Server{
		e:          e,
		logger:     logger,
		apps:       applications,
		controller: ctrl, // Store controller reference
	}

	// Register API routes
	s.registerRoutes()

	return s
}

// RegisterRoutes defines all API endpoints.
//
// It sets up the routes for managing applications, health checks, and other API functionalities.
func (s *Server) registerRoutes() {
	v1 := s.e.Group("/api/v1")

	// Applications Management
	v1.POST("/applications", s.registerApplication)
	v1.GET("/applications", s.listApplications)
	v1.GET("/applications/:name", s.getApplicationStatus)
	v1.DELETE("/applications/:name", s.unregisterApplication)
	v1.POST("/applications/:name/sync", s.triggerSync)

	// Health Check
	s.e.GET("/health", s.healthCheck)
}

// Echo returns the Echo instance used by the server.
//
// This is useful for accessing Echo-specific methods or configurations outside the server struct.
func (s *Server) Echo() *echo.Echo {
	return s.e
}

// Start starts the HTTP server.
//
// It binds the server to the specified address and begins listening for incoming requests.
func (s *Server) Start(address string) error {
	s.logger.Info("Starting API server", zap.String("address", address))
	return s.e.Start(address)
}

// Stop stops the HTTP server.
//
// It gracefully shuts down the server, allowing ongoing requests to complete.
// This method can be called from the controller or directly via an API endpoint.
func (s *Server) Stop(ctx echo.Context) error {
	s.logger.Info("Shutting down API server...")
	return s.e.Shutdown(ctx.Request().Context())
}

// --- API Handlers ---

// RegisterApplication handles the registration of a new application.
//
// It binds the request payload to a RegisterAppRequest struct, validates it,
// and either adds a new application or updates an existing one.
func (s *Server) registerApplication(c echo.Context) error {
	req := new(RegisterAppRequest)
	if err := c.Bind(req); err != nil {
		s.logger.Error("Failed to bind register application request", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request payload")
	}
	if err := c.Validate(req); err != nil {
		s.logger.Error("Failed to validate register application request", zap.Error(err))
		return err // Validation error is already an HTTPError
	}

	// Lock the applications map for modification
	s.apps.Lock()
	defer s.apps.Unlock()

	// Check if app already exists to decide between add/update
	existingApp, exists := s.apps.Get(req.Name)
	if exists {
		s.logger.Warn("Application with this name already exists. Updating it.", zap.String("name", req.Name))
		// Update existing application details
		existingApp.RepoURL = req.RepoURL
		existingApp.Branch = req.Branch
		existingApp.Path = req.Path
		existingApp.KubeconfigPath = req.KubeconfigPath
		existingApp.Interval = req.Interval
		// Re-parse polling interval
		parsedInterval, err := time.ParseDuration(req.Interval)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid interval format: %v", err))
		}
		existingApp.PollingInterval = parsedInterval
		// Reset status/message/failures on update, assuming it's a re-registration
		existingApp.Status = "Pending"
		existingApp.Message = "Application updated, awaiting next sync."
		existingApp.ConsecutiveFailures = 0

	} else {
		// Create new application
		parsedInterval, err := time.ParseDuration(req.Interval)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid interval format: %v", err))
		}
		newApp := &app.Application{
			Name:                req.Name,
			RepoURL:             req.RepoURL,
			Branch:              req.Branch,
			Path:                req.Path,
			KubeconfigPath:      req.KubeconfigPath,
			Interval:            req.Interval,
			PollingInterval:     parsedInterval,
			Status:              "Pending",
			Message:             "Application registered, awaiting first sync.",
			ConsecutiveFailures: 0,
		}
		s.apps.Add(newApp)
	}

	if err := app.SaveApplications(s.apps, app.DefaultAppConfigFile); err != nil {
		s.logger.Error("Failed to save applications after registration", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save application configuration")
	}

	// Inform the controller to potentially re-evaluate / restart the app's goroutine
	// For simplicity in Phase 2 MVP, we restart the whole controller (which restarts relevant goroutines).
	// In a more complex setup, you'd signal the controller to manage individual app lifecycle.
	// We'll handle this by having cmd/start pass a stop/restart signal to the controller.
	// For now, the next polling interval will pick up the change.

	s.logger.Info("Application registered/updated via API", zap.String("name", req.Name))
	return c.JSON(http.StatusOK, map[string]string{"message": "Application registered/updated successfully", "name": req.Name})
}

// ListApplications handles the retrieval of all registered applications.
//
// It returns a list of ApplicationResponse objects containing the details of each application.
func (s *Server) listApplications(c echo.Context) error {
	s.apps.RLock()
	defer s.apps.RUnlock()

	var responses []ApplicationResponse
	for _, app := range s.apps.List() {
		responses = append(responses, ConvertAppToResponse(app))
	}
	return c.JSON(http.StatusOK, responses)
}

// GetApplicationStatus retrieves the status of a specific application by name.
//
// It returns an ApplicationResponse object containing the application's details.
// If the application does not exist, it returns a 404 Not Found error.
// It is useful for monitoring and debugging purposes.
func (s *Server) getApplicationStatus(c echo.Context) error {
	name := c.Param("name")

	s.apps.RLock()
	defer s.apps.RUnlock()

	app, ok := s.apps.Get(name)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "Application not found")
	}
	return c.JSON(http.StatusOK, ConvertAppToResponse(app))
}

// UnregisterApplication handles the removal of an application by name.
//
// It deletes the application from the applications store and saves the updated configuration.
// If the application does not exist, it returns a 404 Not Found error.
// This is useful for cleaning up applications that are no longer needed or have been removed from the Git repository.
func (s *Server) unregisterApplication(c echo.Context) error {
	name := c.Param("name")

	s.apps.RLock()
	defer s.apps.RUnlock()

	_, exists := s.apps.Get(name)
	if !exists {
		return echo.NewHTTPError(http.StatusNotFound, "Application not found")
	}

	// Stop the controller's goroutine for this application FIRST
	s.controller.StopApp(name)
	s.apps.Lock()
	defer s.apps.Unlock()

	// Remove the application from the store
	s.apps.Delete(name)
	if err := app.SaveApplications(s.apps, app.DefaultAppConfigFile); err != nil {
		s.logger.Error("Failed to save applications after unregister", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to remove application configuration")
	}

	// Signal the controller to stop the goroutine for this app (will be part of advanced controller logic)
	// For now, the goroutine for the unregistered app will just continue to run until it gets an error
	// trying to save its state (as the app won't be in the map anymore) or until a full restart.
	// A proper solution for this would be handled by the controller via a channel.

	s.logger.Info("Application unregistered via API", zap.String("name", name))
	return c.JSON(http.StatusOK, map[string]string{"message": "Application unregistered successfully", "name": name})
}

// TriggerSync handles manual sync requests for an application.
//
// It updates the application's status to "SyncRequested" and logs the request.
// This is a placeholder for triggering an immediate sync, which would typically involve signaling the controller
// to wake up the specific application's goroutine and perform a sync now.
func (s *Server) triggerSync(c echo.Context) error {
	name := c.Param("name")

	s.apps.Lock()

	app, ok := s.apps.Get(name)
	if !ok {
		s.apps.Unlock() // Unlock before returning
		s.logger.Warn("Manual sync requested for non-existent application", zap.String("name", name))
		return echo.NewHTTPError(http.StatusNotFound, "Application not found")
	}

	// This is a placeholder for triggering an *immediate* sync.
	// In a real controller, you'd send a signal/message to the specific
	// application's goroutine to wake it up and perform a sync now.
	// For MVP Phase 2, we'll simply update the status to "Syncing" and
	// rely on the next polling interval, or add a simple manual trigger to the controller.
	// Given the current controller structure, making it truly on-demand requires
	// adding a channel to each reconcileApp goroutine.
	// For now, let's update status and log that a sync was *requested*.

	app.Status = "SyncRequested"
	app.Message = "Manual sync requested."
	s.apps.Unlock()
	// No need to save to disk here, controller's next loop or signal will handle it.

	// Placeholder: In a real system, send a signal to the specific application's goroutine
	// For now, the existing polling loop will pick it up on its next interval.
	s.logger.Info("Manual sync requested for application", zap.String("name", name))
	return c.JSON(http.StatusAccepted, SyncTriggerResponse{
		Message: "Manual sync requested. The controller will process it shortly.",
		Status:  "SyncRequested",
	})
}

// HealthCheck is a simple endpoint to check if the API server is running.
//
// It responds with a 200 OK status and a simple message.
// This is useful for monitoring and health checks in production environments.
func (s *Server) healthCheck(c echo.Context) error {
	// Simple health check: just respond with 200 OK
	return c.String(http.StatusOK, "OK")
}
