package api

import (
	"crowdstrike-takehome-nmap/pkg/hostnamevalidate"
	"crowdstrike-takehome-nmap/pkg/scanscheduler"
	"crowdstrike-takehome-nmap/pkg/scanscheduler/scandiff"
	"errors"
	"github.com/Ullaakut/nmap"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

//go:generate bash -c "oapi-codegen -package api openapi.yaml > api.gen.go"

// ApiImpl implements the actual nmap-api
type apiImpl struct {
	scheduler scanscheduler.SchedulerClient
	version   string
}

func NewApi(scheduler scanscheduler.SchedulerClient, version string) ServerInterface {
	return &apiImpl{
		scheduler,
		version,
	}
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

func (a *apiImpl) GetResultsIdScanId(ctx echo.Context, scanId string) error {
	data, err := a.scheduler.RequestResultsByScanId(ctx.Request().Context(), scanId)
	if err != nil {
		var notFoundError *scanscheduler.ErrNotFound
		if errors.As(err, &notFoundError) {
			return ctx.JSON(http.StatusNotFound, &ClientError{
				Error:       err.Error(),
				Description: "Requested Scan ID could not be found",
			})
		} else {
			return ctx.JSON(http.StatusInternalServerError, &ClientError{
				Error:       err.Error(),
				Description: "Internal Server Error",
			})
		}
	}

	return ctx.JSON(http.StatusOK, data)
}

func (a *apiImpl) GetResultsHostHost(ctx echo.Context, host string) error {
	if !hostnamevalidate.ValidHostnameOrIp(host) {
		return ctx.JSON(http.StatusBadRequest, &ClientError{
			Error:       "Invalid hostname or IP address supplied",
			Description: "Invalid hostname or IP address supplied",
		})
	}

	summary, err := a.scheduler.RequestResultsByHost(ctx.Request().Context(), host)
	if err != nil {
		var notFoundError *scanscheduler.ErrNotFound
		if errors.As(err, &notFoundError) {
			return ctx.JSON(http.StatusNotFound, &ClientError{
				Error:       err.Error(),
				Description: "Requested host could not be found",
			})
		} else {
			return ctx.JSON(http.StatusInternalServerError, &ClientError{
				Error:       err.Error(),
				Description: "Internal Server Error",
			})
		}
	}

	return ctx.JSON(http.StatusOK, summary)
}

func (a *apiImpl) GetResultsHostHostLatest(ctx echo.Context, host string) error {
	if !hostnamevalidate.ValidHostnameOrIp(host) {
		return ctx.JSON(http.StatusBadRequest, &ClientError{
			Error:       "Invalid hostname or IP address supplied",
			Description: "Invalid hostname or IP address supplied",
		})
	}

	summary, err := a.scheduler.RequestResultsByHost(ctx.Request().Context(), host)
	if err != nil {
		var notFoundError *scanscheduler.ErrNotFound
		if errors.As(err, &notFoundError) {
			return ctx.JSON(http.StatusNotFound, &ClientError{
				Error:       err.Error(),
				Description: "Requested host could not be found",
			})
		} else {
			return ctx.JSON(http.StatusInternalServerError, &ClientError{
				Error:       err.Error(),
				Description: "Internal Server Error",
			})
		}
	}

	if len(summary.ScanHistory) == 0 {
		return ctx.JSON(http.StatusNotFound, &ClientError{
			Error:       "Requested host has no scan results",
			Description: "Requested host has no scan results",
		})
	}

	return a.GetResultsIdScanId(ctx, summary.ScanHistory[0].ScanId)
}

func (a *apiImpl) PostScanHost(ctx echo.Context, host string) error {
	if !hostnamevalidate.ValidHostnameOrIp(host) {
		return ctx.JSON(http.StatusBadRequest, &ClientError{
			Error:       "Invalid hostname or IP address supplied",
			Description: "Invalid hostname or IP address supplied",
		})
	}

	scanId, created, err := a.scheduler.RequestScanByHost(host, false)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, &ClientError{
			Error:       err.Error(),
			Description: "Scan request failed to complete",
		})
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	return ctx.JSON(status, scanId)
}

func (a *apiImpl) GetResultsDiffPortsScanId1ScanId2(ctx echo.Context, scanId1 string, scanId2 string) error {
	scan1Ch := make(chan *nmap.Run)
	scan2Ch := make(chan *nmap.Run)

	go func() {
		scan1, _ := a.scheduler.RequestResultsByScanId(ctx.Request().Context(), scanId1)
		scan1Ch <- scan1
	}()

	go func() {
		scan2, _ := a.scheduler.RequestResultsByScanId(ctx.Request().Context(), scanId2)
		scan2Ch <- scan2
	}()

	scan1 := <-scan1Ch
	scan2 := <-scan2Ch

	if scan1 == nil {
		return ctx.JSON(http.StatusBadRequest, &ClientError{
			Error:       "scan1 parameter is not a valid scanId",
			Description: "Port diff cannot be generated",
		})
	}

	if scan2 == nil {
		return ctx.JSON(http.StatusBadRequest, &ClientError{
			Error:       "scan2 parameter is not a valid scanId",
			Description: "Port diff cannot be generated",
		})
	}

	result := scandiff.DiffPorts(scanId1, scan1, scanId2, scan2)
	if result == nil {
		return ctx.JSON(http.StatusBadRequest, &ClientError{
			Error:       "Port difference failed to generate",
			Description: "Port diff cannot be generated",
		})
	}

	return ctx.JSON(http.StatusOK, result)
}

func (a *apiImpl) GetResultsDiffPortsHostHost(ctx echo.Context, host string) error {
	if !hostnamevalidate.ValidHostnameOrIp(host) {
		return ctx.JSON(http.StatusBadRequest, &ClientError{
			Error:       "Invalid hostname or IP address supplied",
			Description: "Invalid hostname or IP address supplied",
		})
	}

	hostSummary, err := a.scheduler.RequestResultsByHost(ctx.Request().Context(), host)
	if err != nil {
		var notFoundError *scanscheduler.ErrNotFound
		if errors.As(err, &notFoundError) {
			return ctx.JSON(http.StatusNotFound, &ClientError{
				Error:       err.Error(),
				Description: "Requested host could not be found could not be found",
			})
		} else {
			return ctx.JSON(http.StatusInternalServerError, &ClientError{
				Error:       err.Error(),
				Description: "Internal Server Error",
			})
		}
	}

	if len(hostSummary.ScanHistory) < 2 {
		return ctx.JSON(http.StatusNotFound, &ClientError{
			Error:       "There are less then 2 scanIds available for this host",
			Description: "Port diff cannot be generated",
		})
	}

	scan2 := hostSummary.ScanHistory[0]
	scan1 := hostSummary.ScanHistory[1]

	// Cross-call to the other API function for implementation logic.
	return a.GetResultsDiffPortsScanId1ScanId2(ctx, scan1.ScanId, scan2.ScanId)
}
