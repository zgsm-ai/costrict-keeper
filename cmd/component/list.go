package component

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

var listCmd = &cobra.Command{
	Use:   "list [组件名称]",
	Short: "列出所有组件的信息",
	Long:  "列出所有组件的信息，包括本地版本、服务器最新版本。如果指定了组件名称，则只显示该组件的详细信息。",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := listInfo(context.Background(), args); err != nil {
			fmt.Println(err)
		}
	},
}

/**
 * List component information with version details
 * @param {context.Context} ctx - Context for request cancellation and timeout
 * @param {[]string} args - Command line arguments, optionally containing component name
 * @returns {error} Returns error if listing fails, nil on success
 * @description
 * - Loads system configuration from system-spec.json
 * - Lists all components with version info if no arguments provided
 * - Lists specific component details if name provided
 * - Shows local version and remote version
 * @throws
 * - Configuration loading errors
 * - Version checking errors
 */
func listInfo(ctx context.Context, args []string) error {
	// 加载系统配置
	spec, err := loadSystemSpec()
	if err != nil {
		return fmt.Errorf("加载系统配置失败: %v", err)
	}

	if len(args) == 0 {
		// 列出所有组件信息
		return listAllComponents(spec)
	} else {
		// 列出指定组件的详细信息
		return listSpecificComponent(spec, args[0])
	}
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
	services.FetchRemoteSystemSpecification()
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
 * Get local version for a component
 * @param {string} componentName - Name of the component
 * @returns {(*utils.VersionNumber, error)} Returns local version and error if any
 * @description
 * - Creates upgrade config for component
 * - Calls GetLocalVersion to retrieve installed version
 * - Returns version number or error
 * @throws
 * - Version retrieval errors
 */
func getLocalVersion(componentName string) (*utils.VersionNumber, error) {
	cfg := utils.UpgradeConfig{
		PackageName: componentName,
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
 * Get remote version for a component
 * @param {string} componentName - Name of the component
 * @returns {(*utils.VersionNumber, error)} Returns remote version and error if any
 * @description
 * - Creates upgrade config for component
 * - Calls GetRemoteVersions to retrieve available versions
 * - Returns newest version number or error
 * @throws
 * - Version retrieval errors
 */
func getRemoteVersion(componentName string) (*utils.PlatformInfo, error) {
	cfg := utils.UpgradeConfig{
		PackageName: componentName,
		PackageDir:  config.Config.Directory.Package,
		InstallDir:  config.Config.Directory.Bin,
		BaseUrl:     config.Config.Upgrade.BaseUrl,
	}
	cfg.Correct()

	versions, err := utils.GetRemoteVersions(cfg)
	if err != nil {
		return nil, err
	}
	return &versions, nil
}

/**
 * List all components with detailed information
 * @param {spec *models.SystemSpecification} System configuration
 * @returns {error} Returns error if listing fails, nil on success
 * @description
 * - Lists components with local and remote versions
 * - Uses tabwriter for formatted output
 */
func listAllComponents(spec *models.SystemSpecification) error {
	if len(spec.Components) == 0 {
		fmt.Println("没有找到组件")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "名称\t本地版本\t服务器最新版本\t路径")
		for _, comp := range spec.Components {
			localVer := "未安装"
			remoteVer := "未知"

			if ver, err := getLocalVersion(comp.Name); err == nil {
				localVer = fmt.Sprintf("%d.%d.%d", ver.Major, ver.Minor, ver.Micro)
			}

			compPath := "-"
			vers, err := getRemoteVersion(comp.Name)
			if err == nil {
				remoteVer = utils.PrintVersion(vers.Newest.VersionId)
				compPath = vers.Newest.AppUrl
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", comp.Name, localVer, remoteVer, compPath)
		}
		w.Flush()
	}

	return nil
}

/**
 * List specific component details
 * @param {spec *models.SystemSpecification} System configuration
 * @param {string} name - Name of component
 * @returns {error} Returns error if listing fails, nil on success
 * @description
 * - Searches for component by name
 * - Displays detailed information with version comparison
 * @throws
 * - Component not found errors
 */
func listSpecificComponent(spec *models.SystemSpecification, name string) error {
	// 查找组件
	for _, comp := range spec.Components {
		if comp.Name == name {
			fmt.Printf("=== 组件 '%s' 的详细信息 ===\n", name)
			fmt.Printf("名称: %s\n", comp.Name)
			fmt.Printf("依赖版本: %s\n", comp.Version)

			// 显示版本信息
			if localVer, err := getLocalVersion(comp.Name); err == nil {
				fmt.Printf("本地版本: %d.%d.%d\n", localVer.Major, localVer.Minor, localVer.Micro)
			} else {
				fmt.Printf("本地版本: 未安装\n")
			}

			remoteVers, err := getRemoteVersion(comp.Name)
			if err == nil {
				fmt.Printf("服务器最新版本: %s\n", utils.PrintVersion(remoteVers.Newest.VersionId))
			} else {
				fmt.Printf("服务器最新版本: 无法获取\n")
			}

			// 显示升级配置
			if comp.Upgrade != nil {
				fmt.Printf("升级模式: %s\n", comp.Upgrade.Mode)
				if comp.Upgrade.Lowest != "" {
					fmt.Printf("最低支持版本: %s\n", comp.Upgrade.Lowest)
				}
				if comp.Upgrade.Highest != "" {
					fmt.Printf("最高支持版本: %s\n", comp.Upgrade.Highest)
				}
			}

			return nil
		}
	}

	return fmt.Errorf("未找到名为 '%s' 的组件", name)
}

func init() {
	root.RootCmd.AddCommand(listCmd)
}
