package tunnel

import (
	"fmt"
	"log"

	"costrict-keeper/internal/rpc"
	"costrict-keeper/services"

	"github.com/spf13/cobra"
)

var (
	startApp  string
	startPort int
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start tunnel connection",
	Run: func(cmd *cobra.Command, args []string) {
		if startApp == "" {
			log.Fatal("Must specify app name (--app)")
		}

		// 尝试使用 RPC 客户端连接 costrict 服务器
		rpcClient := rpc.NewHTTPClient(nil)
		if rpcClient != nil && tryStartTunnelViaRPC(rpcClient, startApp, startPort) {
			// RPC 调用成功，直接返回
			return
		}

		// RPC 连接失败，回退到原有逻辑
		log.Printf("Failed to connect to costrict server via RPC, falling back to local tunnel management")
		tunnelSvc := services.GetTunnelManager()
		tun, err := tunnelSvc.StartTunnel(startApp, startPort)
		if err != nil {
			log.Fatalf("Failed to start tunnel: %v", err)
		}
		fmt.Printf("Successfully started tunnel for app %s, local port: %d, remote port: %d",
			startApp, tun.LocalPort, tun.MappingPort)
	},
}

// tryStartTunnelViaRPC 尝试通过 RPC 连接启动隧道
/**
 * Try to start tunnel via RPC connection to costrict server
 * @param {rpc.HTTPClient} rpcClient - RPC client instance
 * @param {string} appName - Application name
 * @param {int} port - Port number for tunnel
 * @returns {bool} True if RPC call succeeded, false otherwise
 * @description
 * - Attempts to connect to costrict server via Unix socket
 * - Calls /costrict/api/v1/tunnels endpoint to create tunnel
 * - Handles connection errors and API response errors
 * - Returns success/failure status for fallback logic
 * @throws
 * - Connection establishment errors
 * - API request errors
 * - Response parsing errors
 * @example
 * success := tryStartTunnelViaRPC(rpcClient, "myapp", 8080)
 * if success {
 *     fmt.Println("Tunnel started via RPC")
 * }
 */
func tryStartTunnelViaRPC(rpcClient rpc.HTTPClient, appName string, port int) bool {
	// 构建请求数据
	requestData := map[string]interface{}{
		"app":  appName,
		"port": port,
	}

	// 尝试调用 costrict 的 RESTful API
	response, err := rpcClient.Post("/costrict/api/v1/tunnels", requestData)
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
						fmt.Printf("Successfully started tunnel via costrict server: %s\n", messageStr)
						return true
					}
				}
			}
			// 即使没有message字段，只要状态码在200-299范围内，也认为成功
			fmt.Printf("Successfully started tunnel via costrict server, status code: %d\n", httpResp.StatusCode)
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
	startCmd.Flags().SortFlags = false
	startCmd.Flags().StringVarP(&startApp, "app", "a", "", "App name")
	startCmd.Flags().IntVarP(&startPort, "port", "p", 0, "Mapping port")

	tunnelCmd.AddCommand(startCmd)
}
