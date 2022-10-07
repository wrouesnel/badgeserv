package server

import (
	"fmt"
	"github.com/flowchartsman/swaggerui"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
	"time"
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
