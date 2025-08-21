package service

import (
	"context"
	"costrict-keeper/internal/rpc"
	"costrict-keeper/services"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [service name]",
	Short: "Start service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := startService(context.Background(), args[0]); err != nil {
			fmt.Println(err)
		}
	},
}

/**
 * Start service by name
 * @param {context.Context} ctx - Context for request cancellation and timeout
 * @param {string} serviceName - Name of the service to start
 * @returns {error} Returns error if service start fails, nil on success
 * @description
 * - Creates new service manager instance
 * - Attempts to start the specified service
 * - Prints success message if service starts successfully
 * @throws
 * - Service start failure errors
 * @example
 * err := startService(context.Background(), "codebase-syncer")
 * if err != nil {
 *     logger.Fatal(err)
 * }
 */
func startService(ctx context.Context, serviceName string) error {
	// 尝试使用 RPC 客户端连接 costrict 服务器
	config := rpc.DefaultHTTPConfig()
	config.Timeout = 10 * time.Second
	rpcClient := rpc.NewHTTPClient(config)

	// 尝试连接服务器并发送请求
	apiPath := fmt.Sprintf("/costrict/api/v1/services/%s/start", serviceName)
	resp, err := rpcClient.Post(apiPath, nil)
	if err != nil {
		// 连接服务器失败或请求失败，使用原有逻辑
		rpcClient.Close()
		return startServiceLocally(ctx, serviceName)
	}

	// 检查响应状态
	if httpResp, ok := resp.(*rpc.HTTPResponse); ok {
		if httpResp.StatusCode >= 200 && httpResp.StatusCode < 300 {
			// 成功调用 API
			rpcClient.Close()
			fmt.Printf("Service %s has been started via costrict server\n", serviceName)
			return nil
		}
	}

	// API 调用失败，使用原有逻辑
	rpcClient.Close()
	return startServiceLocally(ctx, serviceName)
}

// startServiceLocally 使用本地服务管理器启动服务
/**
 * Start service using local service manager
 * @param {context.Context} ctx - Context for request cancellation and timeout
 * @param {string} serviceName - Name of the service to start
 * @returns {error} Returns error if service start fails, nil on success
 * @description
 * - Creates new service manager instance
 * - Attempts to start the specified service locally
 * - Prints success message if service starts successfully
 * @throws
 * - Service start failure errors
 * @example
 * err := startServiceLocally(context.Background(), "codebase-syncer")
 * if err != nil {
 *     logger.Fatal(err)
 * }
 */
func startServiceLocally(ctx context.Context, serviceName string) error {
	manager := services.GetServiceManager()
	if err := manager.StartService(ctx, serviceName); err != nil {
		return fmt.Errorf("Failed to start service: %v", err)
	}
	fmt.Printf("Service %s has been started locally\n", serviceName)
	return nil
}

func init() {
	serviceCmd.AddCommand(startCmd)
}
