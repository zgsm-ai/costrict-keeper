package service

import (
	"costrict-keeper/cmd/root"
	"costrict-keeper/services"
	"fmt"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [服务名称]",
	Short: "停止服务",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := stopService(args[0]); err != nil {
			fmt.Println(err)
		}
	},
}

/**
 * Stop service by name
 * @param {string} serviceName - Name of the service to stop
 * @returns {error} Returns error if service stop fails, nil on success
 * @description
 * - Creates new service manager instance
 * - Attempts to stop the specified service
 * - Prints success message if service stops successfully
 * @throws
 * - Service stop failure errors
 * @example
 * err := stopService("codebase-syncer")
 * if err != nil {
 *     logger.Fatal(err)
 * }
 */
func stopService(serviceName string) error {
	manager := services.NewServiceManager()
	if err := manager.StopService(serviceName); err != nil {
		return fmt.Errorf("停止服务失败: %v", err)
	}
	fmt.Printf("服务 %s 已停止\n", serviceName)
	return nil
}

func init() {
	root.RootCmd.AddCommand(stopCmd)
}
