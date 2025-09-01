package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"costrict-keeper/internal/config"
	"costrict-keeper/internal/env"
	"costrict-keeper/internal/logger"
	"costrict-keeper/internal/models"
	"costrict-keeper/internal/utils"
)

const (
	COSTRICT_NAME = "costrict"
)

/**
 * Service instance information
 * @property {int} pid - Process ID
 * @property {string} status - Service status: running/stopped/error/exited
 * @property {string} startTime - Service start time in ISO format
 * @property {models.ServiceSpecification} config - Service configuration
 */
type ServiceInstance struct {
	Name      string `json:"name"`
	Pid       int    `json:"pid"`
	Port      int    `json:"port"`
	Status    string `json:"status"`
	StartTime string `json:"startTime"`

	Spec      models.ServiceSpecification `json:"-"`
	component *ComponentInstance
	proc      *ProcessInstance
	tun       *TunnelInstance
}

type ServiceDetail struct {
	Name      string                      `json:"name"`
	Pid       int                         `json:"pid"`
	Port      int                         `json:"port"`
	Status    string                      `json:"status"`
	StartTime string                      `json:"startTime"`
	Spec      models.ServiceSpecification `json:"spec"`
	Tunnel    *TunnelInstance             `json:"tunnel,omitempty"`
	Process   *ProcessInstance            `json:"process,omitempty"`
	Component *ComponentInstance          `json:"component,omitempty"`
}

type ServiceArgs struct {
	LocalPort   int
	ProcessPath string
	ProcessName string
}

type ServiceManager struct {
	cm       *ComponentManager
	tm       *TunnelManager
	pm       *ProcessManager
	self     ServiceInstance
	services map[string]*ServiceInstance
}

var serviceManager *ServiceManager

/**
 * Get service manager singleton instance
 * @returns {ServiceManager} Returns the singleton ServiceManager instance
 * @description
 * - Implements singleton pattern to ensure only one ServiceManager exists
 * - Initializes service manager with component, tunnel, and process managers
 * - Creates service instances from configuration
 * - Loads existing service state from cache
 * - Sets up self service instance for the manager
 * - Returns existing instance if already initialized
 * @example
 * serviceManager := GetServiceManager()
 * services := serviceManager.GetInstances()
 */
func GetServiceManager() *ServiceManager {
	if serviceManager != nil {
		return serviceManager
	}
	sm := &ServiceManager{
		services: make(map[string]*ServiceInstance),
		cm:       GetComponentManager(),
		tm:       GetTunnelManager(),
		pm:       GetProcessManager(),
	}
	for _, svc := range config.Spec().Services {
		instance := &ServiceInstance{
			Name:      svc.Name,
			Pid:       0,
			Status:    "exited",
			Spec:      svc,
			component: sm.cm.GetComponent(svc.Name),
		}
		sm.services[svc.Name] = instance
	}
	for _, svc := range sm.services {
		svc.loadService()
		svc.attachProcess()
	}
	sm.self.Name = COSTRICT_NAME
	sm.self.Status = "exited"
	sm.self.Spec = config.Spec().Manager.Service
	sm.self.component = sm.cm.GetSelf()
	sm.self.loadService()
	if env.Daemon {
		sm.self.Pid = os.Getpid()
		sm.self.Status = "running"
		sm.self.Port = env.ListenPort
		sm.self.StartTime = time.Now().Format(time.RFC3339)
		sm.self.saveService()
	}
	serviceManager = sm
	return serviceManager
}

/**
 * Update costrict service status
 * @param {string} status - New status to set for costrict service
 * @description
 * - Updates the status of the costrict self service
 * - Saves the updated service information to cache
 * - Used to track the current state of the manager service
 * @example
 * UpdateCostrictStatus("running")
 */
func UpdateCostrictStatus(status string) {
	svc := serviceManager.GetSelf()
	svc.Status = status
	svc.saveService()
}

/**
 * Get self service knowledge information
 * @returns {ServiceKnowledge} Returns self service knowledge structure
 * @description
 * - Creates ServiceKnowledge structure for manager service
 * - Includes manager name, version, and installation status
 * - Uses current environment settings for port and status
 * - Used for system knowledge export and manager discovery
 * @private
 */
func getSelfKnowledge() models.ServiceKnowledge {
	spec := config.Spec().Manager.Service
	component := GetComponentManager().GetSelf()
	name := COSTRICT_NAME
	if runtime.GOOS == "windows" {
		name = fmt.Sprintf("%s.exe", name)
	}
	args := ServiceArgs{
		LocalPort:   env.ListenPort,
		ProcessName: name,
		ProcessPath: filepath.Join(env.CostrictDir, "bin", name),
	}
	command, _, err := utils.GetCommandLine(spec.Command, spec.Args, args)
	if err != nil {
		command = name
	}
	return models.ServiceKnowledge{
		Name:       COSTRICT_NAME,
		Version:    component.LocalVersion,
		Installed:  component.Installed,
		Status:     "running",
		Port:       env.ListenPort,
		Startup:    spec.Startup,
		Protocol:   spec.Protocol,
		Command:    command,
		Metrics:    spec.Metrics,
		Healthy:    spec.Healthy,
		Accessible: spec.Accessible,
	}
}

/**
 * Get detailed service information
 * @param {ServiceInstance} svc - Service instance to get details for
 * @returns {ServiceDetail} Returns detailed service information
 * @description
 * - Creates ServiceDetail structure from ServiceInstance
 * - Includes service name, PID, port, status, and start time
 * - Includes service specification and tunnel information
 * - Used for API responses and detailed service views
 * @example
 * detail := serviceInstance.GetDetail()
 * fmt.Printf("Service %s is %s", detail.Name, detail.Status)
 */
func (svc *ServiceInstance) GetDetail() ServiceDetail {
	detail := &ServiceDetail{
		Name:      svc.Name,
		Pid:       svc.Pid,
		Port:      svc.Port,
		Status:    svc.Status,
		StartTime: svc.StartTime,
		Spec:      svc.Spec,
		Tunnel:    svc.tun,
		Process:   svc.proc,
		Component: svc.component,
	}
	if svc.proc == nil {
		detail.Process, _ = svc.CreateProcessInstance()
	}
	return *detail
}

/**
 * Get process instance associated with service
 * @returns {ProcessInstance} Returns process instance if exists, nil otherwise
 * @description
 * - Returns the process instance that runs this service
 * - Returns nil if service is not running or has no associated process
 * - Used to access process-level operations and information
 * @example
 * proc := serviceInstance.GetProc()
 * if proc != nil {
 *     fmt.Printf("Process PID: %d", proc.Pid)
 * }
 */
func (svc *ServiceInstance) GetProc() *ProcessInstance {
	return svc.proc
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
func (svc *ServiceInstance) IsHealthy() bool {
	if svc.Status != "running" {
		return false
	}
	// 如果端口不可用（已被占用），说明服务正在监听
	if svc.Port > 0 {
		return utils.CheckPortConnectable(svc.Port)
	}
	return true
}

/**
 * Get service knowledge information
 * @returns {ServiceKnowledge} Returns service knowledge structure
 * @description
 * - Creates ServiceKnowledge structure from service instance
 * - Includes service name, version, installation status, and configuration
 * - Retrieves component information for version and installation status
 * - Used for system knowledge export and service discovery
 * @private
 */
func (svc *ServiceInstance) getKnowledge() models.ServiceKnowledge {
	spec := svc.Spec

	installed := false
	version := "unknown"
	component := GetComponentManager().GetComponent(spec.Name)
	if component != nil {
		version = component.LocalVersion
		installed = component.Installed
	}
	command := spec.Name
	if svc.proc != nil {
		command = svc.proc.Command
	} else {
		if runtime.GOOS == "windows" {
			command = fmt.Sprintf("%s.exe", spec.Name)
		}
	}

	return models.ServiceKnowledge{
		Name:       svc.Name,
		Version:    version,
		Installed:  installed,
		Command:    command,
		Status:     svc.Status,
		Port:       svc.Port,
		Startup:    spec.Startup,
		Protocol:   spec.Protocol,
		Metrics:    spec.Metrics,
		Healthy:    spec.Healthy,
		Accessible: spec.Accessible,
	}
}

/**
 * Save service information to cache file
 * @param {string} serviceName - Name of the service
 * @param {ServiceInstance} svc - Service instance information
 * @returns {error} Returns error if save fails, nil on success
 * @description
 * - Creates service info structure from instance
 * - Ensures cache directory exists
 * - Marshals service info to JSON
 * - Writes to service-specific JSON file in .costrict/cache/services/
 * @throws
 * - Directory creation errors
 * - JSON marshaling errors
 * - File write errors
 */
func (svc *ServiceInstance) saveService() {
	// 确保缓存目录存在
	cacheDir := filepath.Join(env.CostrictDir, "cache", "services")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		logger.Errorf("Service [%s] save info failed, error: %v", svc.Spec.Name, err)
		return
	}

	// 序列化为JSON
	jsonData, err := json.MarshalIndent(svc, "", "  ")
	if err != nil {
		logger.Errorf("Service [%s] save info failed, error: %v", svc.Spec.Name, err)
		return
	}

	// 写入文件
	cacheFile := filepath.Join(cacheDir, svc.Spec.Name+".json")
	if err := os.WriteFile(cacheFile, jsonData, 0644); err != nil {
		logger.Errorf("Service [%s] save info failed, error: %v", svc.Spec.Name, err)
		return
	}

	logger.Infof("Service [%s] info saved to %s", svc.Spec.Name, cacheFile)
}

/**
 * Load service information from cache file
 * @returns {error} Returns error if load fails, nil on success
 * @description
 * - Reads service information from cache file in .costrict/cache/services/
 * - Validates cache file name matches service name
 * - Updates service instance with cached PID, status, start time, and port
 * - Returns os.ErrNotExist if cache file doesn't exist
 * - Used for restoring service state after restart
 * @throws
 * - File read errors
 * - JSON unmarshaling errors
 * - Cache validation errors
 * @example
 * err := serviceInstance.loadService()
 * if err != nil && !errors.Is(err, os.ErrNotExist) {
 *     logger.Error("Failed to load service:", err)
 * }
 */
func (svc *ServiceInstance) loadService() error {
	cacheFile := filepath.Join(env.CostrictDir, "cache", "services", svc.Name+".json")

	// 检查缓存文件是否存在
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		logger.Debugf("No cache file found for service %s, skipping", svc.Name)
		return os.ErrNotExist
	}

	// 读取缓存文件
	jsonData, err := os.ReadFile(cacheFile)
	if err != nil {
		logger.Errorf("Failed to read cache file for service %s: %v", svc.Name, err)
		return err
	}

	// 反序列化服务实例
	var cachedInstance ServiceInstance
	if err := json.Unmarshal(jsonData, &cachedInstance); err != nil {
		logger.Errorf("Failed to unmarshal cache data for service %s: %v", svc.Name, err)
		return err
	}

	// 验证缓存的服务实例名称是否匹配
	if cachedInstance.Name != svc.Name {
		logger.Warnf("Cache file name mismatch for service %s (cached name: %s), skipping", svc.Name, cachedInstance.Name)
		return fmt.Errorf("not matched")
	}

	// 更新服务实例状态
	svc.Pid = cachedInstance.Pid
	svc.Status = cachedInstance.Status
	svc.StartTime = cachedInstance.StartTime
	svc.Port = cachedInstance.Port
	return nil
}

/**
 * Attach to existing process for service
 * @returns {error} Returns error if attach fails, nil on success
 * @description
 * - Creates process instance for service if PID > 0
 * - Uses process manager to attach to existing process
 * - Updates service status based on attach result
 * - Marks service as exited if process not found
 * - Saves updated service state to cache
 * - Used for reconnecting to running processes after restart
 * @throws
 * - Process instance creation errors
 * - Process attachment errors
 * @example
 * err := serviceInstance.attachProcess()
 * if err != nil {
 *     logger.Error("Failed to attach to process:", err)
 * }
 */
func (svc *ServiceInstance) attachProcess() error {
	if svc.Pid <= 0 {
		return nil
	}
	name := svc.Name
	// 如果服务状态为running，尝试重新关联进程
	if _, err := svc.CreateProcessInstance(); err != nil {
		logger.Errorf("Process %d for service %s configure error: %v", svc.Pid, name, err)
		svc.Status = "exited"
		svc.Pid = 0
		svc.saveService()
		return err
	}
	err := GetProcessManager().AttachProcess(svc.proc, svc.Pid)
	if err != nil {
		logger.Warnf("Process %d for service %s not found, marking as exited", svc.Pid, name)
		svc.Status = "exited"
		svc.Pid = 0
		svc.proc = nil
		svc.saveService()
		return err
	} else {
		// 进程存在
		logger.Infof("Service %s process %d is still running", name, svc.Pid)
	}
	return nil
}

/**
 * Create process instance for service execution
 * @returns {ProcessInstance} Returns created process instance
 * @returns {error} Returns error if creation fails, nil on success
 * @description
 * - Adjusts process name for Windows (.exe extension)
 * - Creates ServiceArgs with port, process name, and path
 * - Generates command line using service specification
 * - Creates new ProcessInstance with generated command and args
 * - Used for starting new service processes
 * @throws
 * - Command line generation errors
 * @example
 * proc, err := serviceInstance.CreateProcessInstance()
 * if err != nil {
 *     logger.Error("Failed to create process instance:", err)
 *     return nil, err
 * }
 */
func (svc *ServiceInstance) CreateProcessInstance() (*ProcessInstance, error) {
	name := svc.Spec.Name
	if runtime.GOOS == "windows" {
		name = fmt.Sprintf("%s.exe", svc.Spec.Name)
	}
	args := ServiceArgs{
		LocalPort:   svc.Port,
		ProcessName: name,
		ProcessPath: filepath.Join(env.CostrictDir, "bin", name),
	}
	command, cmdArgs, err := utils.GetCommandLine(svc.Spec.Command, svc.Spec.Args, args)
	if err != nil {
		return nil, err
	}
	svc.proc = NewProcessInstance("service "+svc.Name, name, command, cmdArgs)
	return svc.proc, nil
}

/**
 * Get self service instance (costrict manager)
 * @returns {ServiceInstance} Returns the manager service instance
 * @description
 * - Returns the service instance representing the manager itself
 * - Contains manager's PID, port, status, and configuration
 * - Used for manager self-management and monitoring
 * @example
 * serviceManager := GetServiceManager()
 * selfService := serviceManager.GetSelf()
 * fmt.Printf("Manager PID: %d", selfService.Pid)
 */
func (sm *ServiceManager) GetSelf() *ServiceInstance {
	return &sm.self
}

/**
 * Get all managed service instances (excluding self)
 * @returns {[]ServiceInstance} Returns slice of managed service instances
 * @description
 * - Returns slice containing all configured service instances
 * - Excludes the self service instance
 * - Used for managing and monitoring configured services
 * @example
 * serviceManager := GetServiceManager()
 * services := serviceManager.GetInstances(true)
 * for _, svc := range services {
 *     fmt.Printf("Service: %s, Status: %s", svc.Name, svc.Status)
 * }
 */
func (sm *ServiceManager) GetInstances(includeSelf bool) []*ServiceInstance {
	var svcs []*ServiceInstance
	if includeSelf {
		svcs = append(svcs, &sm.self)
	}
	for _, svc := range sm.services {
		svcs = append(svcs, svc)
	}
	return svcs
}

/**
 * Get service instance by name
 * @param {string} name - Name of the service to retrieve
 * @returns {ServiceInstance} Returns service instance if found, nil otherwise
 * @description
 * - Searches for service by name in the services map
 * - Returns nil if service is not found
 * - Used to access specific service information and operations
 * @example
 * serviceManager := GetServiceManager()
 * service := serviceManager.GetInstance("my-service")
 * if service != nil {
 *     fmt.Printf("Service status: %s", service.Status)
 * }
 */
func (sm *ServiceManager) GetInstance(name string) *ServiceInstance {
	if svc, exist := sm.services[name]; exist {
		return svc
	}
	return nil
}

/**
 * Start all services with "always" or "once" startup mode
 * @param {context.Context} ctx - Context for cancellation and timeout
 * @returns {error} Returns nil (always returns nil for backward compatibility)
 * @description
 * - Iterates through all managed services
 * - Starts services with startup mode "always" or "once"
 * - Skips services that are already running
 * - Logs errors for individual service start failures
 * - Continues processing other services even if some fail
 * @example
 * ctx := context.Background()
 * if err := serviceManager.StartAll(ctx); err != nil {
 *     logger.Error("Some services failed to start")
 * }
 */
func (sm *ServiceManager) StartAll(ctx context.Context) error {
	for _, svc := range sm.services {
		// 只启动启动模式为 "always"和"once" 的服务
		if svc.Spec.Startup == "always" || svc.Spec.Startup == "once" {
			if svc.Status == "running" {
				continue
			}
			if err := sm.startService(ctx, svc); err != nil {
				logger.Errorf("Failed to start service '%s': %v", svc.Spec.Name, err)
			}
		}
	}
	return nil
}

/**
 * Stop all managed services
 * @description
 * - Iterates through all managed services
 * - Stops each service regardless of current status
 * - Exports service knowledge after stopping all services
 * - Used for graceful shutdown and service restart
 * @example
 * serviceManager := GetServiceManager()
 * serviceManager.StopAll()
 */
func (sm *ServiceManager) StopAll() {
	for _, svc := range sm.services {
		sm.stopService(svc)
	}
	sm.export()
}

/**
 * Start specific service by name
 * @param {context.Context} ctx - Context for cancellation and timeout
 * @param {string} name - Name of the service to start
 * @returns {error} Returns error if start fails, nil on success
 * @description
 * - Checks if service exists in service manager
 * - Returns error if service is already running
 * - Calls startService to perform actual service start
 * - Logs error if service start fails
 * @throws
 * - Service not found errors
 * - Service already running errors
 * - Service start errors
 * @example
 * ctx := context.Background()
 * if err := serviceManager.StartService(ctx, "my-service"); err != nil {
 *     logger.Error("Failed to start service:", err)
 * }
 */
func (sm *ServiceManager) StartService(ctx context.Context, name string) error {
	svc, ok := sm.services[name]
	if !ok {
		return fmt.Errorf("service %s not found", name)
	}
	if svc.Status == "running" {
		return fmt.Errorf("service %s is already running", name)
	}
	if err := sm.startService(ctx, svc); err != nil {
		logger.Errorf("Start [%s] failed: %v", name, err)
		return err
	}
	return nil
}

/**
 * Restart specific service by name
 * @param {context.Context} ctx - Context for cancellation and timeout
 * @param {string} name - Name of the service to restart
 * @returns {error} Returns error if restart fails, nil on success
 * @description
 * - Checks if service exists in service manager
 * - Stops service if currently running
 * - Starts service with new configuration
 * - Logs error if service restart fails
 * @throws
 * - Service not found errors
 * - Service stop errors
 * - Service start errors
 * @example
 * ctx := context.Background()
 * if err := serviceManager.RestartService(ctx, "my-service"); err != nil {
 *     logger.Error("Failed to restart service:", err)
 * }
 */
func (sm *ServiceManager) RestartService(ctx context.Context, name string) error {
	svc, ok := sm.services[name]
	if !ok {
		logger.Errorf("Restart [%s] failed: service not found", name)
		return fmt.Errorf("service %s not found", name)
	}
	if svc.Status == "running" {
		sm.stopService(svc)
	}
	if err := sm.startService(ctx, svc); err != nil {
		logger.Errorf("Restart [%s] failed: %v", name, err)
		return err
	}
	return nil
}

/**
 * Stop specific service by name
 * @param {string} name - Name of the service to stop
 * @returns {error} Returns error if stop fails, nil on success
 * @description
 * - Checks if service exists in service manager
 * - Returns nil if service is not running
 * - Calls stopService to perform actual service stop
 * - Logs error if service not found
 * @throws
 * - Service not found errors
 * @example
 * if err := serviceManager.StopService("my-service"); err != nil {
 *     logger.Error("Failed to stop service:", err)
 * }
 */
func (sm *ServiceManager) StopService(name string) error {
	svc, ok := sm.services[name]
	if !ok {
		logger.Errorf("Stop [%s] failed: service not found", name)
		return fmt.Errorf("service %s not found", name)
	}
	if svc.Status != "running" {
		return nil
	}
	sm.stopService(svc)
	return nil
}

/**
 * Check health status of all running services
 * @returns {error} Returns nil (always returns nil for backward compatibility)
 * @description
 * - Iterates through all managed services
 * - Checks port connectivity for services with port > 0
 * - Logs error for services that are unhealthy
 * - Used for periodic health monitoring
 * @example
 * if err := serviceManager.CheckServices(); err != nil {
 *     logger.Error("Service health check failed")
 * }
 */
func (sm *ServiceManager) CheckServices() {
	for _, svc := range sm.services {
		if svc.Status == "running" && svc.Port > 0 && !utils.CheckPortConnectable(svc.Port) {
			logger.Errorf("Service [%s] is unhealthy", svc.Spec.Name)
		}
	}
}

func (sm *ServiceManager) MonitorServices() error {
	for _, svc := range sm.services {
		if svc.Status == "running" && svc.Port > 0 && !utils.CheckPortConnectable(svc.Port) {
			logger.Errorf("Service [%s] is unhealthy", svc.Spec.Name)
		}
	}
	return nil
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
 * err := ExportKnowledge(context.Background(), "")
 * if err != nil {
 *     logger.Fatal(err)
 * }
 */
func (sm *ServiceManager) ExportKnowledge(outputPath string) error {
	if err := sm.exportKnowledge(outputPath); err != nil {
		logger.Errorf("Failed to export .well-known to file [%s]: %v", outputPath, err)
		return err
	}
	return nil
}

func (sm *ServiceManager) exportKnowledge(outputPath string) error {
	serviceKnowledge := []models.ServiceKnowledge{}
	serviceKnowledge = append(serviceKnowledge, getSelfKnowledge())
	for _, svc := range sm.services {
		serviceKnowledge = append(serviceKnowledge, svc.getKnowledge())
	}
	// 构建日志知识
	logKnowledge := models.LogKnowledge{
		Dir:   filepath.Join(env.CostrictDir, "logs"),
		Level: config.Get().Log.Level,
	}

	// 构建要导出的信息结构
	info := models.SystemKnowledge{
		Logs:     logKnowledge,
		Services: serviceKnowledge,
	}

	// 确保目录存在
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 将信息编码为 JSON
	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 编码失败: %v", err)
	}
	// 写入文件
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}
	return nil
}

/**
 * Export service knowledge to default well-known file
 * @returns {error} Returns error if export fails, nil on success
 * @description
 * - Calls exportKnowledge with default output file path
 * - Default path is .costrict/share/.well-known.json
 * - Logs error if export fails
 * - Used for automatic knowledge export
 * @private
 */
func (sm *ServiceManager) export() error {
	outputFile := filepath.Join(env.CostrictDir, "share", ".well-known.json")
	if err := sm.exportKnowledge(outputFile); err != nil {
		logger.Errorf("Failed to export .well-known to file [%s]: %v", outputFile, err)
		return err
	}
	return nil
}

/**
 * Start individual service
 * @param {context.Context} ctx - Context for cancellation and timeout
 * @param {ServiceInstance} svc - Service instance to start
 * @returns {error} Returns error if start fails, nil on success
 * @description
 * - Allocates port for service from specification
 * - Creates process instance for service
 * - Sets restart callback to update service information
 * - Starts process via process manager
 * - Updates service status and saves to cache
 * - Creates tunnel if service has tunnel configuration
 * - Logs successful service start
 * @throws
 * - Port allocation errors
 * - Process creation errors
 * - Process start errors
 * - Tunnel creation errors
 * @private
 */
func (sm *ServiceManager) startService(ctx context.Context, svc *ServiceInstance) error {
	spec := &svc.Spec
	port, err := utils.AllocPort(spec.Port)
	if err != nil {
		return err
	}
	svc.Port = port

	if _, err = svc.CreateProcessInstance(); err != nil {
		svc.Pid = 0
		svc.Status = "error"
		return err
	}
	svc.proc.SetRestartCallback(func(pi *ProcessInstance) {
		svc.Pid = pi.Pid
		svc.Status = "running"
		svc.saveService()
	})
	if err := sm.pm.StartProcess(svc.proc); err != nil {
		svc.Status = "error"
		svc.Pid = 0
		svc.proc = nil
		return err
	}
	svc.Pid = svc.proc.Pid
	svc.StartTime = time.Now().Format(time.RFC3339)
	svc.Status = "running"

	if spec.Accessible == "remote" {
		svc.tun, err = sm.tm.StartTunnel(spec.Name, svc.Port)
		if err != nil {
			logger.Errorf("Start tunnel (%s:%d) failed: %v", spec.Name, svc.Port, err)
		}
	} else {
		logger.Debugf("ignore %s", spec.Name)
	}
	svc.saveService()
	sm.export()
	return nil
}

func (sm *ServiceManager) stopService(svc *ServiceInstance) {
	if svc.proc != nil {
		if err := sm.pm.StopProcess(svc.proc); err != nil {
			logger.Errorf("Failed to stop the service %s (PID: %d)", svc.Spec.Name, svc.Pid)
		} else {
			logger.Infof("Successfully stopped the service %s (PID: %d)", svc.Spec.Name, svc.Pid)
		}
	}
	if svc.tun != nil {
		if err := sm.tm.CloseTunnel(svc.Name, svc.Port); err != nil {
			logger.Errorf("Failed to close tunnel %s (Port: %d)", svc.Name, svc.Port)
		} else {
			logger.Infof("Successfully closed the tunnel %s (Port: %d)", svc.Name, svc.Port)
		}
		svc.tun = nil
	}
	svc.Status = "stopped"
	svc.Pid = 0
	svc.proc = nil
	svc.saveService()
	sm.export()
}
