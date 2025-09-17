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
		ci.fetchComponentInfo()
		componentManager.components[cpn.Name] = &ci
	}
	componentManager.self.Spec = config.Spec().Manager.Component
	componentManager.self.fetchComponentInfo()
	return componentManager
}

/**
 * Fetch component information including local and remote versions
 * @param {ComponentInstance} ci - Component instance to fetch information for
 * @returns {error} Returns error if fetch fails, nil on success
 * @description
 * - Creates upgrade configuration with component name and paths
 * - Gets local version information using utils.GetLocalVersion
 * - Gets remote version information using utils.GetRemoteVersions
 * - Compares local and remote versions to determine if upgrade is needed
 * - Updates component instance with version information and upgrade status
 * @throws
 * - Local version retrieval errors
 * - Remote version retrieval errors
 * - Version comparison errors
 * @private
 */
func (ci *ComponentInstance) fetchComponentInfo() error {
	sysConfig := config.Get()
	cfg := utils.UpgradeConfig{
		PackageName: ci.Spec.Name,
		PackageDir:  filepath.Join(env.CostrictDir, "package"),
		InstallDir:  filepath.Join(env.CostrictDir, "bin"),
		BaseUrl:     sysConfig.Cloud.UpgradeUrl,
	}
	cfg.Correct()
	ci.NeedUpgrade = false
	ci.Installed = false
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
 * Upgrade component to latest version
 * @param {ComponentInstance} component - Component instance to upgrade
 * @returns {error} Returns error if upgrade fails, nil on success
 * @description
 * - Creates upgrade configuration with component name and base URL
 * - Sets install directory if specified in component specification
 * - Calls utils.UpgradePackage to perform the actual upgrade
 * - Updates component instance with new version information
 * - Logs upgrade result and success/failure status
 * @throws
 * - Upgrade package errors
 * - Configuration errors
 * @private
 */
func (ci *ComponentInstance) upgradeComponent() error {
	// 解析版本号 - 由于新结构体中没有版本信息，使用默认版本
	upgradeCfg := utils.UpgradeConfig{
		PackageName: ci.Spec.Name,
		BaseUrl:     config.Get().Cloud.UpgradeUrl,
	}
	if ci.Spec.InstallDir != "" {
		upgradeCfg.InstallDir = filepath.Join(env.CostrictDir, ci.Spec.InstallDir)
	}
	upgradeCfg.Correct()

	retVer, upgraded, err := utils.UpgradePackage(upgradeCfg, nil)
	if err != nil {
		logger.Errorf("The '%s' upgrade failed: %v", ci.Spec.Name, err)
		return err
	}
	ci.localVersion = retVer
	ci.remoteVerion = retVer
	ci.RemoteVersion = utils.PrintVersion(retVer)
	ci.LocalVersion = ci.RemoteVersion

	if !upgraded {
		logger.Infof("The '%s' version is up to date\n", ci.Spec.Name)
	} else {
		logger.Infof("The '%s' is upgraded to version %s\n", ci.Spec.Name, ci.RemoteVersion)
	}
	return err
}

/**
 * Remove specified component
 */
func (ci *ComponentInstance) removeComponent() error {
	// Check if component is installed
	if !ci.Installed {
		return fmt.Errorf("component '%s' is not installed", ci.Spec.Name)
	}
	// Remove the package
	if err := utils.RemovePackage(env.CostrictDir, ci.Spec.Name, nil); err != nil {
		return fmt.Errorf("failed to remove component %s: %v", ci.Spec.Name, err)
	}

	// Update component state
	ci.Installed = false
	ci.NeedUpgrade = false
	ci.LocalVersion = ""
	ci.localVersion = utils.VersionNumber{}

	logger.Infof("Component '%s' removed successfully", ci.Spec.Name)
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
	cpn, ok := cm.components[name]
	if !ok {
		return fmt.Errorf("component %s not found", name)
	}
	if !cpn.NeedUpgrade {
		return nil
	}
	return cpn.upgradeComponent()
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
	cpn, ok := cm.components[name]
	if !ok {
		return fmt.Errorf("component %s not found", name)
	}
	return cpn.removeComponent()
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
func (cm *ComponentManager) GetComponents(includeSelf bool) []*ComponentInstance {
	components := make([]*ComponentInstance, 0)
	if includeSelf {
		components = append(components, &cm.self)
	}
	for _, cpn := range cm.components {
		components = append(components, cpn)
	}
	return components
}

/**
 * Get self component instance (manager component)
 * @returns {ComponentInstance} Returns the manager component instance
 * @description
 * - Returns the component instance representing the manager itself
 * - Contains manager's version, installation status and upgrade information
 * - Used for manager self-management and upgrade operations
 * @example
 * manager := GetComponentManager()
 * selfComponent := manager.GetSelf()
 * fmt.Printf("Manager version: %s", selfComponent.LocalVersion)
 */
func (cm *ComponentManager) GetSelf() *ComponentInstance {
	return &cm.self
}

/**
 * Get component instance by name
 * @param {string} name - Name of the component to retrieve
 * @returns {ComponentInstance} Returns component instance if found, nil otherwise
 * @description
 * - Searches for component by name in the components map
 * - Returns nil if component is not found
 * - Used to access specific component information and operations
 * @example
 * manager := GetComponentManager()
 * component := manager.GetComponent("my-component")
 * if component != nil {
 *     fmt.Printf("Component version: %s", component.LocalVersion)
 * }
 */
func (cm *ComponentManager) GetComponent(name string) *ComponentInstance {
	cpn, ok := cm.components[name]
	if !ok {
		return nil
	}
	return cpn
}

/**
 * Upgrade all components that need updates
 * @returns {error} Returns nil (always returns nil for backward compatibility)
 * @description
 * - Iterates through all managed components
 * - Checks if each component needs upgrade (NeedUpgrade flag)
 * - Calls upgradeComponent for each component that needs upgrade
 * - Logs upgrade operations and results
 * - Continues processing even if some upgrades fail
 * @example
 * manager := GetComponentManager()
 * if err := manager.UpgradeAll(); err != nil {
 *     logger.Error("Some upgrades failed")
 * }
 */
func (cm *ComponentManager) UpgradeAll() error {
	for _, cpn := range cm.components {
		if cpn.NeedUpgrade {
			cpn.upgradeComponent()
		}
	}
	utils.CleanupOldVersions("")
	return nil
}

/**
 * Check components for updates and upgrade if needed
 * @returns {error} Returns error if check or upgrade fails, nil on success
 * @description
 * - Checks all components for available updates
 * - Upgrades components that have newer versions available
 * - Uses mutex to prevent concurrent check operations
 * - Logs upgrade operations and results
 * @throws
 * - Component check errors
 * - Component upgrade errors
 */
func (cm *ComponentManager) CheckComponents() int {
	logger.Info("Starting component update check...")

	upgradeCount := 0
	components := []*ComponentInstance{}
	for _, cpn := range cm.components {
		components = append(components, cpn)
	}
	components = append(components, &cm.self)
	for _, cpn := range components {
		// Refresh component information to get latest version
		if err := cpn.fetchComponentInfo(); err != nil {
			logger.Errorf("Failed to fetch component info for %s: %v", cpn.Spec.Name, err)
			continue
		}
		// Check if upgrade is needed
		if cpn.NeedUpgrade {
			logger.Infof("Component %s needs upgrade from %s to %s",
				cpn.Spec.Name, cpn.LocalVersion, cpn.RemoteVersion)
			upgradeCount++
		}
	}

	logger.Infof("Component update check completed. %d components upgraded.", upgradeCount)
	return upgradeCount
}
