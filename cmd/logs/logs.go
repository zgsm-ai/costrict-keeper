package logs

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"costrict-host/cmd/root"
	"costrict-host/services"
)

var (
	logFile     string
	serviceName string
	logService  *services.LogService
)

func init() {
	root.RootCmd.AddCommand(Cmd)
	Cmd.Flags().StringVarP(&logFile, "file", "f", "", "日志文件路径")
	Cmd.Flags().StringVarP(&serviceName, "service", "s", "", "服务名称")
}

var Cmd = &cobra.Command{
	Use:   "logs",
	Short: "上报日志到云端",
	Run: func(cmd *cobra.Command, args []string) {
		if logFile == "" {
			fmt.Println("请使用-f参数指定日志文件")
			return
		}
		logService = services.NewLogService(viper.GetViper())

		dest, err := logService.UploadLog(logFile, serviceName)
		if err != nil {
			fmt.Printf("日志上传失败: %v\n", err)
			return
		}

		fmt.Printf("上传成功: %s -> %s\n", logFile, dest)
	},
}
