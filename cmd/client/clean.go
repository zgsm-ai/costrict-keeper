package client

import (
	"costrict-keeper/cmd/root"
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/env"
	"costrict-keeper/internal/logger"
	"costrict-keeper/internal/utils"
	"costrict-keeper/services"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up all services, tunnels, processes and cache",
	Long:  `Stop all services, terminate all tunnels, kill specified processes and clean up .costrict cache directory`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cleanAll(); err != nil {
			logger.Fatal("Clean failed:", err)
		}
		fmt.Println("Clean completed successfully")
	},
}

/**
 * Clean up all services, tunnels, processes and cache
 * @returns {error} Returns error if any cleanup step fails, nil on success
 * @description
 * - Stops all running services
 * - Closes all active tunnels
 * - Kills processes with names: costrict, codebase-indexer, cotun
 * - Cleans up .costrict cache directory
 * @throws
 * - Service stop errors
 * - Tunnel close errors
 * - Process kill errors
 * - Cache cleanup errors
 * @example
 * err := cleanAll()
 * if err != nil {
 *     logger.Fatal(err)
 * }
 */
func cleanAll() error {
	fmt.Println("Starting cleanup process...")

	if err := utils.KillSpecifiedProcess(services.COSTRICT_NAME); err != nil {
		logger.Errorf("Failed to kill specified process: %v", err)
		return fmt.Errorf("failed to kill specified process: %v", err)
	}
	fmt.Println("Successfully killed the 'costrict'")

	// 1. 停止所有服务
	stopAllServices()
	fmt.Println("All services stopped successfully")

	// 2. 杀死所有组件/服务的进程
	targetProcesses := []string{"costrict"}
	for _, svc := range config.Spec().Components {
		targetProcesses = append(targetProcesses, svc.Name)
	}
	if err := utils.KillSpecifiedProcesses(targetProcesses); err != nil {
		logger.Errorf("Failed to kill specified processes: %v", err)
		return fmt.Errorf("failed to kill specified processes: %v", err)
	}
	fmt.Println("Specified processes killed successfully")

	// 3. 清理.costrict目录下的cache目录
	if err := cleanCacheDirectory(); err != nil {
		logger.Errorf("Failed to clean cache directory: %v", err)
		return fmt.Errorf("failed to clean cache directory: %v", err)
	}
	fmt.Println("Cache directory cleaned successfully")
	return nil
}

/**
 * Stop all running services
 * @returns {error} Returns error if service stopping fails, nil on success
 * @description
 * - Gets service manager instance
 * - Calls StopAll to stop all services
 * @throws
 * - Service manager access errors
 * - Service stop errors
 */
func stopAllServices() {
	logger.Info("Stopping all services...")
	serviceManager := services.GetServiceManager()
	serviceManager.StopAll()
}

/**
 * Clean up .costrict cache directory
 * @returns {error} Returns error if cache cleanup fails, nil on success
 * @description
 * - Gets .costrict directory path from config
 * - Constructs cache directory path
 * - Removes cache directory and all its contents
 * @throws
 * - Directory path construction errors
 * - Directory removal errors
 */
func cleanCacheDirectory() error {
	logger.Info("Cleaning up cache directory...")

	// 获取.costrict目录路径
	costrictDir := env.CostrictDir
	if costrictDir == "" {
		return fmt.Errorf("failed to get .costrict directory path")
	}

	// 构建cache目录路径
	cacheDir := filepath.Join(costrictDir, "cache")
	logger.Infof("Cache directory path: %s", cacheDir)

	// 检查cache目录是否存在
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		logger.Info("Cache directory does not exist, skipping cleanup")
		return nil
	}

	// 删除cache/services目录及其所有内容
	servicesDir := filepath.Join(cacheDir, "services")
	logger.Infof("Removing cache directory: %s", servicesDir)
	if err := os.RemoveAll(servicesDir); err != nil {
		return fmt.Errorf("failed to remove cache directory %s: %v", servicesDir, err)
	}
	logger.Infof("Successfully removed cache directory: %s", servicesDir)

	// 删除cache/tunnels目录及其所有内容
	tunnelsDir := filepath.Join(cacheDir, "tunnels")
	logger.Infof("Removing cache directory: %s", tunnelsDir)
	if err := os.RemoveAll(tunnelsDir); err != nil {
		return fmt.Errorf("failed to remove cache directory %s: %v", tunnelsDir, err)
	}
	logger.Infof("Successfully removed cache directory: %s", tunnelsDir)

	// 删除run目录及其所有内容
	runDir := filepath.Join(env.CostrictDir, "run")
	logger.Infof("Removing run directory: %s", runDir)
	if err := os.RemoveAll(runDir); err != nil {
		return fmt.Errorf("failed to remove run directory %s: %v", runDir, err)
	}
	logger.Infof("Successfully removed run directory: %s", runDir)
	return nil
}

func init() {
	root.RootCmd.AddCommand(cleanCmd)
}
