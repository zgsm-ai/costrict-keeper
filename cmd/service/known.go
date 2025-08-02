package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"costrict-keeper/cmd/root"
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/models"
	"costrict-keeper/services"

	"github.com/spf13/cobra"
)

// SystemKnowledge 包含系统知识的结构
type SystemKnowledge struct {
	Logs     models.LogKnowledge       `json:"logs"`
	Services []models.ServiceKnowledge `json:"services"`
}

var outputPath string

var knownCmd = &cobra.Command{
	Use:   "known",
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
 * err := exportStatus(context.Background(), "")
 * if err != nil {
 *     logger.Fatal(err)
 * }
 */
func exportStatus(ctx context.Context, customOutputPath string) error {
	manager := services.NewServiceManager()

	// 获取所有信息
	serviceSpecs := manager.GetServices()

	// 转换ServiceSpecification为ServiceKnowledge
	serviceKnowledge := make([]models.ServiceKnowledge, 0)
	for _, svc := range serviceSpecs {
		// 检查服务是否正在运行
		isHealthy := manager.IsServiceHealthy(svc.Name)
		status := "norun"
		if isHealthy {
			status = "running"
		}

		// 获取服务版本信息（从组件信息中获取）
		components, err := manager.GetComponents()
		version := "unknown"
		if err == nil {
			for _, comp := range components {
				if comp.Name == svc.Name {
					version = comp.Version
					break
				}
			}
		}

		serviceKnowledge = append(serviceKnowledge, models.ServiceKnowledge{
			Name:       svc.Name,
			Version:    version,
			Installed:  true, // 假设服务已安装
			Startup:    svc.Startup,
			Status:     status,
			Protocol:   svc.Protocol,
			Port:       svc.Port,
			Command:    svc.Command,
			Metrics:    svc.Metrics,
			Healthy:    svc.Healthy,
			Accessible: svc.Accessible,
		})
	}

	// 构建日志知识
	logKnowledge := models.LogKnowledge{
		Dir:   config.Config.Directory.Logs,
		Level: config.Config.Log.Level,
	}

	// 构建要导出的信息结构
	info := SystemKnowledge{
		Logs:     logKnowledge,
		Services: serviceKnowledge,
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
	knownCmd.Flags().StringVarP(&outputPath, "output", "o", "", "指定输出文件路径")
	root.RootCmd.AddCommand(knownCmd)
}
