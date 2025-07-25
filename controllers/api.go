package controllers

import (
	"costrict-host/internal/config"
	"costrict-host/services"

	"github.com/gin-gonic/gin"
)

type APIController struct {
	svc *services.ServiceManager
}

/**
 * Create new API controller instance
 * @returns {*APIController} New API controller instance
 * @description
 * - Initializes controller with application configuration
 * - Used to manage API routes and handlers
 */
func NewAPIController(svc *services.ServiceManager) *APIController {
	return &APIController{
		svc: svc,
	}
}

/**
 * Register all API routes to Gin engine
 * @param {*gin.Engine} r - Gin router instance
 * @description
 * - Creates /api route group
 * - Registers routes for:
 *   - Service management (list/restart)
 *   - Component management (list/upgrade)
 *   - Endpoint listing
 */
func (a *APIController) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// 服务管理接口
		api.GET("/services", a.ListServices)
		api.POST("/services/:name/restart", a.RestartService)

		// 组件管理接口
		api.GET("/components", a.ListComponents)
		api.POST("/components/:name/upgrade", a.UpgradeComponent)

		// 服务地址管理接口
		api.GET("/endpoints", a.ListEndpoints)
	}
}

// @Summary 获取服务列表
// @Description 获取当前管理的所有服务信息
// @Tags Services
// @Produce json
// @Success 200 {array} config.ServiceConfig
// @Router /api/services [get]
func (a *APIController) ListServices(c *gin.Context) {
	c.JSON(200, a.svc.GetServices())
}

// @Summary 重启服务
// @Description 根据服务名重启指定服务
// @Tags Services
// @Param name path string true "服务名称"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/services/{name}/restart [post]
func (a *APIController) RestartService(c *gin.Context) {
	name := c.Param("name")

	if err := a.svc.RestartService(c.Request.Context(), name); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

// @Summary 获取组件列表
// @Description 获取所有已安装组件信息
// @Tags Components
// @Produce json
// @Success 200 {array} config.ComponentInfo
// @Router /api/components [get]
func (a *APIController) ListComponents(c *gin.Context) {
	components, err := a.svc.GetComponents()
	if err != nil {
		c.JSON(500, gin.H{
			"code":    "component.list_failed",
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, components)
}

// @Summary 升级组件
// @Description 升级指定组件到最新版本
// @Tags Components
// @Param name path string true "组件名称"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string "{"code": "component.not_found", "message": "Component not found"}"
// @Router /api/components/{name}/upgrade [post]
func (a *APIController) UpgradeComponent(c *gin.Context) {
	name := c.Param("name")
	if err := a.svc.UpgradeComponent(name); err != nil {
		if err == config.ErrComponentNotFound {
			c.JSON(404, gin.H{
				"code":    "component.not_found",
				"message": "Component not found",
			})
		} else {
			c.JSON(500, gin.H{
				"code":    "component.upgrade_failed",
				"message": err.Error(),
			})
		}
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

// @Summary 获取服务地址
// @Description 获取所有服务的访问地址
// @Tags Endpoints
// @Produce json
// @Success 200 {array} config.ServiceEndpoint
// @Router /api/endpoints [get]
func (a *APIController) ListEndpoints(c *gin.Context) {
	c.JSON(200, a.svc.GetEndpoints())
}
