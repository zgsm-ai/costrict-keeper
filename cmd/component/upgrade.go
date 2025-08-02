package component

import (
	"costrict-keeper/cmd/root"
	"costrict-keeper/internal/utils"
	"fmt"

	"github.com/spf13/cobra"
)

var optComponent string
var optVersion string

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [component]",
	Short: "升级指定组件",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// 确定组件名称：优先使用位置参数，其次使用命令行参数
		component := optComponent
		if len(args) > 0 && args[0] != "" {
			component = args[0]
		}

		if component == "" {
			fmt.Println("错误：必须指定组件名称")
			return
		}

		if err := upgradeComponent(component, optVersion); err != nil {
			fmt.Println(err)
		}
	},
}

func upgradeComponent(component string, version string) error {
	cfg := utils.UpgradeConfig{
		PackageName: component,
	}
	cfg.Correct()
	curVer, _ := utils.GetLocalVersion(cfg)

	var specVer *utils.VersionNumber
	if version != "" {
		v, err := utils.ParseVersion(version)
		if err != nil {
			return fmt.Errorf("无效的版本号: %s", version)
		}
		specVer = &v
	}

	retVer, err := utils.UpgradePackage(cfg, curVer, specVer)
	if err != nil {
		fmt.Printf("The '%s' upgrade failed: %v", component, err)
		return err
	}
	if utils.CompareVersion(retVer, curVer) == 0 {
		fmt.Printf("The '%s' version is up to date\n", component)
	} else {
		fmt.Printf("The '%s' is upgraded to version %s\n", component, utils.PrintVersion(retVer))
	}
	return nil
}

func init() {
	upgradeCmd.Flags().SortFlags = false
	upgradeCmd.Flags().StringVarP(&optVersion, "version", "v", "", "指定要升级的目标版本")
	upgradeCmd.Flags().StringVarP(&optComponent, "component", "c", "", "指定要升级的组件名称")
	root.RootCmd.AddCommand(upgradeCmd)
}
