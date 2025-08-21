package models

/**
 * Service configuration
 * @property {string} name - Service name
 * @property {string} startup - Startup mode: always/once/none
 * @property {string} command - Startup command
 * @property {string} protocol - Network protocol
 * @property {int} port - Service port
 * @property {string} metrics - Metrics endpoint path
 * @property {string} healthy - Health check endpoint path
 * @property {string} accessible - Accessible: remote/local
 */
type ServiceSpecification struct {
	Name       string   `mapstructure:"name" json:"name"`
	Startup    string   `mapstructure:"startup" json:"startup"`
	Command    string   `mapstructure:"command" json:"command,omitempty"`
	Args       []string `mapstructure:"args" json:"args,omitempty"`
	Protocol   string   `mapstructure:"protocol" json:"protocol,omitempty"`
	Port       int      `mapstructure:"port" json:"port,omitempty"`
	Metrics    string   `mapstructure:"metrics" json:"metrics,omitempty"`
	Healthy    string   `mapstructure:"healthy" json:"healthy,omitempty"`
	Accessible string   `mapstructure:"accessible" json:"accessible,omitempty"`
}

/**
 * Component configuration
 * @property {string} name - Component name
 * @property {string} version - Version compatibility range
 * @property {UpgradeSpecification} upgrade - Upgrade configuration
 */
type ComponentSpecification struct {
	Name       string                `mapstructure:"name" json:"name"`
	Version    string                `mapstructure:"version" json:"version"`
	Upgrade    *UpgradeSpecification `mapstructure:"upgrade" json:"upgrade,omitempty"`
	InstallDir string                `mapstructure:"install_dir" json:"install_dir,omitempty"`
}

type ManagerSpecification struct {
	Component ComponentSpecification `mapstructure:"component" json:"component"`
	Service   ServiceSpecification   `mapstructure:"service" json:"service"`
}

/**
 * Upgrade configuration
 * @property {string} mode - Upgrade mode: auto/manual
 * @property {string} lowest - Lowest version for forced upgrade
 * @property {string} highest - Highest version limit
 */
type UpgradeSpecification struct {
	Mode    string `mapstructure:"mode" json:"mode"`
	Lowest  string `mapstructure:"lowest" json:"lowest"`
	Highest string `mapstructure:"highest" json:"highest"`
}

/**
 * System definition (system-spec.json)
 * @property {string} configuration - Configuration format version
 * @property {string} platform - Target platform
 * @property {string} arch - Target architecture
 * @property {string} version - System version
 * @property {ManagerSpecification} manager - Service manager configuration
 * @property {[]ComponentSpecification} components - Component configurations
 * @property {[]ServiceSpecification} services - Service configurations
 */
type SystemSpecification struct {
	Configuration string                   `mapstructure:"configuration" json:"configuration"`
	Platform      string                   `mapstructure:"platform" json:"platform"`
	Arch          string                   `mapstructure:"arch" json:"arch"`
	Version       string                   `mapstructure:"version" json:"version"`
	Manager       ManagerSpecification     `mapstructure:"manager" json:"manager"`
	Components    []ComponentSpecification `mapstructure:"components" json:"components"`
	Services      []ServiceSpecification   `mapstructure:"services" json:"services"`
}
