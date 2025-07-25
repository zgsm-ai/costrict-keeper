package service

import (
	"context"
	"costrict-host/services"
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

func restartService(ctx context.Context, serviceName string) error {
	manager := services.NewServiceManager()
	if err := manager.RestartService(ctx, serviceName); err != nil {
		return fmt.Errorf("重启服务失败: %v", err)
	}
	fmt.Printf("服务 %s 已重启\n", serviceName)
	return nil
}

func init() {
	ServiceCmd.AddCommand(restartCmd)
}
