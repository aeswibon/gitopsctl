package app

import (
	"aeswibon.com/github/gitopsctl/internal/controller"
	appcore "aeswibon.com/github/gitopsctl/internal/core/app"
	clustercore "aeswibon.com/github/gitopsctl/internal/core/cluster"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// Handler handles application-related HTTP requests.
type Handler struct {
	logger     *zap.Logger
	apps       *appcore.Applications
	clusters   *clustercore.Clusters
	controller *controller.Controller
}

// NewHandler creates a new application handler.
func NewHandler(logger *zap.Logger, apps *appcore.Applications, clusters *clustercore.Clusters, controller *controller.Controller) *Handler {
	return &Handler{
		logger:     logger,
		apps:       apps,
		clusters:   clusters,
		controller: controller,
	}
}

// RegisterRoutes registers all application-related routes.
func RegisterRoutes(g *echo.Group, handler *Handler) {
	// Applications Management
	g.POST("/applications", handler.Register)
	g.GET("/applications", handler.List)
	g.GET("/applications/:name", handler.Get)
	g.DELETE("/applications/:name", handler.Unregister)
	g.POST("/applications/:name/sync", handler.Sync)
}
