package tunnel

import (
	"context"
	"costrict-keeper/internal/models"
	"costrict-keeper/internal/utils"
	"costrict-keeper/services"
	"fmt"
	"time"

	"github.com/iancoleman/orderedmap"
	"github.com/spf13/cobra"
)

var (
	listApp  string
	listPort int
)
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active tunnels",
	Run: func(cmd *cobra.Command, args []string) {
		if err := listTunnels(context.Background()); err != nil {
			fmt.Println(err)
		}
	},
}

/**
 * List tunnel information with filtering
 * @param {context.Context} ctx - Context for request cancellation and timeout
 * @returns {error} Returns error if listing fails, nil on success
 * @description
 * - Lists all active tunnels with filtering options
 * - Filters by app name and/or port if specified
 * - Uses utils.PrintFormat for formatted output
 * @throws
 * - Tunnel service errors
 */
func listTunnels(ctx context.Context) error {
	tunnelSvc := services.GetTunnelManager()
	tunnels := tunnelSvc.ListTunnels()

	// Filter tunnels based on app and port parameters
	var filteredTunnels []*models.Tunnel
	for _, tunnel := range tunnels {
		// If app parameter is specified, only show matching applications
		if listApp != "" && tunnel.Name != listApp {
			continue
		}
		// If port parameter is specified, only show matching ports
		if listPort != 0 && tunnel.LocalPort != listPort {
			continue
		}
		filteredTunnels = append(filteredTunnels, tunnel)
	}

	if len(filteredTunnels) == 0 {
		if listApp != "" || listPort != 0 {
			fmt.Printf("No matching tunnels found")
			if listApp != "" {
				fmt.Printf(" (app: %s", listApp)
				if listPort != 0 {
					fmt.Printf(", port: %d", listPort)
				}
				fmt.Print(")")
			} else if listPort != 0 {
				fmt.Printf(" (port: %d)", listPort)
			}
			fmt.Println()
		} else {
			fmt.Println("No active tunnels")
		}
		return nil
	}

	return listAllTunnels(filteredTunnels)
}

/**
 *	Fields displayed in list format
 */
type Tunnel_Columns struct {
	Name        string `json:"name"`
	LocalPort   int    `json:"local_port"`
	MappingPort int    `json:"mapping_port"`
	Status      string `json:"status"`
	Pid         int    `json:"pid"`
	Healthy     string `json:"healthy"`
	CreateTime  string `json:"create_time"`
}

/**
 * List all tunnels with formatted output
 * @param {[]*models.Tunnel} tunnels - List of tunnels to display
 * @returns {error} Returns error if listing fails, nil on success
 * @description
 * - Formats tunnel data for display
 * - Uses utils.PrintFormat for table output
 */
func listAllTunnels(tunnels []*models.Tunnel) error {
	// Format output package list
	var dataList []*orderedmap.OrderedMap
	for _, tunnel := range tunnels {
		row := Tunnel_Columns{}
		row.Name = tunnel.Name
		row.LocalPort = tunnel.LocalPort
		row.MappingPort = tunnel.MappingPort
		row.Pid = tunnel.Pid
		row.Status = string(tunnel.Status)
		row.CreateTime = tunnel.CreatedTime.Format(time.RFC3339)

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

func init() {
	listCmd.Flags().SortFlags = false
	listCmd.Flags().StringVarP(&listApp, "app", "a", "", "App name")
	listCmd.Flags().IntVarP(&listPort, "port", "p", 0, "Port number")
	tunnelCmd.AddCommand(listCmd)
}
