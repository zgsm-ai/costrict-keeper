package services

import (
	"context"
	"costrict-host/internal/config"
	"costrict-host/internal/models"
	"costrict-host/internal/utils"
	"fmt"
	"os/exec"
)

type ServiceManager struct {
	Services    []config.ServiceConfig
	upgradeFn   func(*utils.UpgradeConfig, utils.VersionNumber, *utils.VersionNumber) error
	runningSvcs map[string]*ServiceInstance
}

func (sm *ServiceManager) GetServices() []config.ServiceConfig {
	return sm.Services
}

func (sm *ServiceManager) GetComponents() ([]models.ComponentInfo, error) {
	// 实现获取组件列表逻辑
	components := make([]models.ComponentInfo, 0)
	// 获取服务作为组件信息
	for _, svc := range sm.Services {
		components = append(components, models.ComponentInfo{
			Name:    svc.Name,
			Version: svc.Versions.Highest,
			Path:    svc.Startup,
		})
	}
	return components, nil
}

func (sm *ServiceManager) UpgradeComponent(name string) error {
	// 实现组件升级逻辑
	// 获取组件配置
	var svc *config.ServiceConfig
	for _, s := range sm.Services {
		if s.Name == name {
			svc = &s
			break
		}
	}
	if svc == nil {
		return fmt.Errorf("service %s not found", name)
	}

	// 解析版本号
	ver, err := utils.ParseVersion(svc.Versions.Highest)
	if err != nil {
		return err
	}
	upgradeCfg := utils.UpgradeConfig{PackageName: name}
	return sm.upgradeFn(&upgradeCfg, ver, nil)
}

func (sm *ServiceManager) IsServiceHealthy(name string) bool {
	// 实现服务健康检查
	// 检查服务进程状态
	if instance, ok := sm.runningSvcs[name]; ok {
		if instance.Command.ProcessState != nil && !instance.Command.ProcessState.Exited() {
			return true
		}
		// 检查服务端口是否可访问
		for _, svc := range sm.Services {
			if svc.Name == name {
				return utils.CheckPortAvailable(svc.Port)
			}
		}
	}
	return false
}

func (sm *ServiceManager) GetEndpoints() []models.ServiceEndpoint {
	endpoints := make([]models.ServiceEndpoint, 0)
	for _, svc := range sm.Services {
		endpoints = append(endpoints, models.ServiceEndpoint{
			Name:    svc.Name,
			URL:     fmt.Sprintf("%s://localhost:%d", svc.Protocol, svc.Port),
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

func (sm *ServiceManager) LoadConfig() {
	// 加载远程服务配置，如果无法加载则使用本地配置
	servicesAddr := fmt.Sprintf("%s/costrict-host/services.json", config.Config.Upgrade.BaseUrl)
	remoteCfg, err := config.LoadRemoteServicesConfig(servicesAddr)
	if err != nil {
		sm.Services = config.Config.Services
	} else {
		sm.Services = remoteCfg.Services
	}
}

func (sm *ServiceManager) StartAll(ctx context.Context) error {
	for _, svc := range sm.Services {
		if err := sm.StartService(ctx, svc.Name); err != nil {
			return err
		}
	}
	return nil
}

func (sm *ServiceManager) StartService(ctx context.Context, name string) error {
	var svcConfig *config.ServiceConfig
	for _, svc := range sm.Services {
		if svc.Name == name {
			svcConfig = &svc
			break
		}
	}

	if svcConfig == nil {
		return fmt.Errorf("service %s not found", name)
	}

	ver := utils.VersionNumber{Major: 1, Minor: 0, Micro: 0}
	upgradeCfg := utils.UpgradeConfig{}
	upgradeCfg.PackageName = name
	if err := sm.upgradeFn(&upgradeCfg, ver, nil); err != nil {
		return err
	}

	return nil
}

func (sm *ServiceManager) RestartService(ctx context.Context, name string) error {
	if err := sm.StopService(name); err != nil {
		return err
	}
	return sm.StartService(ctx, name)
}

func (sm *ServiceManager) StopService(name string) error {
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
