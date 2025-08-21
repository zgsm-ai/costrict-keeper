package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"costrict-keeper/internal/models"
	"costrict-keeper/services"

	"github.com/gin-gonic/gin"
)

// TunnelController handles tunnel-related HTTP requests
type TunnelController struct {
	tunnelService *services.TunnelManager
}

// NewTunnelController creates a new TunnelController with initialized tunnel service
func NewTunnelController() *TunnelController {
	return &TunnelController{
		tunnelService: services.GetTunnelManager(),
	}
}

// CreateTunnel creates reverse tunnel for application
//
//	@Summary		Create reverse tunnel
//	@Description	Create reverse tunnel for specified application
//	@Tags			Tunnels
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.CreateTunnelRequest	true	"Create tunnel request parameters"
//	@Success		200		{object}	models.Tunnel		"Tunnel  creation"
//	@Failure		400		{object}	models.ErrorResponse		"Invalid parameter error response"
//	@Failure		500		{object}	models.ErrorResponse		"Tunnel creation failure error response"
//	@Router			/costrict/api/v1/tunnels [post]
func (tc *TunnelController) CreateTunnel(c *gin.Context) {
	var req models.CreateTunnelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, &models.ErrorResponse{
			Error: "Invalid request parameters",
		})
		return
	}

	tun, err := tc.tunnelService.StartTunnel(req.AppName, req.Port)
	if err != nil {
		c.JSON(http.StatusInternalServerError, &models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tun.Tunnel)
}

// DeleteTunnel closes application's reverse tunnel
//
//	@Summary		Close reverse tunnel
//	@Description	Close reverse tunnel for specified application
//	@Tags			Tunnels
//	@Accept			json
//	@Produce		json
//	@Param			app		path		string						true	"Application name"
//	@Param			port	path		string						true	"Port number"
//	@Success		200		{object}	models.TunnelResponse		"Tunnel close success response"
//	@Failure		400		{object}	models.ErrorResponse		"Invalid parameter error response"
//	@Failure		500		{object}	models.ErrorResponse		"Tunnel close failure error response"
//	@Router			/costrict/api/v1/tunnels/{app}/{port} [delete]
func (tc *TunnelController) DeleteTunnel(c *gin.Context) {
	var req models.DeleteTunnelRequest
	req.AppName = c.Param("app")

	// Convert port parameter from string to int
	portStr := c.Param("port")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, &models.ErrorResponse{
			Error: "Invalid port parameter",
		})
		return
	}

	if err := tc.tunnelService.CloseTunnel(req.AppName, port); err != nil {
		c.JSON(http.StatusInternalServerError, &models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, &models.TunnelResponse{
		AppName: req.AppName,
		Status:  "success",
		Message: fmt.Sprintf("Successfully closed tunnel for app %s", req.AppName),
	})
}

// ListTunnels lists all active tunnels
//
//	@Summary		List all tunnels
//	@Description	Get list of all active tunnels
//	@Tags			Tunnels
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		models.Tunnel			"Tunnel list response"
//	@Failure		500	{object}	models.ErrorResponse	"Internal server error response"
//	@Router			/costrict/api/v1/tunnels [get]
func (tc *TunnelController) ListTunnels(c *gin.Context) {
	tunnels := tc.tunnelService.ListTunnels()

	c.JSON(http.StatusOK, tunnels)
}

// GetTunnelInfo gets details of specific tunnel
//
//	@Summary		Get tunnel info
//	@Description	Get details of specified tunnel
//	@Tags			Tunnels
//	@Accept			json
//	@Produce		json
//	@Param			app	path		string					true	"Application name"
//	@Param			port	path		int						true	"Port number"
//	@Success		200	{object}	models.Tunnel			"Tunnel details response"
//	@Failure		404	{object}	models.ErrorResponse	"Tunnel not found error response"
//	@Failure		500	{object}	models.ErrorResponse	"Internal server error response"
//	@Router			/costrict/api/v1/tunnels/{app}/{port} [get]
func (tc *TunnelController) GetTunnelInfo(c *gin.Context) {
	appName := c.Param("app")
	portStr := c.Param("port")

	// Convert port parameter from string to int
	port, err := strconv.Atoi(portStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, &models.ErrorResponse{
			Error: "Invalid port parameter",
		})
		return
	}

	tunnel, err := tc.tunnelService.GetTunnelInfo(appName, port)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}

		c.JSON(status, &models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tunnel)
}

/**
* Register all tunnel-related routes to Gin engine
* @param {*gin.Engine} r - Gin router instance
* @description
* - Creates /costrict/api/v1 route group
* - Registers routes for:
*   - Create tunnel (POST /tunnels)
*   - List all tunnels (GET /tunnels)
*   - Get specific tunnel info (GET /tunnels/{app}/{port})
*   - Delete tunnel (DELETE /tunnels/{app}/{port})
* @example
* router := gin.Default()
* tunnelController, err := NewTunnelController()
* if err != nil {
*     log.Fatal(err)
* }
* tunnelController.RegisterRoutes(router)
 */
func (tc *TunnelController) RegisterRoutes(r *gin.Engine) {
	tunnelAPI := r.Group("/costrict/api/v1")
	{
		// 隧道管理接口
		tunnels := tunnelAPI.Group("/tunnels")
		{
			tunnels.POST("", tc.CreateTunnel)
			tunnels.GET("", tc.ListTunnels)
			tunnels.GET("/:app/:port", tc.GetTunnelInfo)
			tunnels.DELETE("/:app/:port", tc.DeleteTunnel)
		}
	}
}
