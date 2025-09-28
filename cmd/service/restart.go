package service

import (
	"context"
	"costrict-keeper/internal/rpc"
	"fmt"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart {service-name}",
	Short: "Restart service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := restartServiceViaRPC(context.Background(), args[0]); err != nil {
			fmt.Println(err)
		}
	},
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
	client := rpc.NewHTTPClient(nil)
	defer client.Close()

	// 调用重启服务的 API
	apiPath := fmt.Sprintf("/costrict/api/v1/services/%s/restart", serviceName)
	resp, err := client.Post(apiPath, nil)
	if err != nil {
		return fmt.Errorf("failed to call restart API: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode >= 400 {
		return fmt.Errorf("restart API returned status code %d", resp.StatusCode)
	}
	fmt.Printf("Service %s has been restarted via RPC\n", serviceName)
	return nil
}

func init() {
	serviceCmd.AddCommand(restartCmd)
}
