package service

import (
	"context"
	"costrict-keeper/cmd/root"
	"costrict-keeper/internal/models"
	"costrict-keeper/services"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [组件或服务名称]",
	Short: "列出所有组件和服务的信息",
	Long:  "列出所有组件和服务的信息，如果指定了名称，则只显示该组件或服务的详细信息",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := listInfo(context.Background(), args); err != nil {
			fmt.Println(err)
		}
	},
}

/**
 * List component and service information
 * @param {context.Context} ctx - Context for request cancellation and timeout
 * @param {[]string} args - Command line arguments, optionally containing component/service name
 * @returns {error} Returns error if listing fails, nil on success
 * @description
 * - Creates new service manager instance
 * - Lists all components and services if no arguments provided
 * - Lists specific component or service details if name provided
 * @throws
 * - Component/service listing failure errors
 * @example
 * err := listInfo(context.Background(), []string{})
 * if err != nil {
 *     log.Fatal(err)
 * }
 */
func listInfo(ctx context.Context, args []string) error {
	manager := services.NewServiceManager()

	if len(args) == 0 {
		// 列出所有组件和服务信息
		return listAllInfo(manager)
	} else {
		// 列出指定组件或服务的详细信息
		return listSpecificInfo(manager, args[0])
	}
}

func listAllInfo(manager *services.ServiceManager) error {
	fmt.Println("=== 组件信息 ===")
	components, err := manager.GetComponents()
	if err != nil {
		return fmt.Errorf("获取组件信息失败: %v", err)
	}

	if len(components) == 0 {
		fmt.Println("没有找到组件")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "名称\t版本\t路径")
		for _, comp := range components {
			fmt.Fprintf(w, "%s\t%s\t%s\n", comp.Name, comp.Version, comp.Path)
		}
		w.Flush()
	}

	fmt.Println("\n=== 服务信息 ===")
	services := manager.GetServices()
	if len(services) == 0 {
		fmt.Println("没有找到服务")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "名称\t协议\t端口\t启动命令\t启动模式")
		for _, svc := range services {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				svc.Name, svc.Protocol, svc.Port, svc.Command, svc.Startup)
		}
		w.Flush()
	}

	fmt.Println("\n=== 端点信息 ===")
	endpoints := manager.GetEndpoints()
	if len(endpoints) == 0 {
		fmt.Println("没有找到端点")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "名称\tURL\t健康状态")
		for _, endpoint := range endpoints {
			healthStatus := "健康"
			if !endpoint.Healthy {
				healthStatus = "不健康"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", endpoint.Name, endpoint.URL, healthStatus)
		}
		w.Flush()
	}

	return nil
}

func listSpecificInfo(manager *services.ServiceManager, name string) error {
	// 查找服务
	services := manager.GetServices()
	var foundService *models.ServiceConfig
	for _, svc := range services {
		if svc.Name == name {
			foundService = &svc
			break
		}
	}

	if foundService != nil {
		fmt.Printf("=== 服务 '%s' 的详细信息 ===\n", name)
		fmt.Printf("名称: %s\n", foundService.Name)
		fmt.Printf("协议: %s\n", foundService.Protocol)
		fmt.Printf("端口: %s\n", foundService.Port)
		fmt.Printf("启动命令: %s\n", foundService.Command)
		fmt.Printf("启动模式: %s\n", foundService.Startup)
		if foundService.Metrics != "" {
			fmt.Printf("指标端点: %s\n", foundService.Metrics)
		}

		// 检查健康状态
		isHealthy := manager.IsServiceHealthy(name)
		healthStatus := "健康"
		if !isHealthy {
			healthStatus = "不健康"
		}
		fmt.Printf("健康状态: %s\n", healthStatus)

		// 显示端点URL
		if foundService.Protocol != "" && foundService.Port != "" {
			endpointURL := fmt.Sprintf("%s://localhost:%s", foundService.Protocol, foundService.Port)
			fmt.Printf("访问URL: %s\n", endpointURL)
		}

		return nil
	}

	// 如果没有找到服务，尝试查找组件
	components, err := manager.GetComponents()
	if err != nil {
		return fmt.Errorf("获取组件信息失败: %v", err)
	}

	var foundComponent *models.ComponentInfo
	for _, comp := range components {
		if comp.Name == name {
			foundComponent = &comp
			break
		}
	}

	if foundComponent != nil {
		fmt.Printf("=== 组件 '%s' 的详细信息 ===\n", name)
		fmt.Printf("名称: %s\n", foundComponent.Name)
		fmt.Printf("版本: %s\n", foundComponent.Version)
		fmt.Printf("路径: %s\n", foundComponent.Path)
		return nil
	}

	return fmt.Errorf("未找到名为 '%s' 的组件或服务", name)
}

func init() {
	root.RootCmd.AddCommand(listCmd)
}
