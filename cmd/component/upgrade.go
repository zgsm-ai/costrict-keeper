package component

import (
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/utils"
	"fmt"

	"github.com/spf13/cobra"
)

var optComponent string
var optVersion string

var upgradeCmd = &cobra.Command{
	Use:   "upgrade {component | -n component}",
	Short: "Upgrade specified component",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Determine component name: prioritize positional arguments, then use command line arguments
		component := optComponent
		if len(args) > 0 && args[0] != "" {
			component = args[0]
		}

		if component == "" {
			fmt.Println("Error: Component name must be specified")
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
		BaseUrl:     config.Get().Cloud.UpgradeUrl,
	}
	cfg.Correct()

	var specVer *utils.VersionNumber
	if version != "" {
		v, err := utils.ParseVersion(version)
		if err != nil {
			return fmt.Errorf("invalid version number: %s", version)
		}
		specVer = &v
	}

	retVer, upgraded, err := utils.UpgradePackage(cfg, specVer)
	if err != nil {
		fmt.Printf("The '%s' upgrade failed: %v", component, err)
		return err
	}
	if !upgraded {
		fmt.Printf("The '%s' version is up to date\n", component)
	} else {
		fmt.Printf("The '%s' is upgraded to version %s\n", component, utils.PrintVersion(retVer))
	}
	return nil
}

func init() {
	upgradeCmd.Flags().SortFlags = false
	upgradeCmd.Flags().StringVarP(&optVersion, "version", "v", "", "Specify the target version to upgrade")
	upgradeCmd.Flags().StringVarP(&optComponent, "component", "n", "", "Specify the component name to upgrade")
	componentCmd.AddCommand(upgradeCmd)
}
