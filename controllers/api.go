package controllers

import (
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/env"
	"costrict-keeper/services"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type APIController struct {
	server *services.Server
}

/**
 * Create new API controller instance
 * @param {*services.ServiceManager} svc - Service manager instance for managing services
 * @returns {*APIController} New API controller instance
 * @description
 * - Initializes controller with service manager
 * - Used to manage API routes and handlers for service operations
 * @example
 * svcManager := services.GetServiceManager()
 * controller := controllers.NewAPIController(svcManager)
 */
func NewAPIController(server *services.Server) *APIController {
	return &APIController{
		server: server,
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
	r.POST("/costrict/api/v1/reload", a.ReloadConfig)
	r.GET("/healthz", a.Healthz)
}

// @Summary 重新加载配置
// @Description 重新加载应用配置文件
// @Tags Config
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /costrict/api/v1/reload [post]
func (a *APIController) ReloadConfig(c *gin.Context) {
	// 获取配置文件路径
	configPath := filepath.Join(env.CostrictDir, "config", "costrict.json")

	// 调用配置重新加载方法
	if err := config.LoadConfigFromPath(configPath); err != nil {
		c.JSON(500, gin.H{
			"code":    "config.reload_failed",
			"message": "Failed to reload configuration: " + err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "Configuration reloaded successfully",
	})
}

func (a *APIController) Healthz(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "success",
		"message": "",
	})
}
