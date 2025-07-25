package service

import (
	"costrict-host/cmd/root"

	"github.com/spf13/cobra"
)

var ServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "服务管理",
	Long:  "服务管理命令集，包含启动、停止和重启服务等功能",
}

func init() {
	root.RootCmd.AddCommand(ServiceCmd)
}
