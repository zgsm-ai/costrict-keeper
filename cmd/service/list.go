package service

import (
	"context"
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/utils"
	"costrict-keeper/services"
	"fmt"
	"os"

	"github.com/iancoleman/orderedmap"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [service name]",
	Short: "View service status",
	Long:  "View running status of all services. If service name is specified, only show detailed information of that service.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := showServiceStatus(context.Background(), args); err != nil {
			fmt.Println(err)
		}
	},
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
	// Load system configuration
	if err := config.LoadSpec(); err != nil {
		fmt.Printf("Failed to load system configuration: %v\n", err)
		return err
	}
	manager := services.GetServiceManager()
	if len(args) == 0 {
		// Display all services status
		return showAllServicesStatus(manager)
	} else {
		// Display detailed information of specified service
		return showSpecificServiceStatus(manager, args[0])
	}
}

type Service_Columns struct {
	Name      string
	Protocol  string
	Port      int
	Startup   string
	Status    string
	Pid       int
	Healthy   string
	StartTime string
}

/**
 * Show all services status with detailed information
 * @param {spec *models.SystemSpecification} System configuration
 * @returns {error} Returns error if showing status fails, nil on success
 * @description
 * - Lists all services with status information
 * - Uses tabwriter for formatted output
 */
func showAllServicesStatus(manager *services.ServiceManager) error {
	svcs := manager.GetInstances(true)
	if len(svcs) == 0 {
		fmt.Println("No services found")
		return nil
	}

	var dataList []*orderedmap.OrderedMap
	for _, svc := range svcs {
		row := Service_Columns{}
		row.Name = svc.Spec.Name
		row.Protocol = svc.Spec.Protocol
		row.Port = svc.Port
		row.Startup = svc.Spec.Startup
		row.Status = svc.Status
		row.Pid = svc.Pid
		row.StartTime = svc.StartTime
		if running, err := utils.IsProcessRunning(row.Pid); err == nil && running {
			row.Healthy = "Y"
		} else {
			row.Healthy = "N"
		}

		recordMap, _ := utils.StructToOrderedMap(row)
		dataList = append(dataList, recordMap)
	}
	utils.PrintFormat(dataList)
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
func showSpecificServiceStatus(manager *services.ServiceManager, name string) error {
	svc := manager.GetSelf()
	if name != services.COSTRICT_NAME {
		svc = manager.GetInstance(name)
		if svc == nil {
			fmt.Printf("Service named '%s' not found\n", name)
			return os.ErrNotExist
		}
	}
	detail := svc.GetDetail()
	component := detail.Component

	fmt.Printf("=== Detailed information of service '%s' ===\n", name)
	fmt.Printf("Name: %s\n", svc.Name)
	fmt.Printf("Running status: %s\n", svc.Status)
	fmt.Printf("Port: %d\n", svc.Port)
	fmt.Printf("PID: %d\n", svc.Pid)
	fmt.Printf("Start time: %s\n", svc.StartTime)
	fmt.Printf("Startup command: %s\n", detail.Process.Command)
	fmt.Printf("Startup args: %+v\n", detail.Process.Args)

	fmt.Printf("Startup mode: %s\n", svc.Spec.Startup)
	fmt.Printf("Protocol: %s\n", svc.Spec.Protocol)
	if svc.Spec.Metrics != "" {
		fmt.Printf("Metrics endpoint: %s\n", svc.Spec.Metrics)
	}
	if svc.Spec.Accessible != "" {
		fmt.Printf("Access permission: %s\n", svc.Spec.Accessible)
	}
	// Display version information
	if component != nil {
		fmt.Printf("Local version: %s\n", component.LocalVersion)
		fmt.Printf("Latest server version: %s\n", component.RemoteVersion)
	} else {
		fmt.Printf("Local version: Not installed\n")
		fmt.Printf("Latest server version: Unable to retrieve\n")
	}

	// Display endpoint URL
	if svc.Spec.Protocol != "" && svc.Port > 0 {
		endpointURL := fmt.Sprintf("%s://localhost:%d", svc.Spec.Protocol, svc.Port)
		fmt.Printf("Access URL: %s\n", endpointURL)
	}

	return nil
}

func init() {
	serviceCmd.AddCommand(listCmd)
}
