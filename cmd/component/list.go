package component

import (
	"context"
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/env"
	"costrict-keeper/internal/utils"
	"costrict-keeper/services"
	"fmt"

	"github.com/iancoleman/orderedmap"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [component name]",
	Short: "List information of all components",
	Long:  "List information of all components, including local version and latest server version. If component name is specified, only show detailed information of that component.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.LoadLocalSpec(); err != nil {
			fmt.Printf("Costrict is uninitialized")
			return
		}

		listInfo(context.Background(), args)
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
func listInfo(ctx context.Context, args []string) {
	fmt.Printf("------------------------------------------\n")
	fmt.Printf("Base URL: %s\n", config.GetBaseURL())
	fmt.Printf("Local: %s\n", env.CostrictDir)
	fmt.Printf("------------------------------------------\n")
	if len(args) == 0 {
		// List all components information
		listAllComponents()
	} else {
		// List detailed information of specified component
		listSpecificComponent(args[0])
	}
}

/**
 *	Fields displayed in list format
 */
type Component_Columns struct {
	Name        string `json:"name"`
	Local       string `json:"local"`
	Remote      string `json:"remote"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

/**
 * List all components with detailed information
 * @param {spec *models.SystemSpecification} System configuration
 * @returns {error} Returns error if listing fails, nil on success
 * @description
 * - Lists components with local and remote versions
 * - Uses tabwriter for formatted output
 */
func listAllComponents() {
	manager := services.GetComponentManager()
	manager.Init()
	components := manager.GetComponents(true, true)
	if len(components) == 0 {
		fmt.Println("No components found")
		return
	}
	var dataList []*orderedmap.OrderedMap
	for _, ci := range components {
		cpn := ci.GetDetail()
		row := Component_Columns{}
		row.Name = cpn.Spec.Name
		row.Path = "-"
		row.Local = cpn.Local.Version
		row.Remote = cpn.Remote.Newest
		if cpn.Installed {
			row.Path = cpn.Local.FileName
			row.Description = cpn.Local.Description
		}
		recordMap, _ := utils.StructToOrderedMap(row)
		dataList = append(dataList, recordMap)
	}

	utils.PrintFormat(dataList)
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
func listSpecificComponent(name string) {
	manager := services.GetComponentManager()
	manager.Init()

	ci := manager.GetComponent(name)
	if ci == nil {
		fmt.Printf("Component '%s' not found\n", name)
		return
	}
	cpn := ci.GetDetail()
	spec := &cpn.Spec
	fmt.Printf("=== Detailed information of component '%s' ===\n", name)
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Need upgrade: %v\n", cpn.NeedUpgrade)

	// Display version information
	if cpn.Local.Version != "" {
		fmt.Printf("Local version: %s\n", cpn.Local.Version)
	} else {
		fmt.Printf("Local version: Not installed\n")
	}
	if cpn.Remote.Newest != "" {
		fmt.Printf("Latest server version: %s\n", cpn.Remote.Newest)
	} else {
		fmt.Printf("Latest server version: Unable to retrieve\n")
	}

	// Display upgrade configuration
	if spec.Upgrade != nil {
		fmt.Printf("Upgrade mode: %s\n", spec.Upgrade.Mode)
		if spec.Upgrade.Lowest != "" {
			fmt.Printf("Minimum supported version: %s\n", spec.Upgrade.Lowest)
		}
		if spec.Upgrade.Highest != "" {
			fmt.Printf("Maximum supported version: %s\n", spec.Upgrade.Highest)
		}
	}
}

func init() {
	componentCmd.AddCommand(listCmd)
}
