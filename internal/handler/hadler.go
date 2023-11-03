package handler

import (
	"github.com/DwarfWizzard/stat-bot-backend/internal/service"
	"github.com/labstack/echo/v4"
)

type handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *handler {
	return &handler{
		svc: svc,
	}
}

func (h *handler) InitRoutes() *echo.Echo {
	router := echo.New()
	router.Debug = true
	//router.HTTPErrorHandler = h.HTTPErrorHandler
	router.HideBanner = true
	router.HidePort = true

	rtApi := router.Group("/api") //TODO: add middleware
	{
		rtApi.GET("/metrics", h.svc.CollectMetrics)
		rtRecovery := rtApi.Group("/recovery")
		{
			rtRecovery.GET("/terminate/:pid", h.svc.TerminateConn)
			//rtRecovery.Get("/restart/:dbname", h.svc.RestartDatabase)
		}

		//rtApi.GET("/initarmagedon", h.svc.Armagedon) //it will be removed
	}

	return router
}