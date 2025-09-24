package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"costrict-keeper/cmd/root"
	"costrict-keeper/internal/env"
	"costrict-keeper/internal/logger"
	"costrict-keeper/internal/models"
	"costrict-keeper/internal/rpc"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check server status and health",
	Long:  `Check server status and health by connecting to the costrict server via unix socket and calling the check API`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := checkServerStatus(); err != nil {
			log.Fatal(err)
		}
	},
}

const checkExample = `  # Check server status
  costrict check

  # Check server status with custom costrict directory
  costrict check --costrict /path/to/costrict`

/**
 * Check server status by connecting via unix socket and calling check API
 * @returns {error} Returns error if check fails, nil on success
 * @description
 * - Creates HTTP client with unix socket transport
 * - Connects to costrict server via unix socket
 * - Calls /costrict/api/v1/check endpoint
 * - Parses and displays check results
 * - Shows overall system health status
 * @throws
 * - Unix socket connection errors
 * - HTTP request errors
 * - JSON parsing errors
 * @example
 * err := checkServerStatus()
 * if err != nil {
 *     logger.Fatal("Check failed:", err)
 * }
 */
func checkServerStatus() error {
	// Get unix socket path
	socketPath := rpc.GetSocketPath("costrict.sock", "")

	// Check if socket file exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return fmt.Errorf("unix socket not found at %s, please ensure costrict server is running", socketPath)
	}

	// Create HTTP client with unix socket transport
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, "unix", socketPath)
			},
		},
		Timeout: 30 * time.Second,
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "http://unix/costrict/api/v1/check", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "costrict-cli")

	// Send request
	logger.Info("Connecting to costrict server via unix socket...")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned error status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var checkResp models.CheckResponse
	if err := json.Unmarshal(body, &checkResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Display check results
	displayCheckResults(checkResp)

	return nil
}

/**
 * Get unix socket path for costrict server
 * @returns {string} Returns the unix socket file path
 * @description
 * - Constructs socket path using costrict directory
 * - Uses "costrict.sock" as socket filename
 * @example
 * socketPath := getUnixSocketPath()
 * fmt.Printf("Socket path: %s", socketPath)
 */
func getUnixSocketPath() string {
	return filepath.Join(env.CostrictDir, "costrict.sock")
}

/**
 * Display formatted check results to user
 * @param {models.CheckResponse} results - Check results from server
 * @description
 * - Formats and displays overall system status
 * - Shows service, process, tunnel, and component check results
 * - Displays midnight rooster status
 * - Shows summary statistics
 * @example
 * results := models.CheckResponse{OverallStatus: "healthy", ...}
 * displayCheckResults(results)
 */
func displayCheckResults(results models.CheckResponse) {
	fmt.Println("=== Costrict Server Status Check ===")
	fmt.Println()

	// Display timestamp
	fmt.Printf("检查时间: %s\n", results.Timestamp.Format(time.RFC3339))
	fmt.Println()

	// Display overall status
	statusIcon := "✅"
	if results.OverallStatus == "warning" {
		statusIcon = "⚠️"
	} else if results.OverallStatus == "error" {
		statusIcon = "❌"
	}
	fmt.Printf("%s 总体状态: %s\n", statusIcon, results.OverallStatus)
	fmt.Println()

	// Display statistics
	fmt.Printf("总检查项: %d\n", results.TotalChecks)
	fmt.Printf("通过检查: %d\n", results.PassedChecks)
	fmt.Printf("失败检查: %d\n", results.FailedChecks)
	fmt.Println()

	// Display services
	if len(results.Services) > 0 {
		fmt.Printf("=== 服务检查结果 (%d 项) ===\n", len(results.Services))
		for _, svc := range results.Services {
			statusIcon := "✅"
			if !svc.Healthy || svc.Status != "running" {
				statusIcon = "❌"
			}

			fmt.Printf("%s %s", statusIcon, svc.Name)
			if svc.Pid > 0 {
				fmt.Printf(" (PID: %d)", svc.Pid)
			}
			if svc.Port > 0 {
				fmt.Printf(" 端口: %d", svc.Port)
			}
			if svc.RestartCount > 0 {
				fmt.Printf(" 重启次数: %d", svc.RestartCount)
			}
			fmt.Printf(" 状态: %s", svc.Status)
			if svc.Healthy {
				fmt.Printf(" 健康")
			} else {
				fmt.Printf(" 不健康")
			}
			fmt.Println()
			if svc.Tunnel.Enabled {
				statusIcon := "✅"
				if !svc.Tunnel.Healthy {
					statusIcon = "❌"
				}
				fmt.Printf("%s %s", statusIcon, svc.Name)
				if svc.Tunnel.Pid > 0 {
					fmt.Printf(" (PID: %d)", svc.Tunnel.Pid)
				}
				fmt.Printf(" %s, 隧道: %d", svc.Tunnel.Status, len(svc.Tunnel.Ports))
				for _, tun := range svc.Tunnel.Ports {
					fmt.Printf(" (本地端口: %d -> 映射端口: %d)", tun.LocalPort, tun.MappingPort)
				}
				if svc.Tunnel.Healthy {
					fmt.Printf(" 健康")
				} else {
					fmt.Printf(" 不健康")
				}
				fmt.Println()
			}
		}
		fmt.Println()
	}

	// Display components
	if len(results.Components) > 0 {
		fmt.Printf("=== 组件检查结果 (%d 项) ===\n", len(results.Components))
		for _, cpn := range results.Components {
			statusIcon := "✅"
			if !cpn.Installed || cpn.NeedUpgrade {
				statusIcon = "❌"
			}

			fmt.Printf("%s %s", statusIcon, cpn.Name)
			if cpn.Installed {
				fmt.Printf(" (本地版本: %s", cpn.LocalVersion)
				if cpn.NeedUpgrade {
					fmt.Printf(" -> 远程版本: %s) 需要升级", cpn.RemoteVersion)
				} else {
					fmt.Printf(") 已安装")
				}
			} else {
				fmt.Printf(" 未安装")
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// Display midnight rooster status
	fmt.Println("=== 半夜鸡叫检查结果 ===")
	fmt.Printf("状态: %s\n", results.MidnightRooster.Status)
	fmt.Printf("下次检查时间: %s\n", results.MidnightRooster.NextCheckTime.Format(time.RFC3339))
	fmt.Printf("最后检查时间: %s\n", results.MidnightRooster.LastCheckTime.Format(time.RFC3339))
	fmt.Printf("组件总数: %d\n", results.MidnightRooster.ComponentsCount)
	fmt.Printf("需要升级: %d\n", results.MidnightRooster.UpgradesNeeded)
	fmt.Println()

	fmt.Println("=== 检查完成 ===")
}

func init() {
	checkCmd.Flags().SortFlags = false
	root.RootCmd.AddCommand(checkCmd)
	checkCmd.Example = checkExample
}
