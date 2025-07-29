package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"costrict-keeper/cmd/root"
	"costrict-keeper/internal/models"
	"costrict-keeper/services"

	"github.com/spf13/cobra"
)

// WellKnownInfo 包含所有服务信息的结构
type WellKnownInfo struct {
	Timestamp  string                   `json:"timestamp"`
	Components []models.ComponentInfo   `json:"components"`
	Services   []models.ServiceConfig   `json:"services"`
	Endpoints  []models.ServiceEndpoint `json:"endpoints"`
}

var outputPath string

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "输出所有服务信息到 well-known.json 文件",
	Long:  "收集所有组件、服务和端点信息，并将其输出到指定文件中。如果未指定输出路径，则默认输出到 %APPDATA%/.costrict/share/.well-known.json",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := exportStatus(context.Background(), outputPath); err != nil {
			fmt.Printf("导出状态信息失败: %v\n", err)
		} else {
			fmt.Println("服务状态信息已成功导出")
		}
	},
}

/**
 * Export service status to well-known.json file
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
 * err := exportStatus(context.Background(), "")
 * if err != nil {
 *     log.Fatal(err)
 * }
 */
func exportStatus(ctx context.Context, customOutputPath string) error {
	manager := services.NewServiceManager()

	// 获取所有信息
	components, err := manager.GetComponents()
	if err != nil {
		return fmt.Errorf("获取组件信息失败: %v", err)
	}

	services := manager.GetServices()
	endpoints := manager.GetEndpoints()

	// 构建要导出的信息结构
	info := WellKnownInfo{
		Timestamp:  time.Now().Format(time.RFC3339),
		Components: components,
		Services:   services,
		Endpoints:  endpoints,
	}

	var outputFile string
	if customOutputPath != "" {
		outputFile = customOutputPath
	} else {
		// 生成默认输出文件路径
		appDataDir := os.Getenv("APPDATA")
		if appDataDir == "" {
			return fmt.Errorf("无法获取 APPDATA 环境变量")
		}
		outputDir := filepath.Join(appDataDir, ".costrict", "share")
		outputFile = filepath.Join(outputDir, ".well-known.json")
	}

	// 确保目录存在
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 将信息编码为 JSON
	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 编码失败: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Printf("状态信息已导出到: %s\n", outputFile)
	return nil
}

func init() {
	statusCmd.Flags().StringVarP(&outputPath, "output", "o", "", "指定输出文件路径")
	root.RootCmd.AddCommand(statusCmd)
}
