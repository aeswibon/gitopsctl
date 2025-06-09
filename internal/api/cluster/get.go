package cluster

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Get retrieves the details of a specific Kubernetes cluster by name.
// It returns a Response object containing the cluster's details.
// If the cluster does not exist, it returns a 404 Not Found error.
func (h *Handler) Get(c echo.Context) error {
	name := c.Param("name")

	h.clusters.RLock()
	defer h.clusters.RUnlock()

	cl, ok := h.clusters.Get(name)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "Cluster not found")
	}
	return c.JSON(http.StatusOK, ConvertToResponse(cl))
}
