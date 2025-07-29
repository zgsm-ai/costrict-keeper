package models

/**
 * Service configuration
 * @property {string} name - Service name
 * @property {string} startup - Startup mode: always/once/none
 * @property {string} command - Startup command
 * @property {string} protocol - Network protocol
 * @property {string} port - Service port
 * @property {string} metrics - Metrics endpoint path
 * @property {string} accessible - Accessible: remote/local
 */
type ServiceConfig struct {
	Name       string `mapstructure:"name" json:"name"`
	Startup    string `mapstructure:"startup" json:"startup"`
	Command    string `mapstructure:"command" json:"command,omitempty"`
	Protocol   string `mapstructure:"protocol" json:"protocol,omitempty"`
	Port       int    `mapstructure:"port" json:"port,omitempty"`
	Metrics    string `mapstructure:"metrics" json:"metrics,omitempty"`
	Accessible string `mapstructure:"accessible" json:"accessible,omitempty"`
}

/**
 * Component configuration
 * @property {string} name - Component name
 * @property {string} version - Version compatibility range
 * @property {UpgradeConfig} upgrade - Upgrade configuration
 */
type ComponentConfig struct {
	Name    string         `mapstructure:"name" json:"name"`
	Version string         `mapstructure:"version" json:"version"`
	Upgrade *UpgradeConfig `mapstructure:"upgrade" json:"upgrade,omitempty"`
}

/**
 * Manager configuration
 * @property {string} name - Manager name
 * @property {string} version - Version compatibility range
 * @property {UpgradeConfig} upgrade - Upgrade configuration
 */
type ManagerConfig struct {
	Name    string         `mapstructure:"name" json:"name"`
	Version string         `mapstructure:"version" json:"version"`
	Upgrade *UpgradeConfig `mapstructure:"upgrade" json:"upgrade"`
}

/**
 * Upgrade configuration
 * @property {string} mode - Upgrade mode: auto/manual
 * @property {string} lowest - Lowest version for forced upgrade
 * @property {string} highest - Highest version limit
 */
type UpgradeConfig struct {
	Mode    string `mapstructure:"mode" json:"mode"`
	Lowest  string `mapstructure:"lowest" json:"lowest"`
	Highest string `mapstructure:"highest" json:"highest"`
}

/**
 * Subsystem definition (system-spec.json)
 * @property {string} configuration - Configuration format version
 * @property {string} platform - Target platform
 * @property {string} arch - Target architecture
 * @property {string} version - Subsystem version
 * @property {ManagerConfig} manager - Service manager configuration
 * @property {[]ComponentConfig} components - Component configurations
 * @property {[]ServiceConfig} services - Service configurations
 */
type SubsystemConfig struct {
	Configuration string            `mapstructure:"configuration" json:"configuration"`
	Platform      string            `mapstructure:"platform" json:"platform"`
	Arch          string            `mapstructure:"arch" json:"arch"`
	Version       string            `mapstructure:"version" json:"version"`
	Manager       ManagerConfig     `mapstructure:"manager" json:"manager"`
	Components    []ComponentConfig `mapstructure:"components" json:"components"`
	Services      []ServiceConfig   `mapstructure:"services" json:"services"`
}
