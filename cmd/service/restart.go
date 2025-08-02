package service

import (
	"context"
	"costrict-keeper/cmd/root"
	"costrict-keeper/services"
	"fmt"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [服务名称]",
	Short: "重启服务",
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
	manager := services.NewServiceManager()
	if err := manager.RestartService(ctx, serviceName); err != nil {
		return fmt.Errorf("重启服务失败: %v", err)
	}
	fmt.Printf("服务 %s 已重启\n", serviceName)
	return nil
}

func init() {
	root.RootCmd.AddCommand(restartCmd)
}
