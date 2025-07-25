package upgrade

import (
	"costrict-host/cmd/root"
	"costrict-host/internal/utils"
	"fmt"

	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [component]",
	Short: "升级指定组件",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := upgradeComponent(args[0], optVersion); err != nil {
			fmt.Println(err)
		}
	},
}

var optVersion string

func upgradeComponent(component string, version string) error {
	cfg := &utils.UpgradeConfig{
		PackageName: component,
	}

	// 假设当前版本号为1.0.0，实际项目中应从配置或系统中获取
	curVer, _ := utils.ParseVersion("1.0.0")

	var specVer *utils.VersionNumber
	if version != "" {
		v, err := utils.ParseVersion(version)
		if err != nil {
			return fmt.Errorf("无效的版本号: %s", version)
		}
		specVer = &v
	}

	if err := utils.UpgradePackage(cfg, curVer, specVer); err != nil {
		return fmt.Errorf("升级组件%s失败: %v", component, err)
	}

	fmt.Printf("组件 %s 升级成功\n", component)
	return nil
}

func init() {
	upgradeCmd.Flags().StringVarP(&optVersion, "version", "v", "", "指定要升级的目标版本")
	root.RootCmd.AddCommand(upgradeCmd)
}
