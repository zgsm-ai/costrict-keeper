package service

import (
	"context"
	"costrict-keeper/internal/rpc"
	"costrict-keeper/services"
	"fmt"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [service name]",
	Short: "Restart service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := restartService(context.Background(), args[0]); err != nil {
			fmt.Println(err)
		}
	},
}

/**
 * Restart service by name
 * @param {context.Context} ctx - Context for request cancellation and timeout
 * @param {string} serviceName - Name of the service to restart
 * @returns {error} Returns error if service restart fails, nil on success
 * @description
 * - Creates new service manager instance
 * - Attempts to restart the specified service
 * - Prints success message if service restarts successfully
 * @throws
 * - Service restart failure errors
 * @example
 * err := restartService(context.Background(), "codebase-syncer")
 * if err != nil {
 *     logger.Fatal(err)
 * }
 */
func restartService(ctx context.Context, serviceName string) error {
	// 尝试通过 RPC 客户端连接到 costrict 服务器并重启服务
	if err := restartServiceViaRPC(ctx, serviceName); err != nil {
		// 如果 RPC 连接失败，回退到原有逻辑
		fmt.Printf("RPC connection failed, using local restart: %v\n", err)
		return restartServiceLocally(ctx, serviceName)
	}

	fmt.Printf("Service %s has been restarted via RPC\n", serviceName)
	return nil
}

/**
 * Restart service via RPC client to costrict server
 * @param {context.Context} ctx - Context for request cancellation and timeout
 * @param {string} serviceName - Name of the service to restart
 * @returns {error} Returns error if RPC restart fails, nil on success
 * @description
 * - Creates RPC client with default configuration
 * - Attempts to connect to costrict server via Unix socket
 * - Calls restart API endpoint if connection succeeds
 * - Returns error if connection or API call fails
 * @throws
 * - RPC client creation errors
 * - Connection establishment errors
 * - API call errors
 * @example
 * err := restartServiceViaRPC(context.Background(), "codebase-syncer")
 * if err != nil {
 *     log.Printf("RPC restart failed: %v", err)
 * }
 */
func restartServiceViaRPC(ctx context.Context, serviceName string) error {
	// 创建 RPC 客户端配置
	config := rpc.DefaultHTTPConfig()
	config.ServerName = "costrict"

	// 创建 RPC 客户端
	client := rpc.NewHTTPClient(config)
	defer client.Close()

	// 调用重启服务的 API
	apiPath := fmt.Sprintf("/costrict/api/v1/services/%s/restart", serviceName)
	resp, err := client.Post(apiPath, nil)
	if err != nil {
		return fmt.Errorf("failed to call restart API: %w", err)
	}

	// 检查响应状态
	if httpResp, ok := resp.(*rpc.HTTPResponse); ok {
		if httpResp.StatusCode >= 400 {
			return fmt.Errorf("restart API returned status code %d", httpResp.StatusCode)
		}
	}

	return nil
}

/**
 * Restart service locally using service manager
 * @param {context.Context} ctx - Context for request cancellation and timeout
 * @param {string} serviceName - Name of the service to restart
 * @returns {error} Returns error if local restart fails, nil on success
 * @description
 * - Gets service manager instance
 * - Calls local service restart method
 * - Returns error if restart fails
 * @throws
 * - Service manager errors
 * - Service restart errors
 * @example
 * err := restartServiceLocally(context.Background(), "codebase-syncer")
 * if err != nil {
 *     log.Fatal(err)
 * }
 */
func restartServiceLocally(ctx context.Context, serviceName string) error {
	manager := services.GetServiceManager()
	if err := manager.RestartService(ctx, serviceName); err != nil {
		return fmt.Errorf("Failed to restart service: %v", err)
	}
	fmt.Printf("Service %s has been restarted locally\n", serviceName)
	return nil
}

func init() {
	serviceCmd.AddCommand(restartCmd)
}
