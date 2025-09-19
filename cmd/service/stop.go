package service

import (
	"costrict-keeper/internal/rpc"
	"costrict-keeper/services"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop {service-name}",
	Short: "Stop service",
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
	rpcClient := rpc.NewHTTPClient(nil)
	apiPath := fmt.Sprintf("/costrict/api/v1/services/%s/stop", serviceName)
	resp, err := rpcClient.Post(apiPath, nil)
	if err == nil {
		rpcClient.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			fmt.Printf("Failed to stop service '%s' via costrict server: %+v\n", serviceName, resp.Body)
			return os.ErrInvalid
		}
		fmt.Printf("Service '%s' has been stopped via costrict server\n", serviceName)
		return nil
	}

	// 如果 API 调用失败，关闭连接并继续原有逻辑
	rpcClient.Close()

	// 如果无法连接到 costrict 服务器或 API 调用失败，走原有逻辑
	manager := services.GetServiceManager()
	if err := manager.StopService(serviceName); err != nil {
		fmt.Printf("Failed to stop service: %v\n", err)
		return err
	}
	fmt.Printf("Service '%s' has been stopped locally\n", serviceName)
	return nil
}

func init() {
	serviceCmd.AddCommand(stopCmd)
}
