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
	e := echo.New()
	e.HideBanner = true
	e.Logger.SetOutput(io.Discard)

	// Setup Prometheus monitoring
	p := prometheus.NewPrometheus(version.Name, nil)
	p.Use(e)

	// Setup logging
	e.Use(echozap.ZapLogger(zap.L()))

	apiPrefix := fmt.Sprintf("%s/api/%s", serverConfig.Prefix, prefix)

	api.RegisterHandlersWithBaseURL(e, apiInstance, apiPrefix)
	// Add the Swagger API as the frontend.
	e.GET("/*", echo.WrapHandler(swaggerui.Handler(api.OpenApiSpec)))

	err := e.Start(fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port))
	return err
}
