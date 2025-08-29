package client

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
	root.RootCmd.AddCommand(metricsCmd)
	metricsCmd.Flags().SortFlags = false
	metricsCmd.Flags().StringVarP(&pushGatewayAddr, "addr", "a", "", "Pushgateway address")
	metricsCmd.Flags().DurationP("timeout", "t", 30*time.Second, "Metrics collection timeout")
}

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Reporting statistical indicators",
	Run: func(cmd *cobra.Command, args []string) {
		// Keep timeout variable for future extension use
		_, _ = cmd.Flags().GetDuration("timeout")

		if pushGatewayAddr == "" {
			pushGatewayAddr = config.Get().Cloud.PushgatewayUrl
		}

		if err := services.CollectAndPushMetrics(pushGatewayAddr); err != nil {
			fmt.Printf("Failed to start metrics service: %v\nPlease check if the gateway address is correct and accessible", err)
		}
	},
}
