package app

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// List handles the retrieval of all registered applications.
// It returns a list of Response objects containing the details of each application.
func (h *Handler) List(c echo.Context) error {
	h.apps.RLock()
	defer h.apps.RUnlock()

	var responses []Response
	for _, app := range h.apps.List() {
		responses = append(responses, ConvertToResponse(app))
	}
	return c.JSON(http.StatusOK, responses)
}
