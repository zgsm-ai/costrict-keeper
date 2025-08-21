package services

import (
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/env"
	"costrict-keeper/internal/logger"
	"costrict-keeper/internal/models"
	"costrict-keeper/internal/utils"
	"fmt"
	"path/filepath"
)

type ComponentInstance struct {
	Spec           models.ComponentSpecification `json:"spec"`
	LocalVersion   string                        `json:"local_version"`
	RemoteVersion  string                        `json:"remote_version"`
	Installed      bool                          `json:"installed"`
	NeedUpgrade    bool                          `json:"need_upgrade"`
	RemotePlatform *utils.PlatformInfo           `json:"-"`

	localVersion utils.VersionNumber
	remoteVerion utils.VersionNumber
}

/**
 * Component manager provides methods to get local and remote version information
 * for both services and components
 */
type ComponentManager struct {
	self       ComponentInstance
	components map[string]*ComponentInstance
}

var componentManager *ComponentManager

/**
 * Create new component manager instance
 * @returns {ComponentManager} Returns new component manager instance
 */
func GetComponentManager() *ComponentManager {
	if componentManager != nil {
		return componentManager
	}
	componentManager = &ComponentManager{
		components: make(map[string]*ComponentInstance),
	}
	for _, cpn := range config.Spec().Components {
		ci := ComponentInstance{
			Spec: cpn,
		}
		fetchComponentInfo(&ci)
		componentManager.components[cpn.Name] = &ci
	}
	componentManager.self.Spec = config.Spec().Manager.Component
	fetchComponentInfo(&componentManager.self)
	return componentManager
}

func fetchComponentInfo(ci *ComponentInstance) error {
	sysConfig := config.Get()
	cfg := utils.UpgradeConfig{
		PackageName: ci.Spec.Name,
		PackageDir:  filepath.Join(env.CostrictDir, "package"),
		InstallDir:  filepath.Join(env.CostrictDir, "bin"),
		BaseUrl:     sysConfig.Cloud.UpgradeUrl,
	}
	cfg.Correct()

	local, err := utils.GetLocalVersion(cfg)
	if err == nil {
		ci.localVersion = local
		ci.Installed = true
		ci.LocalVersion = utils.PrintVersion(local)
	}
	plat, err := utils.GetRemoteVersions(cfg)
	if err == nil {
		ci.remoteVerion = plat.Newest.VersionId
		ci.RemoteVersion = utils.PrintVersion(ci.remoteVerion)
		ci.RemotePlatform = &plat
		if utils.CompareVersion(ci.localVersion, ci.remoteVerion) < 0 {
			ci.NeedUpgrade = true
		}
	}
	return nil
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
func (cm *ComponentManager) UpgradeComponent(name string) error {
	component, ok := cm.components[name]
	if !ok {
		return fmt.Errorf("component %s not found", name)
	}
	if !component.NeedUpgrade {
		return nil
	}
	return cm.upgradeComponent(component)
}

func (cm *ComponentManager) upgradeComponent(component *ComponentInstance) error {
	// 解析版本号 - 由于新结构体中没有版本信息，使用默认版本
	upgradeCfg := utils.UpgradeConfig{
		PackageName: component.Spec.Name,
		BaseUrl:     config.Get().Cloud.UpgradeUrl,
	}
	if component.Spec.InstallDir != "" {
		upgradeCfg.InstallDir = filepath.Join(env.CostrictDir, component.Spec.InstallDir)
	}
	upgradeCfg.Correct()

	retVer, err := utils.UpgradePackage(upgradeCfg, component.localVersion, nil)
	if err != nil {
		logger.Errorf("The '%s' upgrade failed: %v", component.Spec.Name, err)
		return err
	}
	component.remoteVerion = retVer
	component.RemoteVersion = utils.PrintVersion(retVer)
	if utils.CompareVersion(retVer, component.localVersion) == 0 {
		logger.Infof("The '%s' version is up to date\n", component.Spec.Name)
	} else {
		logger.Infof("The '%s' is upgraded to version %s\n", component.Spec.Name, component.RemoteVersion)
	}
	return err
}

/**
* Remove specified component
* @param {string} name - Name of the component to remove
* @returns {error} Returns error if removal fails, nil on success
* @description
* - Finds component by name in component manager
* - Checks if component is installed before removal
* - Uses RemovePackage function to remove component files and metadata
* - Updates component manager state after successful removal
* @throws
* - Component not found errors
* - Package removal errors
 */
func (cm *ComponentManager) RemoveComponent(name string) error {
	component, ok := cm.components[name]
	if !ok {
		return fmt.Errorf("component %s not found", name)
	}

	// Check if component is installed
	if !component.Installed {
		return fmt.Errorf("component %s is not installed", name)
	}
	// Remove the package
	if err := utils.RemovePackage(env.CostrictDir, name); err != nil {
		return fmt.Errorf("failed to remove component %s: %v", name, err)
	}

	// Update component state
	component.Installed = false
	component.LocalVersion = ""
	component.localVersion = utils.VersionNumber{}
	component.NeedUpgrade = false

	logger.Infof("Component '%s' removed successfully", name)
	return nil
}

/**
 * Get all components derived from services
 * @returns {([]ComponentInstance, error)} Returns slice of component information and error if any
 * @description
 * - Converts service configurations to component information
 * - Each service becomes a component with name, version and path
 * - Returns empty slice if no services exist
 * @throws
 * - Component conversion errors
 */
func (cm *ComponentManager) GetComponents() []*ComponentInstance {
	components := make([]*ComponentInstance, 0)
	components = append(components, &cm.self)
	for _, cpn := range cm.components {
		components = append(components, cpn)
	}
	return components
}

func (cm *ComponentManager) GetSelf() *ComponentInstance {
	return &cm.self
}

func (cm *ComponentManager) GetComponent(name string) *ComponentInstance {
	component, ok := cm.components[name]
	if !ok {
		return nil
	}
	return component
}

func (cm *ComponentManager) UpgradeAll() error {
	for _, cpn := range cm.components {
		if cpn.NeedUpgrade {
			cm.upgradeComponent(cpn)
		}
	}
	return nil
}
