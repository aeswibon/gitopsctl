package app

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// Sync handles manual sync requests for an application.
// It updates the application's status to "SyncRequested" and logs the request.
// This is a placeholder for triggering an immediate sync, which would typically involve signaling the controller
// to wake up the specific application's goroutine and perform a sync now.
func (h *Handler) Sync(c echo.Context) error {
	name := c.Param("name")

	h.apps.Lock()
	defer h.apps.Unlock()

	app, ok := h.apps.Get(name)
	if !ok {
		h.logger.Warn("Manual sync requested for non-existent application", zap.String("name", name))
		return echo.NewHTTPError(http.StatusNotFound, "Application not found")
	}

	h.controller.TriggerSync(name)

	app.Status = "SyncRequested"
	app.Message = "Manual sync requested."
	// No need to save to disk here, controller's next loop or signal will handle it.
	h.logger.Info("Manual sync requested for application", zap.String("name", name))
	return c.JSON(http.StatusAccepted, SyncTriggerResponse{
		Message: "Manual sync requested. The controller will process it shortly.",
		Status:  "SyncRequested",
	})
}
