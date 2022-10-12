package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/flowchartsman/swaggerui"
	"github.com/labstack/echo/v4"
	"go.withmatt.com/httpheaders"
)

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
	c.Response().Header().Set(httpheaders.CacheControl, "no-cache")
	resp := &LivenessResponse{RespondedAt: time.Now()}
	return c.JSON(http.StatusOK, resp)
}

// Ready returns 200 OK if the application is ready to serve new requests.
func Ready(c echo.Context) error {
	c.Response().Header().Set(httpheaders.CacheControl, "no-cache")
	resp := &ReadinessResponse{RespondedAt: time.Now()}
	return c.JSON(http.StatusOK, resp)
}

// Started returns 200 OK once the application is started.
func Started(c echo.Context) error {
	c.Response().Header().Set(httpheaders.CacheControl, "no-cache")
	resp := &StartedResponse{RespondedAt: time.Now()}
	return c.JSON(http.StatusOK, resp)
}

func Index(c echo.Context) error {
	c.Response().Header().Set(httpheaders.CacheControl, "no-cache")
	return c.Render(http.StatusOK, "index.html.p2", nil)
}
