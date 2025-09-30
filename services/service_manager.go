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
	Name      string
	Status    models.RunStatus
	StartTime string

	spec        models.ServiceSpecification
	component   *ComponentInstance
	proc        *ProcessInstance
	tun         *TunnelInstance
	port        int
	failedCount int  //检测失败，连续三次检测失败，需要重启服务
	child       bool //被本进程直接管理控制的子服务
}

type ServiceCache struct {
	Name      string           `json:"name"`
	Pid       int              `json:"pid"`
	Port      int              `json:"port"`
	Status    models.RunStatus `json:"status"`
	StartTime string           `json:"startTime"`
}

type ServiceArgs struct {
	LocalPort   int
	ProcessPath string
	ProcessName string
}

type ServiceManager struct {
	cm       *ComponentManager
	self     *ServiceInstance
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
	}
	serviceManager = sm
	return serviceManager
}

func newService(spec *models.ServiceSpecification, cpn *ComponentInstance, child bool) *ServiceInstance {
	svc := &ServiceInstance{
		Name:      spec.Name,
		Status:    models.StatusExited,
		spec:      *spec,
		component: cpn,
		child:     child,
	}
	svc.proc = svc.createProcessInstance()
	if spec.Accessible == "remote" {
		svc.tun = CreateTunnel(spec.Name, []int{spec.Port})
	}
	return svc
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
	svc.Status = models.RunStatus(status)
	svc.saveService()
	serviceManager.export()
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
 */
func (svc *ServiceInstance) GetDetail() models.ServiceDetail {
	detail := &models.ServiceDetail{
		Name:      svc.Name,
		Port:      svc.port,
		Status:    svc.Status,
		StartTime: svc.StartTime,
		Spec:      svc.spec,
	}
	if svc.spec.Accessible == "remote" {
		tun := svc.tun.GetDetail()
		detail.Tunnel = &tun
	}
	if !svc.child {
		detail.Pid = os.Getpid()
	} else {
		detail.Pid = svc.proc.Pid
	}
	detail.Process = svc.proc.GetDetail()
	if svc.component != nil {
		cpn := svc.component.GetDetail()
		detail.Component = &cpn
	} else {
		detail.Component = nil
	}
	detail.Healthy = svc.IsHealthy()
	return *detail
}

/**
 * Get process instance associated with service
 * @returns {ProcessInstance} Returns process instance if exists, nil otherwise
 * @description
 * - Returns the process instance that runs this service
 * - Returns nil if service is not running or has no associated process
 * - Used to access process-level operations and information
 */
func (svc *ServiceInstance) GetProc() *ProcessInstance {
	return svc.proc
}

func (svc *ServiceInstance) GetTunnel() *TunnelInstance {
	return svc.tun
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
	if svc.Status != models.StatusRunning {
		return false
	}
	// 如果端口不可用（已被占用），说明服务正在监听
	if svc.port > 0 {
		return utils.CheckPortConnectable(svc.port)
	}
	running, err := utils.IsProcessRunning(svc.proc.Pid)
	if err != nil || !running {
		return false
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
	installed := false
	version := ""
	if svc.component != nil && svc.component.local != nil {
		version = utils.PrintVersion(svc.component.local.VersionId)
		installed = svc.component.installed
	}
	return models.ServiceKnowledge{
		Name:       svc.Name,
		Version:    version,
		Installed:  installed,
		Command:    svc.proc.Command,
		Status:     string(svc.Status),
		Port:       svc.port,
		Startup:    svc.spec.Startup,
		Protocol:   svc.spec.Protocol,
		Metrics:    svc.spec.Metrics,
		Healthy:    svc.spec.Healthy,
		Accessible: svc.spec.Accessible,
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
		logger.Errorf("Service [%s] save info failed, error: %v", svc.Name, err)
		return
	}

	var cache ServiceCache
	cache.Name = svc.Name
	cache.Port = svc.port
	cache.StartTime = svc.StartTime
	cache.Status = svc.Status
	if svc.child {
		cache.Pid = svc.proc.Pid
	} else {
		cache.Pid = os.Getpid()
	}

	jsonData, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		logger.Errorf("Service [%s] save info failed, error: %v", svc.Name, err)
		return
	}

	// 写入文件
	cacheFile := filepath.Join(cacheDir, svc.Name+".json")
	if err := os.WriteFile(cacheFile, jsonData, 0644); err != nil {
		logger.Errorf("Service [%s] save info failed, error: %v", svc.Name, err)
		return
	}

	logger.Infof("Service [%s] info saved to %s", svc.Name, cacheFile)
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
func (svc *ServiceInstance) StartService(ctx context.Context) error {
	var err error

	svc.port, err = utils.AllocPort(svc.spec.Port)
	if err != nil {
		return err
	}
	svc.proc = svc.createProcessInstance()
	if svc.proc.Status == models.StatusError {
		svc.Status = models.StatusError
		return err
	}
	svc.proc.SetOnRestarted(func(pi *ProcessInstance) {
		if pi.Pid == 0 {
			svc.Status = models.StatusError
		} else {
			svc.Status = models.StatusRunning
		}
		svc.saveService()
	})
	if err := svc.proc.StartProcess(ctx); err != nil {
		svc.Status = models.StatusError
		return err
	}
	svc.Status = models.StatusRunning
	svc.StartTime = time.Now().Format(time.RFC3339)
	svc.OpenTunnel(ctx)

	svc.saveService()
	return nil
}

func (svc *ServiceInstance) StopService() {
	svc.Status = models.StatusStopped
	svc.proc.StopProcess()
	if svc.tun != nil {
		svc.tun.CloseTunnel()
	}
	svc.saveService()
}

func (svc *ServiceInstance) RecoverService() {
	if svc.Status == models.StatusStopped {
		return
	}
	svc.CheckService()
	if svc.failedCount > 2 {
		logger.Warnf("Service '%s' failed detection three times, automatically restart", svc.Name)
		svc.failedCount = 0
		svc.StopService()
		svc.StartService(context.Background())
	}
	if !svc.IsHealthy() {
		// 服务进程如果挂了，要把隧道也关了，避免端口变化隧道无法使用
		if svc.tun != nil {
			svc.tun.CloseTunnel()
		}
		svc.StartService(context.Background())
	}
	if svc.tun == nil || !svc.tun.IsHealthy() {
		svc.OpenTunnel(context.Background())
	}
}

func (svc *ServiceInstance) CheckService() error {
	if svc.Status == models.StatusStopped {
		return nil
	}
	if svc.port > 0 {
		if !utils.CheckPortConnectable(svc.port) {
			logger.Errorf("Service [%s] is unhealthy", svc.Name)
			svc.failedCount++
		} else {
			svc.failedCount = 0
		}
	}
	svc.proc.CheckProcess()
	if svc.tun != nil {
		svc.tun.CheckTunnel()
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
 */
func (svc *ServiceInstance) createProcessInstance() *ProcessInstance {
	name := svc.Name
	if runtime.GOOS == "windows" {
		name = fmt.Sprintf("%s.exe", svc.Name)
	}
	args := ServiceArgs{
		LocalPort:   svc.port,
		ProcessName: name,
		ProcessPath: filepath.Join(env.CostrictDir, "bin", name),
	}
	command, cmdArgs, err := utils.GetCommandLine(svc.spec.Command, svc.spec.Args, args)
	if err != nil {
		proc := NewProcessInstance("service "+svc.Name, name, command, cmdArgs)
		proc.Status = models.StatusError
		proc.LastExitReason = err.Error()
		return proc
	}
	return NewProcessInstance("service "+svc.Name, name, command, cmdArgs)
}

func (svc *ServiceInstance) OpenTunnel(ctx context.Context) error {
	if svc.spec.Accessible != "remote" {
		return nil
	}
	svc.tun = CreateTunnel(svc.Name, []int{svc.port})
	if err := svc.tun.OpenTunnel(ctx); err != nil {
		logger.Errorf("Start tunnel (%s:%d) failed: %v", svc.Name, svc.port, err)
		return err
	}
	return nil
}

func (svc *ServiceInstance) CloseTunnel() error {
	if svc.tun == nil {
		return nil
	}
	err := svc.tun.CloseTunnel()
	return err
}

func (svc *ServiceInstance) ReopenTunnel(ctx context.Context) error {
	if svc.tun != nil {
		svc.CloseTunnel()
	}
	return svc.OpenTunnel(ctx)
}

func (sm *ServiceManager) Init() error {
	for _, spec := range config.Spec().Services {
		cpn := sm.cm.GetComponent(spec.Name)
		if cpn == nil {
			logger.Errorf("component [%s] isn't exist", spec.Name)
			return os.ErrNotExist
		}
		svc := newService(&spec, cpn, true)
		sm.services[spec.Name] = svc
	}
	sm.self = newService(&config.Spec().Manager.Service, sm.cm.GetSelf(), false)
	if env.Daemon {
		sm.self.Status = models.StatusRunning
		sm.self.port = env.ListenPort
		sm.self.StartTime = time.Now().Format(time.RFC3339)
		sm.self.saveService()
	}
	return nil
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
	return sm.self
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
		svcs = append(svcs, sm.self)
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
	if name == COSTRICT_NAME {
		return sm.GetSelf()
	}
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
		if svc.spec.Startup == "always" || svc.spec.Startup == "once" {
			if svc.Status == models.StatusRunning {
				continue
			}
			if err := svc.StartService(ctx); err != nil {
				logger.Errorf("Failed to start service '%s': %v", svc.Name, err)
			}
		}
	}
	sm.export()
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
		svc.StopService()
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
 * - Calls StartService to perform actual service start
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
	if svc.Status == models.StatusRunning {
		return fmt.Errorf("service %s is already running", name)
	}
	if err := svc.StartService(ctx); err != nil {
		logger.Errorf("Start [%s] failed: %v", name, err)
		return err
	}
	sm.export()
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
	if svc.Status == models.StatusRunning {
		svc.StopService()
	}
	if err := svc.StartService(ctx); err != nil {
		logger.Errorf("Restart [%s] failed: %v", name, err)
		return err
	}
	sm.export()
	return nil
}

/**
 * Stop specific service by name
 * @param {string} name - Name of the service to stop
 * @returns {error} Returns error if stop fails, nil on success
 * @description
 * - Checks if service exists in service manager
 * - Returns nil if service is not running
 * - Calls StopService to perform actual service stop
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
	if svc.Status != models.StatusRunning {
		return nil
	}
	svc.StopService()
	sm.export()
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
 */
func (sm *ServiceManager) CheckServices() {
	for _, svc := range sm.services {
		svc.CheckService()
	}
}

func (sm *ServiceManager) RecoverServices() {
	logger.Debugf("Recover broken services")
	for _, svc := range sm.services {
		svc.RecoverService()
	}
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
	if outputPath == "" {
		outputPath = filepath.Join(env.CostrictDir, "share", ".well-known.json")
	}
	if err := sm.exportKnowledge(outputPath); err != nil {
		logger.Errorf("Failed to export .well-known to file [%s]: %v", outputPath, err)
		return err
	}
	return nil
}

func (sm *ServiceManager) exportKnowledge(outputPath string) error {
	serviceKnowledge := []models.ServiceKnowledge{}
	serviceKnowledge = append(serviceKnowledge, sm.self.getKnowledge())
	for _, svc := range sm.services {
		serviceKnowledge = append(serviceKnowledge, svc.getKnowledge())
	}
	// 构建日志知识
	logKnowledge := models.LogKnowledge{
		Dir:   filepath.Join(env.CostrictDir, "logs"),
		Level: config.App().Log.Level,
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
