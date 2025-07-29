package metrics

import (
	"fmt"
	"time"

	"costrict-keeper/cmd/root"
	"costrict-keeper/internal/config"
	"costrict-keeper/services"

	"github.com/spf13/cobra"
)

var (
	pushGatewayAddr string
)

func init() {
	root.RootCmd.AddCommand(Cmd)
	Cmd.Flags().SortFlags = false
	Cmd.Flags().StringVarP(&pushGatewayAddr, "addr", "a", "", "Pushgateway地址")
	Cmd.Flags().DurationP("timeout", "t", 30*time.Second, "指标采集超时时间")
}

var Cmd = &cobra.Command{
	Use:   "metrics",
	Short: "上报Prometheus指标",
	Run: func(cmd *cobra.Command, args []string) {
		// 保留timeout变量供后续扩展使用
		_, _ = cmd.Flags().GetDuration("timeout")

		if pushGatewayAddr == "" {
			pushGatewayAddr = config.Config.Metrics.Pushgateway
		}

		if err := services.CollectAndPushMetrics(pushGatewayAddr); err != nil {
			fmt.Printf("指标服务启动失败: %v\n请检查Pushgateway地址是否正确且可访问", err)
		}
	},
}
