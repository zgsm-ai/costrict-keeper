package controllers

import (
	"costrict-keeper/internal/config"
	"costrict-keeper/services"

	"github.com/gin-gonic/gin"
)

type ComponentController struct {
	component *services.ComponentManager
}

/**
 * Create new Component controller instance
 * @param {*services.ComponentManager} component - Component manager instance for managing components
 * @returns {*ComponentController} New Component controller instance
 * @description
 * - Initializes controller with component manager
 * - Used to manage API routes and handlers for component operations
 * @example
 * compManager := services.GetComponentManager()
 * controller := controllers.NewComponentController(compManager)
 */
func NewComponentController(component *services.ComponentManager) *ComponentController {
	return &ComponentController{
		component: component,
	}
}

/**
 * Register all component API routes to Gin router group
 * @param {*gin.RouterGroup} r - Gin router group instance
 * @description
 * - Registers routes for:
 *   - Component management (list/upgrade/delete)
 * @example
 * api := router.Group("/costrict/api/v1")
 * controller := NewComponentController(compManager)
 * controller.RegisterRoutes(api)
 */
func (c *ComponentController) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/costrict/api/v1")
	// 组件管理接口
	api.GET("/components", c.ListComponents)
	api.POST("/components/:name/upgrade", c.UpgradeComponent)
	api.DELETE("/components/:name", c.DeleteComponent)
}

// @Summary 获取组件列表
// @Description 获取所有已安装组件信息
// @Tags Components
// @Produce json
// @Success 200 {array} services.ComponentInstance
// @Router /costrict/api/v1/components [get]
func (c *ComponentController) ListComponents(g *gin.Context) {
	components := c.component.GetComponents(true)
	g.JSON(200, components)
}

// @Summary 升级组件
// @Description 升级指定组件到最新版本
// @Tags Components
// @Param name path string true "组件名称"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string "{"code": "component.not_found", "message": "Component not found"}"
// @Router /costrict/api/v1/components/{name}/upgrade [post]
func (c *ComponentController) UpgradeComponent(g *gin.Context) {
	name := g.Param("name")
	if err := c.component.UpgradeComponent(name); err != nil {
		if err == config.ErrComponentNotFound {
			g.JSON(404, gin.H{
				"code":    "component.not_found",
				"message": "Component not found",
			})
		} else {
			g.JSON(500, gin.H{
				"code":    "component.upgrade_failed",
				"message": err.Error(),
			})
		}
		return
	}
	g.JSON(200, gin.H{"status": "success"})
}

// @Summary 删除组件
// @Description 根据组件名删除指定组件
// @Tags Components
// @Param name path string true "组件名称"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /costrict/api/v1/components/{name} [delete]
func (c *ComponentController) DeleteComponent(g *gin.Context) {
	_ = g.Param("name")

	// 注意：这里需要实现删除组件的逻辑
	// 目前先返回成功状态，实际项目中需要实现具体的删除逻辑
	g.JSON(200, gin.H{"status": "success", "message": "component deletion not implemented yet"})
}
