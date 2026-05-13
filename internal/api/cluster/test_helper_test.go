package cluster

import (
	"net/http/httptest"
	"strings"

	"aeswibon.com/github/gitopsctl/internal/controller"
	appcore "aeswibon.com/github/gitopsctl/internal/core/app"
	clustercore "aeswibon.com/github/gitopsctl/internal/core/cluster"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type testValidator struct {
	validator *validator.Validate
}

func (cv *testValidator) Validate(i any) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(400, err.Error())
	}
	return nil
}

func newTestHandler() (*Handler, *echo.Echo, *appcore.Applications, *clustercore.Clusters) {
	e := echo.New()
	v := validator.New()
	_ = v.RegisterValidation("kubeconfigfile", func(fl validator.FieldLevel) bool {
		return true // skip real file check in tests
	})
	e.Validator = &testValidator{validator: v}
	apps := appcore.NewApplications()
	clusters := clustercore.NewClusters()
	ctrl := controller.NewController(zap.NewNop(), apps, clusters)
	h := NewHandler(zap.NewNop(), clusters, apps, ctrl)
	return h, e, apps, clusters
}

func newJSONContext(e *echo.Echo, method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}
