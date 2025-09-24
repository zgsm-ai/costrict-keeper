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
	"time"

	"costrict-keeper/cmd/root"
	"costrict-keeper/internal/logger"
	"costrict-keeper/internal/rpc"

	"github.com/spf13/cobra"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload server configuration",
	Long:  `Reload server configuration by connecting to the costrict server via unix socket and calling the reload API`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := reloadServerConfig(); err != nil {
			log.Fatal(err)
		}
	},
}

const reloadExample = `  # Reload server configuration
  costrict reload

  # Reload server configuration with custom costrict directory
  costrict reload --costrict /path/to/costrict`

/**
 * Reload server configuration by connecting via unix socket and calling reload API
 * @returns {error} Returns error if reload fails, nil on success
 * @description
 * - Creates HTTP client with unix socket transport
 * - Connects to costrict server via unix socket
 * - Calls /costrict/api/v1/reload endpoint
 * - Parses and displays reload results
 * - Shows configuration reload status
 * @throws
 * - Unix socket connection errors
 * - HTTP request errors
 * - JSON parsing errors
 * @example
 * err := reloadServerConfig()
 * if err != nil {
 *     logger.Fatal("Reload failed:", err)
 * }
 */
func reloadServerConfig() error {
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
	req, err := http.NewRequest("POST", "http://unix/costrict/api/v1/reload", nil)
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
	var reloadResp map[string]interface{}
	if err := json.Unmarshal(body, &reloadResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Display reload results
	displayReloadResults(reloadResp)

	return nil
}

/**
 * Display formatted reload results to user
 * @param {map[string]interface{}} results - Reload results from server
 * @description
 * - Formats and displays configuration reload status
 * - Shows success/failure status
 * - Displays any relevant information from the reload response
 * @example
 * results := map[string]interface{}{"status": "success", "message": "Configuration reloaded"}
 * displayReloadResults(results)
 */
func displayReloadResults(results map[string]interface{}) {
	fmt.Println("=== Costrict Server Configuration Reload ===")
	fmt.Println()

	// Display timestamp if available
	if timestamp, ok := results["timestamp"].(string); ok {
		fmt.Printf("重载时间: %s\n", timestamp)
		fmt.Println()
	}

	// Display status
	status, ok := results["status"].(string)
	if !ok {
		status = "unknown"
	}

	statusIcon := "✅"
	if status != "success" {
		statusIcon = "❌"
	}
	fmt.Printf("%s 重载状态: %s\n", statusIcon, status)
	fmt.Println()

	// Display message if available
	if message, ok := results["message"].(string); ok {
		fmt.Printf("详细信息: %s\n", message)
		fmt.Println()
	}

	// Display any additional information
	if details, ok := results["details"].(map[string]interface{}); ok {
		fmt.Println("=== 详细信息 ===")
		for key, value := range details {
			fmt.Printf("%s: %v\n", key, value)
		}
		fmt.Println()
	}

	fmt.Println("=== 重载完成 ===")
}

func init() {
	reloadCmd.Flags().SortFlags = false
	root.RootCmd.AddCommand(reloadCmd)
	reloadCmd.Example = reloadExample
}
