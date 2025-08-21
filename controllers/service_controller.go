package controllers

import (
	"costrict-keeper/services"

	"github.com/gin-gonic/gin"
)

type ServiceController struct {
	service *services.ServiceManager
}

/**
 * Create new Service controller instance
 * @param {*services.ServiceManager} service - Service manager instance for managing services
 * @returns {*ServiceController} New Service controller instance
 * @description
 * - Initializes controller with service manager
 * - Used to manage API routes and handlers for service operations
 * @example
 * svcManager := services.GetServiceManager()
 * controller := controllers.NewServiceController(svcManager)
 */
func NewServiceController(service *services.ServiceManager) *ServiceController {
	return &ServiceController{
		service: service,
	}
}

/**
 * Register all service API routes to Gin router group
 * @param {*gin.RouterGroup} r - Gin router group instance
 * @description
 * - Registers routes for:
 *   - Service management (list/start/stop/restart/get)
 * @example
 * api := router.Group("/costrict/api/v1")
 * controller := NewServiceController(svcManager)
 * controller.RegisterRoutes(api)
 */
func (s *ServiceController) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/costrict/api/v1")
	// 服务管理接口
	api.GET("/services", s.ListServices)
	api.POST("/services/:name/start", s.StartService)
	api.POST("/services/:name/stop", s.StopService)
	api.POST("/services/:name/restart", s.RestartService)
	api.GET("/services/:name", s.GetService)
}

// ListServices lists all managed services
//
//	@Summary		List all services
//	@Description	Get list of all managed services with their current status
//	@Tags			Services
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		services.ServiceInstance	"List of service instances"
//	@Failure		500	{object}	models.ErrorResponse		"Internal server error response"
//	@Router			/costrict/api/v1/services [get]
func (s *ServiceController) ListServices(c *gin.Context) {
	c.JSON(200, s.service.GetInstances())
}

// RestartService restarts a specific service by name
//
//	@Summary		Restart service
//	@Description	Restart a specific service by its name
//	@Tags			Services
//	@Accept			json
//	@Produce		json
//	@Param			name	path		string					true	"Service name"
//	@Success		200		{object}	map[string]interface{}	"Service restart success response"
//	@Failure		404		{object}	models.ErrorResponse	"Service not found error response"
//	@Failure		500		{object}	models.ErrorResponse	"Internal server error response"
//	@Router			/costrict/api/v1/services/{name}/restart [post]
func (s *ServiceController) RestartService(c *gin.Context) {
	name := c.Param("name")

	if err := s.service.RestartService(c.Request.Context(), name); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

// StartService starts a specific service by name
//
//	@Summary		Start service
//	@Description	Start a specific service by its name
//	@Tags			Services
//	@Accept			json
//	@Produce		json
//	@Param			name	path		string					true	"Service name"
//	@Success		200		{object}	map[string]interface{}	"Service start success response"
//	@Failure		404		{object}	models.ErrorResponse	"Service not found error response"
//	@Failure		500		{object}	models.ErrorResponse	"Internal server error response"
//	@Router			/costrict/api/v1/services/{name}/start [post]
func (s *ServiceController) StartService(c *gin.Context) {
	name := c.Param("name")

	if err := s.service.StartService(c.Request.Context(), name); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

// StopService stops a specific service by name
//
//	@Summary		Stop service
//	@Description	Stop a specific service by its name
//	@Tags			Services
//	@Accept			json
//	@Produce		json
//	@Param			name	path		string					true	"Service name"
//	@Success		200		{object}	map[string]interface{}	"Service stop success response"
//	@Failure		404		{object}	models.ErrorResponse	"Service not found error response"
//	@Failure		500		{object}	models.ErrorResponse	"Internal server error response"
//	@Router			/costrict/api/v1/services/{name}/stop [post]
func (s *ServiceController) StopService(c *gin.Context) {
	name := c.Param("name")

	if err := s.service.StopService(name); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

// GetService gets detailed information of a specific service by name
//
//	@Summary		Get service information
//	@Description	Get detailed information of a specific service by its name
//	@Tags			Services
//	@Accept			json
//	@Produce		json
//	@Param			name	path		string					true	"Service name"
//	@Success		200		{object}	services.ServiceDetail	"Service detail information"
//	@Failure		404		{object}	models.ErrorResponse	"Service not found error response"
//	@Failure		500		{object}	models.ErrorResponse	"Internal server error response"
//	@Router			/costrict/api/v1/services/{name} [get]
func (s *ServiceController) GetService(c *gin.Context) {
	name := c.Param("name")

	svc := s.service.GetInstance(name)
	if svc != nil {
		c.JSON(200, s.service.GetServiceDetail(svc))
		return
	}

	c.JSON(404, gin.H{"error": "service not found"})
}
