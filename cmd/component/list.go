package component

import (
	"context"
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/models"
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
	// Load system configuration
	if err := config.LoadSpec(); err != nil {
		return fmt.Errorf("Failed to load system configuration: %v", err)
	}

	if len(args) == 0 {
		// List all components information
		return listAllComponents(config.Spec())
	} else {
		// List detailed information of specified component
		return listSpecificComponent(config.Spec(), args[0])
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
func listAllComponents(spec *models.SystemSpecification) error {
	if len(spec.Components) == 0 {
		fmt.Println("No components found")
		return nil
	}

	manager := services.GetComponentManager()
	// Format output package list
	var dataList []*orderedmap.OrderedMap
	for _, comp := range manager.GetComponents() {
		row := Component_Columns{}
		row.Name = comp.Spec.Name
		row.Path = "-"
		row.Local = comp.LocalVersion
		row.Remote = comp.RemoteVersion
		if comp.RemotePlatform != nil {
			row.Path = comp.RemotePlatform.Newest.AppUrl
		}

		recordMap, _ := utils.StructToOrderedMap(row)
		dataList = append(dataList, recordMap)
	}

	utils.PrintFormat(dataList)
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
	manager := services.GetComponentManager()
	for _, comp := range spec.Components {
		if comp.Name != name {
			continue
		}
		fmt.Printf("=== Detailed information of component '%s' ===\n", name)
		fmt.Printf("Name: %s\n", comp.Name)
		fmt.Printf("Required version: %s\n", comp.Version)
		component := manager.GetComponent(comp.Name)

		// Display version information
		if component != nil {
			fmt.Printf("Local version: %s\n", component.LocalVersion)
			fmt.Printf("Latest server version: %s\n", component.RemoteVersion)
		} else {
			fmt.Printf("Local version: Not installed\n")
			fmt.Printf("Latest server version: Unable to retrieve\n")
		}

		// Display upgrade configuration
		if comp.Upgrade != nil {
			fmt.Printf("Upgrade mode: %s\n", comp.Upgrade.Mode)
			if comp.Upgrade.Lowest != "" {
				fmt.Printf("Minimum supported version: %s\n", comp.Upgrade.Lowest)
			}
			if comp.Upgrade.Highest != "" {
				fmt.Printf("Maximum supported version: %s\n", comp.Upgrade.Highest)
			}
		}
		return nil
	}

	return fmt.Errorf("Component named '%s' not found", name)
}

func init() {
	componentCmd.AddCommand(listCmd)
}
