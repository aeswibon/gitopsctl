package cluster

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// List handles the retrieval of all registered Kubernetes clusters.
// It returns a list of Response objects containing the details of each cluster.
func (h *Handler) List(c echo.Context) error {
	h.clusters.RLock()
	defer h.clusters.RUnlock()

	var responses []Response
	for _, cl := range h.clusters.List() {
		responses = append(responses, ConvertToResponse(cl))
	}
	return c.JSON(http.StatusOK, responses)
}
