package service

import (
	"context"
	"costrict-keeper/cmd/root"
	"costrict-keeper/services"
	"fmt"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [服务名称]",
	Short: "启动服务",
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
	manager := services.NewServiceManager()
	if err := manager.StartService(ctx, serviceName); err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}
	fmt.Printf("服务 %s 已启动\n", serviceName)
	return nil
}

func init() {
	root.RootCmd.AddCommand(startCmd)
}
