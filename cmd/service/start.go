package service

import (
	"context"
	"costrict-host/services"
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

func startService(ctx context.Context, serviceName string) error {
	manager := services.NewServiceManager()
	if err := manager.StartService(ctx, serviceName); err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}
	fmt.Printf("服务 %s 已启动\n", serviceName)
	return nil
}

func init() {
	ServiceCmd.AddCommand(startCmd)
}
