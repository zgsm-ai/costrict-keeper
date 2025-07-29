package services

import (
	"context"
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/models"
	"costrict-keeper/internal/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
)

type ServiceManager struct {
	Services    []models.ServiceConfig
	upgradeFn   func(*utils.UpgradeConfig, utils.VersionNumber, *utils.VersionNumber) error
	runningSvcs map[string]*ServiceInstance
}

/**
 * Get all registered services
 * @returns {[]models.ServiceConfig} Returns slice of service configurations
 * @description
 * - Returns current list of managed services
 * - Includes service names, versions, protocols, ports and startup commands
 */
func (sm *ServiceManager) GetServices() []models.ServiceConfig {
	return sm.Services
}

/**
 * Get all components derived from services
 * @returns {([]models.ComponentInfo, error)} Returns slice of component information and error if any
 * @description
 * - Converts service configurations to component information
 * - Each service becomes a component with name, version and path
 * - Returns empty slice if no services exist
 * @throws
 * - Component conversion errors
 */
func (sm *ServiceManager) GetComponents() ([]models.ComponentInfo, error) {
	// 实现获取组件列表逻辑
	components := make([]models.ComponentInfo, 0)
	// 获取服务作为组件信息
	for _, svc := range sm.Services {
		components = append(components, models.ComponentInfo{
			Name:    svc.Name,
			Version: "unknown", // 新的结构体中没有版本信息
			Path:    svc.Command,
		})
	}
	return components, nil
}

/**
 * Upgrade specified component to latest version
 * @param {string} name - Name of the component to upgrade
 * @returns {error} Returns error if upgrade fails, nil on success
 * @description
 * - Finds service configuration by component name
 * - Parses highest version from service configuration
 * - Executes upgrade function with component configuration
 * @throws
 * - Service not found errors
 * - Version parsing errors
 * - Upgrade execution errors
 */
func (sm *ServiceManager) UpgradeComponent(name string) error {
	// 实现组件升级逻辑
	// 获取组件配置
	var svc *models.ServiceConfig
	for _, s := range sm.Services {
		if s.Name == name {
			svc = &s
			break
		}
	}
	if svc == nil {
		return fmt.Errorf("service %s not found", name)
	}

	// 解析版本号 - 由于新结构体中没有版本信息，使用默认版本
	ver, err := utils.ParseVersion("1.0.0")
	if err != nil {
		return err
	}
	upgradeCfg := utils.UpgradeConfig{PackageName: name}
	return sm.upgradeFn(&upgradeCfg, ver, nil)
}

/**
 * Check if service is healthy and running
 * @param {string} name - Name of the service to check
 * @returns {bool} Returns true if service is healthy, false otherwise
 * @description
 * - Checks if service instance exists in running services map
 * - Verifies process state is not exited
 * - Checks if service port is available
 * - Returns false if service is not found or unhealthy
 */
func (sm *ServiceManager) IsServiceHealthy(name string) bool {
	// 实现服务健康检查
	// 检查服务进程状态
	if instance, ok := sm.runningSvcs[name]; ok {
		// 检查进程是否正在运行
		if instance.Command.Process != nil {
			// 检查进程是否存在且未退出
			process, err := os.FindProcess(instance.PID)
			if err != nil {
				return false
			}

			// 发送信号0来检查进程是否存在（不会实际发送信号）
			if err := process.Signal(syscall.Signal(0)); err != nil {
				// 进程可能已经退出
				return false
			}

			// 检查服务端口是否可访问
			for _, svc := range sm.Services {
				if svc.Name == name {
					// 如果端口不可用（已被占用），说明服务正在监听
					if svc.Port > 0 {
						return !utils.CheckPortAvailable(svc.Port)
					}
					return false
				}
			}
		}
	}
	return false
}

/**
 * Get all service endpoints with health status
 * @returns {[]models.ServiceEndpoint} Returns slice of service endpoints
 * @description
 * - Creates endpoint for each service with name and URL
 * - Includes health status for each endpoint
 * - URL format: protocol://localhost:port
 * - Returns empty slice if no services exist
 */
func (sm *ServiceManager) GetEndpoints() []models.ServiceEndpoint {
	endpoints := make([]models.ServiceEndpoint, 0)
	for _, svc := range sm.Services {
		// 构建URL，确保端口有效
		url := ""
		if svc.Protocol != "" && svc.Port > 0 {
			url = fmt.Sprintf("%s://localhost:%d", svc.Protocol, svc.Port)
		}
		endpoints = append(endpoints, models.ServiceEndpoint{
			Name:    svc.Name,
			URL:     url,
			Healthy: sm.IsServiceHealthy(svc.Name),
		})
	}
	return endpoints
}

type ServiceInstance struct {
	PID     int
	Command *exec.Cmd
	Status  string
}

/**
 * Load remote services configuration from URL
 * @param {string} url - URL of the remote configuration file
 * @returns {(*RemoteServicesConfig, error)} Returns configuration struct and error if any
 * @description
 * - Makes HTTP GET request to specified URL
 * - Validates HTTP response status code
 * - Reads response body and parses JSON
 * - Returns unmarshaled configuration structure
 * @throws
 * - HTTP request errors
 * - HTTP status code errors
 * - Response body reading errors
 * - JSON unmarshaling errors
 * @example
 * config, err := LoadRemoteServicesConfig("https://example.com/config.json")
 * if err != nil {
 *     log.Fatal(err)
 * }
 */
func LoadRemoteServicesConfig(url string) (*models.SubsystemConfig, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote config: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var config models.SubsystemConfig
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}

func (sm *ServiceManager) LoadConfig() {
	// 加载远程服务配置，如果无法加载则使用本地配置
	servicesAddr := fmt.Sprintf("%s/costrict-keeper/system-spec.json", config.Config.Upgrade.BaseUrl)
	remoteCfg, err := LoadRemoteServicesConfig(servicesAddr)
	if err != nil {
		log.Printf("fetch config failed: %v", err)
		return
	} else {
		sm.Services = remoteCfg.Services
	}
}

func (sm *ServiceManager) StartAll(ctx context.Context) error {
	for _, svc := range sm.Services {
		// 只启动启动模式为 "always" 的服务
		if svc.Startup == "always" {
			if err := sm.StartService(ctx, svc.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sm *ServiceManager) StartService(ctx context.Context, name string) error {
	var svcConfig *models.ServiceConfig
	for _, svc := range sm.Services {
		if svc.Name == name {
			svcConfig = &svc
			break
		}
	}

	if svcConfig == nil {
		return fmt.Errorf("service %s not found", name)
	}

	// 检查服务启动模式
	if svcConfig.Startup == "none" {
		return fmt.Errorf("service %s is configured not to start automatically", name)
	}

	// 检查服务是否已经在运行
	if _, ok := sm.runningSvcs[name]; ok {
		return fmt.Errorf("service %s is already running", name)
	}

	// 首先尝试升级服务到最新版本
	ver, err := utils.ParseVersion("1.0.0") // 使用默认版本
	if err != nil {
		return fmt.Errorf("failed to parse version for service %s: %v", name, err)
	}
	upgradeCfg := utils.UpgradeConfig{
		PackageName: name,
		TargetName:  svcConfig.Command,
	}
	if err := sm.upgradeFn(&upgradeCfg, ver, nil); err != nil {
		return fmt.Errorf("failed to upgrade service %s: %v", name, err)
	}

	// 启动服务进程
	cmd := exec.CommandContext(ctx, svcConfig.Command)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start service %s: %v", name, err)
	}

	// 创建服务实例并保存到运行服务列表中
	instance := &ServiceInstance{
		PID:     cmd.Process.Pid,
		Command: cmd,
		Status:  "running",
	}
	sm.runningSvcs[name] = instance

	return nil
}

func (sm *ServiceManager) RestartService(ctx context.Context, name string) error {
	if err := sm.StopService(name); err != nil {
		return err
	}
	return sm.StartService(ctx, name)
}

func (sm *ServiceManager) StopService(name string) error {
	// 检查服务是否存在
	var svcConfig *models.ServiceConfig
	for _, svc := range sm.Services {
		if svc.Name == name {
			svcConfig = &svc
			break
		}
	}

	if svcConfig == nil {
		return fmt.Errorf("service %s not found", name)
	}

	// 检查服务是否在运行
	if instance, ok := sm.runningSvcs[name]; ok {
		// 如果进程还在运行，尝试终止它
		if instance.Command.Process != nil {
			if err := instance.Command.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process for service %s: %v", name, err)
			}
			// 等待进程退出
			instance.Command.Wait()
		}
		// 从运行服务列表中移除
		delete(sm.runningSvcs, name)
	}

	return nil
}

func (sm *ServiceManager) CheckServices() error {
	for _, svc := range sm.Services {
		if !sm.IsServiceHealthy(svc.Name) {
			// 如果服务不健康，尝试重启
			if err := sm.RestartService(context.Background(), svc.Name); err != nil {
				return fmt.Errorf("failed to restart service %s: %v", svc.Name, err)
			}
		}
	}
	return nil
}

func NewServiceManager() *ServiceManager {
	sm := &ServiceManager{
		upgradeFn:   utils.UpgradePackage,
		runningSvcs: make(map[string]*ServiceInstance),
	}
	sm.LoadConfig()
	return sm
}
