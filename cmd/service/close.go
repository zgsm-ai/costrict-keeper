package service

import (
	"fmt"
	"log"

	"costrict-keeper/internal/rpc"
	"costrict-keeper/services"

	"github.com/spf13/cobra"
)

var closeCmd = &cobra.Command{
	Use:   "close {service-name}",
	Short: "Close tunnel for specified serivce",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]
		if serviceName == "" {
			log.Fatal("Must specify service name")
		}

		// 尝试使用 RPC 客户端连接 costrict 服务器
		rpcClient := rpc.NewHTTPClient(nil)
		if tryStopTunnelViaRPC(rpcClient, serviceName) {
			// RPC 调用成功，直接返回
			return
		}

		// RPC 连接失败，回退到原有逻辑
		log.Printf("Failed to connect to costrict server via RPC, falling back to local tunnel management")
		service := services.GetServiceManager()
		svc := service.GetInstance(serviceName)
		if svc == nil {
			log.Fatalf("Failed to close '%s' tunnel", serviceName)
		}
		svc.CloseTunnel()

		fmt.Printf("Successfully close tunnel for service '%s'", serviceName)
	},
}

/**
 * Try to close tunnel via RPC connection to costrict server
 * @param {rpc.HTTPClient} rpcClient - RPC client instance
 * @param {string} serviceName - Application name
 * @param {int} port - Port number for tunnel
 * @returns {bool} True if RPC call succeeded, false otherwise
 * @description
 * - Attempts to connect to costrict server via Unix socket
 * - Calls DELETE /costrict/api/v1/tunnels/{app}/{port} endpoint to close tunnel
 * - Handles connection errors and API response errors
 * - Returns success/failure status for fallback logic
 * @throws
 * - Connection establishment errors
 * - API request errors
 * - Response parsing errors
 * @example
 * success := tryStopTunnelViaRPC(rpcClient, "myapp", 8080)
 * if success {
 *     fmt.Println("Tunnel stopped via RPC")
 * }
 */
func tryStopTunnelViaRPC(rpcClient rpc.HTTPClient, serviceName string) bool {
	// 构建 API 路径，包含应用名称和端口参数
	path := fmt.Sprintf("/costrict/api/v1/services/%s/close", serviceName)

	// 尝试调用 costrict 的 RESTful API DELETE 方法
	response, err := rpcClient.Post(path, nil)
	if err != nil {
		log.Printf("Failed to call costrict API: %v", err)
		return false
	}

	// 检查响应状态码
	if httpResp, ok := response.(*rpc.HTTPResponse); ok {
		// 检查HTTP状态码是否在200-299范围内
		if httpResp.StatusCode >= 200 && httpResp.StatusCode <= 299 {
			if httpResp.Body != nil {
				if message, msgExists := httpResp.Body["message"]; msgExists {
					if messageStr, ok := message.(string); ok {
						fmt.Printf("Successfully stopped tunnel via costrict server: %s\n", messageStr)
						return true
					}
				}
			}
			// 即使没有message字段，只要状态码在200-299范围内，也认为成功
			fmt.Printf("Successfully stopped tunnel via costrict server, status code: %d\n", httpResp.StatusCode)
			return true
		}

		// 如果响应中包含错误信息
		if httpResp.Body != nil {
			if errorMsg, exists := httpResp.Body["error"]; exists {
				if errorStr, ok := errorMsg.(string); ok {
					log.Printf("Costrict API returned error: %s", errorStr)
				}
			}
		}
	}

	log.Printf("Unexpected response from costrict API")
	return false
}

func init() {
	serviceCmd.AddCommand(closeCmd)
}
