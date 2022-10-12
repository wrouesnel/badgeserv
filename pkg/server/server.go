package server

import (
	"fmt"
	"github.com/brpaz/echozap"
	"github.com/flosch/pongo2/v6"
	"github.com/go-resty/resty/v2"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/wrouesnel/badgeserv/api/v1"
	"github.com/wrouesnel/badgeserv/assets"
	"github.com/wrouesnel/badgeserv/pkg/badges"
	"github.com/wrouesnel/badgeserv/pkg/pongorenderer"
	"github.com/wrouesnel/badgeserv/pkg/server/badgeconfig"
	"github.com/wrouesnel/badgeserv/version"
	"go.uber.org/zap"
	"go.withmatt.com/httpheaders"
	"io"
	"io/fs"
	"net/url"
	"time"
)

// ApiServerConfig configures local hosting parameters of the API server
type ApiServerConfig struct {
	Prefix string `help:"Prefix the API is bing served under, if any"`
	Host   string `help:"Host the API should be served on" default:""`
	Port   int    `help:"Port to serve on" default:"8080"`

	HttpClient ApiHttpClientConfig `embed:"" prefix:"http"`
}

// ApiHttpClientConfig configures the outbound HTTP request globals
type ApiHttpClientConfig struct {
	Timeout   time.Duration `help:"Default HTTP request timeout" default:"3s"`
	UserAgent string        `help:"User Agent string to send with requests" default:""`
}

var (
	ErrApiInitializationFailed = errors.New("API failed to initialize")
)

// Api launches an ApiV1 instance server and manages it's lifecycle.
func Api(serverConfig ApiServerConfig, badgeConfig badges.BadgeConfig, assetConfig assets.AssetsConfig, badgeConfigDir string) error {
	logger := zap.L()

	var predefinedBadgeConfig *badgeconfig.Config
	if badgeConfigDir != "" {
		logger.Info("Loading predefined badge configs")
		var err error
		predefinedBadgeConfig, err = badgeconfig.LoadDir(badgeConfigDir)
		if err != nil {
			logger.Error("Fatal error loading predefined badge configuration")
			return errors.Wrap(err, "badgeconfig")
		}
	} else {
		logger.Info("No predefined badge configs")
		predefinedBadgeConfig = &badgeconfig.Config{PredefinedBadges: map[string]badgeconfig.BadgeDefinition{}}
	}

	logger.Debug("Configuring API REST client")
	httpClient := resty.New()
	if serverConfig.HttpClient.UserAgent == "" {
		httpClient.SetHeader(httpheaders.UserAgent, fmt.Sprintf("%s/%s", version.Name, version.Version))
	} else {
		httpClient.SetHeader(httpheaders.UserAgent, serverConfig.HttpClient.UserAgent)
	}
	httpClient.SetTimeout(serverConfig.HttpClient.Timeout)

	badgeService := badges.NewBadgeService(&badgeConfig)

	logger.Debug("Creating API config")
	apiConfig := &api.ApiConfig{
		badgeService,
		httpClient,
		predefinedBadgeConfig,
	}
	apiInstance, apiPrefix := api.NewApi(apiConfig)

	if apiInstance == nil {
		err := ErrApiInitializationFailed
		logger.Error("API failed to initialize", zap.Error(err))
		return ErrApiInitializationFailed
	}

	templateGlobals := make(pongo2.Context)
	templateGlobals["ApiVersionPrefix"] = apiPrefix
	templateGlobals["Version"] = map[string]string{
		"Version":     version.Version,
		"Name":        version.Name,
		"Description": version.Description,
	}
	templateGlobals["Colors"] = badgeService.Colors
	templateGlobals["PredefinedBadges"] = lo.MapToSlice(predefinedBadgeConfig.PredefinedBadges, func(k string, v badgeconfig.BadgeDefinition) interface{} {
		exampleUrl := url.URL{Path: fmt.Sprintf("predefined/%s/", k)}
		qry := exampleUrl.Query()
		for k, v := range v.Example {
			qry.Set(k, v)
		}
		exampleUrl.RawQuery = qry.Encode()

		return struct {
			Name       string
			ExampleURL string
			badgeconfig.BadgeDefinition
		}{
			Name:            k,
			ExampleURL:      exampleUrl.String(),
			BadgeDefinition: v,
		}
	})

	logger.Info("Starting API server")
	if err := Server(serverConfig, assetConfig, templateGlobals, ApiConfigure(serverConfig, apiInstance, apiPrefix)); err != nil {
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
func Server(serverConfig ApiServerConfig, assetConfig assets.AssetsConfig, templateGlobals pongo2.Context, ConfigFns ...func(e *echo.Echo) error) error {
	logger := zap.L().With(zap.String("subsystem", "server"))

	e := echo.New()
	e.HideBanner = true
	e.Logger.SetOutput(io.Discard)

	// Configure main renderer to use pongo2
	webAssets := lo.Must(fs.Sub(assets.Assets(), "web"))
	webTemplateSet := pongo2.NewSet("web", pongo2.NewFSLoader(webAssets))
	webTemplateSet.Debug = assetConfig.DebugTemplates
	webTemplateSet.Globals = templateGlobals
	e.Renderer = pongorenderer.NewRenderer(webTemplateSet)

	// Setup Prometheus monitoring
	p := prometheus.NewPrometheus(version.Name, nil)
	p.Use(e)

	// Setup logging
	e.Use(echozap.ZapLogger(zap.L()))

	// Add ready and liveness endpoints
	e.GET("/-/ready", Ready)
	e.GET("/-/live", Live)
	e.GET("/-/started", Started)

	// Add static hosting endpoints
	e.GET("/", Index)

	e.GET("/css/*", StaticGet(webAssets, "text/css"))
	e.HEAD("/css/*", StaticHead(webAssets, "text/css"))

	e.GET("/js/*", StaticGet(webAssets, "application/javascript"))
	e.HEAD("/js/*", StaticHead(webAssets, "application/javascript"))

	//	e.GET("/css/*", Static(webAssets, "text/css"))
	//e.GET("/js/*", Static(webAssets, "application/javascript"))

	for _, configFn := range ConfigFns {
		if err := configFn(e); err != nil {
			logger.Error("Failed calling configuration function", zap.Error(err))
		}
	}

	err := e.Start(fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port))
	return err
}
