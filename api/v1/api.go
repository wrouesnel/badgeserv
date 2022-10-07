package api

import (
	"github.com/labstack/echo/v4"
	"github.com/wrouesnel/badgeserv/version"
	"net/http"
	"time"
)

//go:generate bash -c "oapi-codegen -package api openapi.yaml > api.gen.go"

// ApiImpl implements the actual nmap-api
type apiImpl struct {
	version string
}

func (a *apiImpl) GetBadgeDynamic(ctx echo.Context, params GetBadgeDynamicParams) error {
	//TODO implement me
	panic("implement me")
}

func (a *apiImpl) GetBadgePredefined(ctx echo.Context) error {
	//TODO implement me
	panic("implement me")
}

func (a *apiImpl) GetBadgePredefinedPredefinedName(ctx echo.Context, predefinedName string, params GetBadgePredefinedPredefinedNameParams) error {
	//TODO implement me
	panic("implement me")
}

func (a *apiImpl) GetBadgeStatic(ctx echo.Context, params GetBadgeStaticParams) error {
	//TODO implement me
	panic("implement me")
}

// ApiConfig provides the up-front configuration necessary to launch an API
type ApiConfig struct {
}

// NewApi returns the API server instance and the version prefix
func NewApi(apiConfig *ApiConfig) (ServerInterface, string) {
	return &apiImpl{
		version.Version,
	}, "v1"
}

// GetOpenapiYaml implements returning the openapi.yaml file
func (a *apiImpl) GetOpenapiYaml(ctx echo.Context) error {
	header := ctx.Response().Header()
	header.Set("Content-Type", "text/plain")
	header.Set("Content-Disposition", "inline; filename=\"openapi.yaml\"")

	ctx.Response().WriteHeader(http.StatusOK)

	_, err := ctx.Response().Write(OpenApiSpec)
	return err
}

func (a *apiImpl) GetPing(ctx echo.Context) error {
	now := time.Now()
	status := PingResponseStatus("ok")
	return ctx.JSON(http.StatusOK, &PingResponse{
		RespondedAt: &now,
		Status:      &status,
		Version:     &a.version,
	})
}
