package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.withmatt.com/httpheaders"
)

func StaticGet(root fs.FS, mimeType string) echo.HandlerFunc {
	return func(c echo.Context) error {
		urlPath := strings.TrimLeft(c.Request().URL.Path, "/")
		fdata, err := root.Open(urlPath)
		if err != nil {
			return c.HTML(http.StatusNotFound, "Not Found")
		}

		st, err := fdata.Stat()
		if err != nil {
			return c.HTML(http.StatusInternalServerError, "Internal Server Error")
		}

		c.Response().Header().Set(httpheaders.ContentLength, fmt.Sprintf("%v", st.Size()))
		c.Response().Header().Set(httpheaders.LastModified, st.ModTime().UTC().Format(time.RFC1123))

		return c.Stream(http.StatusOK, mimeType, fdata)
	}
}

func StaticHead(root fs.FS, mimeType string) echo.HandlerFunc {
	return func(c echo.Context) error {
		urlPath := strings.TrimLeft(c.Request().URL.Path, "/")
		fdata, err := root.Open(urlPath)
		if err != nil {
			return c.HTML(http.StatusNotFound, "Not Found")
		}

		st, err := fdata.Stat()
		if err != nil {
			return c.HTML(http.StatusInternalServerError, "Internal Server Error")
		}

		c.Response().Header().Set(httpheaders.ContentLength, fmt.Sprintf("%v", st.Size()))
		c.Response().Header().Set(httpheaders.LastModified, st.ModTime().UTC().Format(time.RFC1123))
		return c.NoContent(http.StatusOK)
	}
}
