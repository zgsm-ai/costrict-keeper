package service

import (
	"context"
	"costrict-keeper/cmd/root"
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/models"
	"costrict-keeper/internal/utils"
	"costrict-keeper/services"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [服务名称]",
	Short: "查看服务状态",
	Long:  "查看所有服务的运行状态，如果指定了服务名称，则只显示该服务的详细信息。",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := showServiceStatus(context.Background(), args); err != nil {
			fmt.Println(err)
		}
	},
}

/**
 * Load system specification from system-spec.json
 * @returns {(*models.SystemSpecification, error)} Returns subsystem configuration and error if any
 * @description
 * - Reads system-spec.json from share directory
 * - Unmarshals JSON into SystemSpecification struct
 * - Returns loaded configuration
 * @throws
 * - File reading errors
 * - JSON unmarshaling errors
 */
func loadSystemSpec() (*models.SystemSpecification, error) {
	specPath := filepath.Join(config.Config.Directory.Share, "system-spec.json")

	bytes, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取 system-spec.json: %v", err)
	}

	var spec models.SystemSpecification
	if err := json.Unmarshal(bytes, &spec); err != nil {
		return nil, fmt.Errorf("解析 system-spec.json 失败: %v", err)
	}

	return &spec, nil
}

/**
 * Get service status
 * @param {string} serviceName - Name of the service
 * @returns {string} Returns service status string
 * @description
 * - Creates service manager instance
 * - Checks if service is healthy using IsServiceHealthy
 * - Returns "运行中" or "已停止"
 */
func getServiceStatus(serviceName string) string {
	manager := services.NewServiceManager()
	if manager.IsServiceHealthy(serviceName) {
		return "运行中"
	}
	return "已停止"
}

/**
 * Get local version for a service
 * @param {string} serviceName - Name of the service
 * @returns {(*utils.VersionNumber, error)} Returns local version and error if any
 * @description
 * - Creates upgrade config for service
 * - Calls GetLocalVersion to retrieve installed version
 * - Returns version number or error
 * @throws
 * - Version retrieval errors
 */
func getLocalVersion(serviceName string) (*utils.VersionNumber, error) {
	cfg := utils.UpgradeConfig{
		PackageName: serviceName,
		PackageDir:  config.Config.Directory.Package,
		InstallDir:  config.Config.Directory.Bin,
	}
	cfg.Correct()

	ver, err := utils.GetLocalVersion(cfg)
	if err != nil {
		return nil, err
	}
	return &ver, nil
}

/**
 * Get remote version for a service
 * @param {string} serviceName - Name of the service
 * @returns {(*utils.VersionNumber, error)} Returns remote version and error if any
 * @description
 * - Creates upgrade config for service
 * - Calls GetRemoteVersions to retrieve available versions
 * - Returns newest version number or error
 * @throws
 * - Version retrieval errors
 */
func getRemoteVersion(serviceName string) (*utils.VersionNumber, error) {
	cfg := utils.UpgradeConfig{
		PackageName: serviceName,
		PackageDir:  config.Config.Directory.Package,
		InstallDir:  config.Config.Directory.Bin,
		BaseUrl:     config.Config.Upgrade.BaseUrl,
	}
	cfg.Correct()

	versions, err := utils.GetRemoteVersions(cfg)
	if err != nil {
		return nil, err
	}
	return &versions.Newest.VersionId, nil
}

/**
 * Show service status information
 * @param {context.Context} ctx - Context for request cancellation and timeout
 * @param {[]string} args - Command line arguments, optionally containing service name
 * @returns {error} Returns error if showing status fails, nil on success
 * @description
 * - Loads system configuration from system-spec.json
 * - Shows all services status if no arguments provided
 * - Shows specific service details if name provided
 * @throws
 * - Configuration loading errors
 * - Service status checking errors
 */
func showServiceStatus(ctx context.Context, args []string) error {
	// 加载系统配置
	spec, err := loadSystemSpec()
	if err != nil {
		return fmt.Errorf("加载系统配置失败: %v", err)
	}

	if len(args) == 0 {
		// 显示所有服务状态
		return showAllServicesStatus(spec)
	} else {
		// 显示指定服务的详细信息
		return showSpecificServiceStatus(spec, args[0])
	}
}

/**
 * Show all services status with detailed information
 * @param {spec *models.SystemSpecification} System configuration
 * @returns {error} Returns error if showing status fails, nil on success
 * @description
 * - Lists all services with status information
 * - Uses tabwriter for formatted output
 */
func showAllServicesStatus(spec *models.SystemSpecification) error {
	fmt.Println("=== 服务状态 ===")
	if len(spec.Services) == 0 {
		fmt.Println("没有找到服务")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "名称\t协议\t端口\t启动模式\t状态")
		for _, svc := range spec.Services {
			status := getServiceStatus(svc.Name)
			portStr := fmt.Sprintf("%d", svc.Port)
			if svc.Port == 0 {
				portStr = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", svc.Name, svc.Protocol, portStr, svc.Startup, status)
		}
		w.Flush()
	}

	return nil
}

/**
 * Show specific service details with status
 * @param {spec *models.SystemSpecification} System configuration
 * @param {string} name - Name of service
 * @returns {error} Returns error if showing status fails, nil on success
 * @description
 * - Searches for service by name
 * - Displays detailed information with status
 * - Shows version information
 * @throws
 * - Service not found errors
 */
func showSpecificServiceStatus(spec *models.SystemSpecification, name string) error {
	// 查找服务
	for _, svc := range spec.Services {
		if svc.Name == name {
			fmt.Printf("=== 服务 '%s' 的详细信息 ===\n", name)
			fmt.Printf("名称: %s\n", svc.Name)
			fmt.Printf("协议: %s\n", svc.Protocol)
			fmt.Printf("端口: %d\n", svc.Port)
			fmt.Printf("启动命令: %s\n", svc.Command)
			fmt.Printf("启动模式: %s\n", svc.Startup)
			if svc.Metrics != "" {
				fmt.Printf("指标端点: %s\n", svc.Metrics)
			}
			if svc.Accessible != "" {
				fmt.Printf("访问权限: %s\n", svc.Accessible)
			}

			// 显示服务状态
			status := getServiceStatus(svc.Name)
			fmt.Printf("运行状态: %s\n", status)

			// 显示版本信息
			if localVer, err := getLocalVersion(svc.Name); err == nil {
				fmt.Printf("本地版本: %d.%d.%d\n", localVer.Major, localVer.Minor, localVer.Micro)
			} else {
				fmt.Printf("本地版本: 未安装\n")
			}

			if remoteVer, err := getRemoteVersion(svc.Name); err == nil {
				fmt.Printf("服务器最新版本: %d.%d.%d\n", remoteVer.Major, remoteVer.Minor, remoteVer.Micro)
			} else {
				fmt.Printf("服务器最新版本: 无法获取\n")
			}

			// 显示端点URL
			if svc.Protocol != "" && svc.Port > 0 {
				endpointURL := fmt.Sprintf("%s://localhost:%d", svc.Protocol, svc.Port)
				fmt.Printf("访问URL: %s\n", endpointURL)
			}

			return nil
		}
	}

	return fmt.Errorf("未找到名为 '%s' 的服务", name)
}

func init() {
	root.RootCmd.AddCommand(statusCmd)
}
