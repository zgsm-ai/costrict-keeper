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
	Tunnel    TunnelInstance              `json:"tunnel"`
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
			StartTime: time.Now().Format(time.RFC3339),
			Spec:      svc,
			component: sm.cm.GetComponent(svc.Name),
		}
		sm.services[svc.Name] = instance
	}
	sm.self.Name = COSTRICT_NAME
	sm.self.Status = "exited"
	sm.self.Spec = config.Spec().Manager.Service
	sm.self.component = sm.cm.GetSelf()
	for name, svc := range sm.services {
		sm.loadService(name, svc)
	}
	sm.loadService(COSTRICT_NAME, &sm.self)
	if env.Daemon {
		sm.self.Pid = os.Getpid()
		sm.self.Status = "running"
		sm.self.Port = env.ListenPort
		sm.self.StartTime = time.Now().Format(time.RFC3339)
		sm.saveService(&sm.self)
	}
	serviceManager = sm
	return serviceManager
}

func (sm *ServiceManager) getServiceKnowledge(svc *ServiceInstance) models.ServiceKnowledge {
	spec := svc.Spec

	installed := false
	version := "unknown"
	component := sm.cm.GetComponent(spec.Name)
	if component != nil {
		version = component.LocalVersion
		installed = component.Installed
	}

	return models.ServiceKnowledge{
		Name:       spec.Name,
		Version:    version,
		Installed:  installed,
		Startup:    spec.Startup,
		Status:     svc.Status,
		Protocol:   spec.Protocol,
		Port:       svc.Port,
		Command:    spec.Command,
		Metrics:    spec.Metrics,
		Healthy:    spec.Healthy,
		Accessible: spec.Accessible,
	}
}

func (sm *ServiceManager) getSelfKnowledge() models.ServiceKnowledge {
	spec := sm.self.Spec
	component := sm.cm.GetSelf()
	return models.ServiceKnowledge{
		Name:       spec.Name,
		Version:    component.LocalVersion,
		Installed:  component.Installed,
		Startup:    spec.Startup,
		Status:     sm.self.Status,
		Protocol:   spec.Protocol,
		Port:       sm.self.Port,
		Command:    spec.Command,
		Metrics:    spec.Metrics,
		Healthy:    spec.Healthy,
		Accessible: spec.Accessible,
	}
}

func (sm *ServiceManager) GetInstances() []*ServiceInstance {
	var svcs []*ServiceInstance
	svcs = append(svcs, &sm.self)
	for _, svc := range sm.services {
		svcs = append(svcs, svc)
	}
	return svcs
}

func (sm *ServiceManager) GetInstance(name string) *ServiceInstance {
	if name == COSTRICT_NAME {
		return &sm.self
	}
	if svc, exist := sm.services[name]; exist {
		return svc
	}
	return nil
}

func (sm *ServiceManager) GetServiceDetail(svc *ServiceInstance) ServiceDetail {
	return ServiceDetail{
		Name:      svc.Name,
		Pid:       svc.Pid,
		Port:      svc.Port,
		Status:    svc.Status,
		StartTime: svc.StartTime,
		Spec:      svc.Spec,
		Tunnel:    *svc.tun,
	}
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
	svc, ok := sm.services[name]
	if !ok {
		return false
	}
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
func (sm *ServiceManager) saveService(svc *ServiceInstance) {
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

func (sm *ServiceManager) loadService(name string, svc *ServiceInstance) error {
	cacheFile := filepath.Join(env.CostrictDir, "cache", "services", name+".json")

	// 检查缓存文件是否存在
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		logger.Debugf("No cache file found for service %s, skipping", name)
		return os.ErrNotExist
	}

	// 读取缓存文件
	jsonData, err := os.ReadFile(cacheFile)
	if err != nil {
		logger.Errorf("Failed to read cache file for service %s: %v", name, err)
		return err
	}

	// 反序列化服务实例
	var cachedInstance ServiceInstance
	if err := json.Unmarshal(jsonData, &cachedInstance); err != nil {
		logger.Errorf("Failed to unmarshal cache data for service %s: %v", name, err)
		return err
	}

	// 验证缓存的服务实例名称是否匹配
	if cachedInstance.Name != name {
		logger.Warnf("Cache file name mismatch for service %s (cached name: %s), skipping", name, cachedInstance.Name)
		return fmt.Errorf("not matched")
	}

	// 更新服务实例状态
	svc.Pid = cachedInstance.Pid
	svc.Status = cachedInstance.Status
	svc.StartTime = cachedInstance.StartTime
	svc.Port = cachedInstance.Port

	// 如果服务状态为running，尝试重新关联进程
	if svc.Pid > 0 {
		svc.proc, err = sm.getProcessInstance(svc)
		if err != nil {
			logger.Errorf("Process %d for service %s configure error: %v", svc.Pid, name, err)
			svc.Status = "exited"
			svc.Pid = 0
			sm.saveService(svc)
			return err
		}
		err := sm.pm.AttachProcess(svc.proc, svc.Pid)
		if err != nil {
			logger.Warnf("Process %d for service %s not found, marking as exited", svc.Pid, name)
			svc.Status = "exited"
			svc.Pid = 0
			sm.saveService(svc)
			return err
		} else {
			// 进程存在
			logger.Infof("Service %s process %d is still running", name, svc.Pid)
		}
	}

	logger.Infof("Successfully loaded service %s from cache", name)

	return nil
}

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

func (sm *ServiceManager) StopAll() {
	for _, svc := range sm.services {
		sm.stopService(svc)
	}
	if env.Daemon {
		sm.self.Pid = 0
		sm.self.Port = 0
		sm.self.Status = "stopped"
		sm.saveService(&sm.self)
	}
	sm.export()
}

func (sm *ServiceManager) getProcessInstance(svc *ServiceInstance) (*ProcessInstance, error) {
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
	return NewProcessInstance("service "+svc.Name, name, command, cmdArgs), nil
}

func (sm *ServiceManager) startService(ctx context.Context, svc *ServiceInstance) error {
	spec := &svc.Spec
	port, err := utils.AllocPort(spec.Port)
	if err != nil {
		return err
	}
	svc.Port = port

	svc.proc, err = sm.getProcessInstance(svc)
	if err != nil {
		return err
	}
	svc.proc.SetRestartCallback(func(pi *ProcessInstance) {
		svc.Pid = pi.Pid
		svc.Status = "running"
		sm.saveService(svc)
	})
	if err := sm.pm.StartProcess(svc.proc); err != nil {
		return err
	}
	svc.Pid = svc.proc.Pid
	svc.StartTime = time.Now().Format(time.RFC3339)
	svc.Status = "running"

	if spec.Accessible == "remote" {
		svc.tun, err = sm.tm.StartTunnel(spec.Name, svc.Port)
		if err != nil {
			logger.Errorf("Start tunnel %s:%d failed: %v", spec.Name, svc.Port, err)
		} else {
			logger.Infof("Start tunnel %s:%d -> %d succeeded", spec.Name, svc.Port, svc.tun.MappingPort)
		}
	} else {
		logger.Infof("ignore %s", spec.Name)
	}
	sm.saveService(svc)
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
	sm.saveService(svc)
	sm.export()
}

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

func (sm *ServiceManager) CheckServices() error {
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
	serviceKnowledge = append(serviceKnowledge, sm.getSelfKnowledge())
	for _, svc := range sm.services {
		serviceKnowledge = append(serviceKnowledge, sm.getServiceKnowledge(svc))
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

func (sm *ServiceManager) export() error {
	outputFile := filepath.Join(env.CostrictDir, "share", ".well-known.json")
	if err := sm.exportKnowledge(outputFile); err != nil {
		logger.Errorf("Failed to export .well-known to file [%s]: %v", outputFile, err)
		return err
	}
	return nil
}
