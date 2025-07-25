package root

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "costrict-host",
	Short: "移动端命令行程序管理器",
	Long:  `costrict-host管理多个CLI程序的下载、安装、启动、配置、监控、服务注册`,
}
