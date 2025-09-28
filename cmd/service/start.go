package service

import (
	"context"
	"costrict-keeper/internal/rpc"
	"fmt"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start {service-name}",
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
	rpcClient := rpc.NewHTTPClient(nil)
	apiPath := fmt.Sprintf("/costrict/api/v1/services/%s/start", serviceName)
	resp, err := rpcClient.Post(apiPath, nil)
	if err != nil {
		rpcClient.Close()
		return err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		rpcClient.Close()
		fmt.Printf("Service '%s' has been started via costrict server\n", serviceName)
		return nil
	}
	// API 调用失败，使用原有逻辑
	rpcClient.Close()
	return fmt.Errorf("http error: %s", resp.Error)
}

func init() {
	serviceCmd.AddCommand(startCmd)
}
