package server

import (
	"context"
	"costrict-keeper/cmd/root"
	"costrict-keeper/controllers"
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/logger"
	"costrict-keeper/services"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var listenAddr string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动HTTP服务",
	Run: func(cmd *cobra.Command, args []string) {
		if err := startServer(context.Background()); err != nil {
			logger.Fatal(err)
		}
	},
}

/**
 * Start HTTP server with all services
 * @param {context.Context} ctx - Context for request cancellation and timeout
 * @returns {error} Returns error if server startup fails, nil on success
 * @description
 * - Initializes Gin router with default middleware
 * - Creates server service and service manager instances
 * - Registers API routes and controllers
 * - Starts all managed services
 * - Launches monitoring and log reporting goroutines
 * - Determines listening address from command line or config
 * - Starts HTTP server on determined address
 * @throws
 * - Service startup errors
 * - HTTP server startup errors
 * @example
 * err := startServer(context.Background())
 * if err != nil {
 *     logger.Fatal(err)
 * }
 */
func startServer(ctx context.Context) error {
	// 初始化服务
	router := gin.Default()
	svc := services.NewServerService(&config.Config)
	// 初始化服务管理器
	svcManager := services.NewServiceManager()

	// 注册API路由
	apiController := controllers.NewAPIController(svcManager)
	apiController.RegisterRoutes(router)

	// 启动所有服务
	if err := svcManager.StartAll(ctx); err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}

	// 启动监控和日志上报
	go svc.StartMonitoring(svcManager)
	go svc.StartLogReporting()

	// 确定监听地址：优先使用命令行参数，其次使用配置文件
	address := config.Config.Server.Address
	if listenAddr != "" {
		address = listenAddr
	}

	return router.Run(address)
}

func init() {
	serverCmd.Flags().SortFlags = false
	serverCmd.Flags().StringVarP(&listenAddr, "listen", "l", "", "服务器侦听地址 (例如: :8080)")
	root.RootCmd.AddCommand(serverCmd)
}
