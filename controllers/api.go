package controllers

import (
	"costrict-keeper/internal/config"
	"costrict-keeper/services"

	"github.com/gin-gonic/gin"
)

type APIController struct {
	svc *services.ServiceManager
}

/**
 * Create new API controller instance
 * @param {*services.ServiceManager} svc - Service manager instance for managing services
 * @returns {*APIController} New API controller instance
 * @description
 * - Initializes controller with service manager
 * - Used to manage API routes and handlers for service operations
 * @example
 * svcManager := services.NewServiceManager()
 * controller := controllers.NewAPIController(svcManager)
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
 * @example
 * router := gin.Default()
 * controller := NewAPIController(svcManager)
 * controller.RegisterRoutes(router)
 */
func (a *APIController) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/costrict/api/v1")
	{
		// 服务管理接口
		api.GET("/services", a.ListServices)
		api.POST("/services/:name/start", a.StartService)
		api.POST("/services/:name/stop", a.StopService)
		api.POST("/services/:name/restart", a.RestartService)
		api.GET("/services/:name", a.GetService)

		// 组件管理接口
		api.GET("/components", a.ListComponents)
		api.POST("/components/:name/upgrade", a.UpgradeComponent)
		api.DELETE("/components/:name", a.DeleteComponent)
	}
}

// @Summary 获取服务列表
// @Description 获取当前管理的所有服务信息
// @Tags Services
// @Produce json
// @Success 200 {array} config.ServiceSpecification
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

// @Summary 启动服务
// @Description 根据服务名启动指定服务
// @Tags Services
// @Param name path string true "服务名称"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /costrict/api/v1/services/{name}/start [post]
func (a *APIController) StartService(c *gin.Context) {
	name := c.Param("name")

	if err := a.svc.StartService(c.Request.Context(), name); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

// @Summary 停止服务
// @Description 根据服务名停止指定服务
// @Tags Services
// @Param name path string true "服务名称"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /costrict/api/v1/services/{name}/stop [post]
func (a *APIController) StopService(c *gin.Context) {
	name := c.Param("name")

	if err := a.svc.StopService(name); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

// @Summary 获取服务信息
// @Description 根据服务名获取指定服务的详细信息
// @Tags Services
// @Param name path string true "服务名称"
// @Success 200 {object} config.ServiceSpecification
// @Failure 404 {object} map[string]interface{}
// @Router /costrict/api/v1/services/{name} [get]
func (a *APIController) GetService(c *gin.Context) {
	name := c.Param("name")

	for _, svc := range a.svc.GetServices() {
		if svc.Name == name {
			c.JSON(200, svc)
			return
		}
	}

	c.JSON(404, gin.H{"error": "service not found"})
}

// @Summary 删除组件
// @Description 根据组件名删除指定组件
// @Tags Components
// @Param name path string true "组件名称"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /costrict/api/v1/components/{name} [delete]
func (a *APIController) DeleteComponent(c *gin.Context) {
	_ = c.Param("name")

	// 注意：这里需要实现删除组件的逻辑
	// 目前先返回成功状态，实际项目中需要实现具体的删除逻辑
	c.JSON(200, gin.H{"status": "success", "message": "component deletion not implemented yet"})
}
