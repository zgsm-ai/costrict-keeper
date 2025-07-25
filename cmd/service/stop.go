package service

import (
	"costrict-host/services"
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

func stopService(serviceName string) error {
	manager := services.NewServiceManager()
	if err := manager.StopService(serviceName); err != nil {
		return fmt.Errorf("停止服务失败: %v", err)
	}
	fmt.Printf("服务 %s 已停止\n", serviceName)
	return nil
}

func init() {
	ServiceCmd.AddCommand(stopCmd)
}
