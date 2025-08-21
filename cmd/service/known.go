package service

import (
	"context"
	"fmt"
	"path/filepath"

	"costrict-keeper/internal/env"
	"costrict-keeper/internal/models"
	"costrict-keeper/services"

	"github.com/spf13/cobra"
)

// SystemKnowledge contains system knowledge structure
type SystemKnowledge struct {
	Logs     models.LogKnowledge       `json:"logs"`
	Services []models.ServiceKnowledge `json:"services"`
}

var outputPath string

var knownCmd = &cobra.Command{
	Use:   "known",
	Short: "Output all service information to well-known.json file",
	Long:  "Collect all component, service and endpoint information and output it to specified file. If output path is not specified, default output to <user directory>/.costrict/share/.well-known.json",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		exportKnowledge(context.Background(), outputPath)
	},
}

/**
 * Export service known to well-known.json file
 * @param {context.Context} ctx - Context for request cancellation and timeout
 * @param {string} customOutputPath - Custom output file path, if empty uses default path
 * @returns {error} Returns error if export fails, nil on success
 * @description
 * - Creates new service manager instance
 * - Collects all components, services and endpoints information
 * - Builds WellKnownInfo structure with timestamp
 * - Writes data to JSON file at specified or default location
 * - Creates necessary directories if they don't exist
 * @throws
 * - Component/service information retrieval errors
 * - Directory creation errors
 * - JSON encoding errors
 * - File writing errors
 * @example
 * err := exportKnowledge(context.Background(), "")
 * if err != nil {
 *     logger.Fatal(err)
 * }
 */
func exportKnowledge(ctx context.Context, customOutputPath string) error {
	manager := services.GetServiceManager()

	var outputFile string
	if customOutputPath != "" {
		outputFile = customOutputPath
	} else {
		outputFile = filepath.Join(env.CostrictDir, "share", ".well-known.json")
	}

	if err := manager.ExportKnowledge(outputFile); err != nil {
		fmt.Printf("Failed to export status information: %s\n", err.Error())
		return err
	}
	fmt.Printf("Status information has been exported to: %s\n", outputFile)
	return nil
}

func init() {
	knownCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Specify output file path")
	serviceCmd.AddCommand(knownCmd)
}
