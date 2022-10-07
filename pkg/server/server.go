package server

import (
	"fmt"
	"github.com/brpaz/echozap"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/wrouesnel/badgeserv/api/v1"
	"github.com/wrouesnel/badgeserv/version"
	"go.uber.org/zap"
	"io"
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
	apiInstance, apiPrefix := api.NewApi(apiConfig)

	if apiInstance == nil {
		err := ErrApiInitializationFailed
		logger.Error("API failed to initialize", zap.Error(err))
		return ErrApiInitializationFailed
	}

	// Start the API
	if err := Server(serverConfig, ApiConfigure(serverConfig, apiInstance, apiPrefix)); err != nil {
		logger.Error("Error from server", zap.Error(err))
		return errors.Wrap(err, "Server exiting with error")
	}

	return nil
}

// ApiConfigure implements the logic necessary to launch an API from a server config and a server.
// The primary difference to Api() is that the apInstance interface is explicitly passed.
func ApiConfigure[T api.ServerInterface](serverConfig ApiServerConfig, apiInstance T, apiPrefix string) func(e *echo.Echo) error {
	return func(e *echo.Echo) error {
		var logger = zap.L().With(zap.String("subsystem", "server"))

		fullApiPrefix := fmt.Sprintf("%s/api/%s", serverConfig.Prefix, apiPrefix)
		logger.Info("Initializing API with apiPrefix",
			zap.String("configured_prefix", serverConfig.Prefix),
			zap.String("api_prefix", apiPrefix),
			zap.String("api_basepath", fullApiPrefix))

		api.RegisterHandlersWithBaseURL(e, apiInstance, fullApiPrefix)
		// Add the Swagger API as the frontend.
		uiPrefix := fmt.Sprintf("%s/ui", fullApiPrefix)
		uiHandler := EchoSwaggerUIHandler(uiPrefix, api.OpenApiSpec)
		e.GET(fmt.Sprintf("%s", uiPrefix), uiHandler)
		e.GET(fmt.Sprintf("%s/*", uiPrefix), uiHandler)
		logger.Info("Swagger UI configured apiPrefix", zap.String("ui_path", uiPrefix))

		return nil
	}
}

// Server configures and starts an Echo server with standard capabilities, and configuration functions.
func Server(serverConfig ApiServerConfig, ConfigFns ...func(e *echo.Echo) error) error {
	logger := zap.L().With(zap.String("subsystem", "server"))

	e := echo.New()
	e.HideBanner = true
	e.Logger.SetOutput(io.Discard)

	// Setup Prometheus monitoring
	p := prometheus.NewPrometheus(version.Name, nil)
	p.Use(e)

	// Setup logging
	e.Use(echozap.ZapLogger(zap.L()))

	// Add ready and liveness endpoints
	e.GET("/-/ready", Ready)
	e.GET("/-/live", Live)
	e.GET("/-/started", Live)

	for _, configFn := range ConfigFns {
		if err := configFn(e); err != nil {
			logger.Error("Failed calling configuration function", zap.Error(err))
		}
	}

	err := e.Start(fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port))
	return err
}
