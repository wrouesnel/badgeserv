package server

import (
	"fmt"
	"github.com/brpaz/echozap"
	"github.com/flowchartsman/swaggerui"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/wrouesnel/badgeserv/api/v1"
	"github.com/wrouesnel/badgeserv/version"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
	"time"
)

type ApiServerConfig struct {
	Prefix string `help:"Prefix the API is bing served under, if any"`
	Host   string `help:"Host the API should be served on" default:""`
	Port   int    `help:"Port to serve on" default:"8080"`
}

var (
	ErrApiInitializationFailed = errors.New("API failed to initialize")
)

// Api launches an ApiV1 instance server and manages it's lifecycle.
func Api(serverConfig ApiServerConfig) error {
	logger := zap.L()
	logger.Info("Starting API server")
	// Create the API
	apiConfig := &api.ApiConfig{}
	apiInstance, prefix := api.NewApi(apiConfig)

	if apiInstance == nil {
		err := ErrApiInitializationFailed
		logger.Error("API failed to initialize", zap.Error(err))
		return ErrApiInitializationFailed
	}

	// Start the API
	if err := ApiServer(serverConfig, apiInstance, prefix); err != nil {
		logger.Error("Error from server", zap.Error(err))
		return errors.Wrap(err, "Server exiting with error")
	}

	return nil
}

// ApiServer implements the logic necessary to launch an API from a server config and a server.
// The primary difference to Api() is that the apInstance interface is explicitly passed.
func ApiServer[T api.ServerInterface](serverConfig ApiServerConfig, apiInstance T, prefix string) error {
	var logger = zap.L().With(zap.String("subsystem", "server"))

	e := echo.New()
	e.HideBanner = true
	e.Logger.SetOutput(io.Discard)

	// Setup Prometheus monitoring
	p := prometheus.NewPrometheus(version.Name, nil)
	p.Use(e)

	// Setup logging
	e.Use(echozap.ZapLogger(zap.L()))

	apiPrefix := fmt.Sprintf("%s/api/%s", serverConfig.Prefix, prefix)
	logger.Info("Initializing API with prefix",
		zap.String("configured_prefix", serverConfig.Prefix),
		zap.String("api_prefix", prefix),
		zap.String("api_basepath", apiPrefix))

	api.RegisterHandlersWithBaseURL(e, apiInstance, apiPrefix)
	// Add the Swagger API as the frontend.
	uiPrefix := fmt.Sprintf("%s/ui", apiPrefix)
	uiHandler := EchoSwaggerUIHandler(uiPrefix, api.OpenApiSpec)
	e.GET(fmt.Sprintf("%s", uiPrefix), uiHandler)
	e.GET(fmt.Sprintf("%s/*", uiPrefix), uiHandler)
	logger.Info("Swagger UI configured prefix", zap.String("ui_path", uiPrefix))

	// Add ready and liveness endpoints
	e.GET("/-/ready", Ready)
	e.GET("/-/live", Live)
	e.GET("/-/started", Live)

	err := e.Start(fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port))
	return err
}

func EchoSwaggerUIHandler(uiPath string, swaggerUISpec []byte) echo.HandlerFunc {
	uiPath = strings.TrimRight(uiPath, "/")
	uiPathWithSlash := fmt.Sprintf("%s/", uiPath)
	handler := http.StripPrefix(uiPath, swaggerui.Handler(swaggerUISpec))
	return func(c echo.Context) error {
		request := c.Request()
		// The Swagger UI handler returns / redirect if it receives an empty
		// string. We need to handle this case ourselves or it just redirects
		// to the root of the application.
		if request.URL.Path == uiPath {
			return c.Redirect(http.StatusMovedPermanently, uiPathWithSlash)
		}

		handler.ServeHTTP(c.Response(), request)
		return nil
	}
}

// Live returns 200 OK if the application server is still functional and able
// to handle requests.
func Live(c echo.Context) error {
	resp := &LivenessResponse{RespondedAt: time.Now()}
	return c.JSON(http.StatusOK, resp)
}

// Ready returns 200 OK if the application is ready to serve new requests.
func Ready(c echo.Context) error {
	resp := &ReadinessResponse{RespondedAt: time.Now()}
	return c.JSON(http.StatusOK, resp)
}

// Started returns 200 OK once the application is started.
func Started(c echo.Context) error {
	resp := &StartedResponse{RespondedAt: time.Now()}
	return c.JSON(http.StatusOK, resp)
}
