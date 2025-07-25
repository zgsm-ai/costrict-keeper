package server

import (
	"context"
	"costrict-host/cmd/root"
	"costrict-host/controllers"
	"costrict-host/internal/config"
	"costrict-host/services"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动HTTP服务",
	Run: func(cmd *cobra.Command, args []string) {
		if err := startServer(context.Background()); err != nil {
			log.Fatal(err)
		}
	},
}

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

	return router.Run(config.Config.Server.Address)
}

func init() {
	root.RootCmd.AddCommand(serverCmd)
}
