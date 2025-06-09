package app

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Get retrieves the details of a specific application by name.
// It returns a Response object containing the app's details.
// If the app does not exist, it returns a 404 Not Found error
func (h *Handler) Get(c echo.Context) error {
	name := c.Param("name")

	h.apps.RLock()
	defer h.apps.RUnlock()

	app, ok := h.apps.Get(name)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "Application not found")
	}
	return c.JSON(http.StatusOK, ConvertToResponse(app))
}
