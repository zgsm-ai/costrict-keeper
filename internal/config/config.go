package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

/**
 * Server configuration parameters
 * @property {string} address - Server listening address (e.g. ":8080")
 * @property {string} mode - Application mode (debug/release/test)
 */
type ServerConfig struct {
	Address string `mapstructure:"address"`
	Mode    string `mapstructure:"mode"`
}

/**
 * Logging configuration
 * @property {string} level - Log level (debug/info/warn/error)
 * @property {string} path - Log file path
 */
type LogConfig struct {
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
}

/**
 * Metrics configuration
 * @property {string} pushgateway - Pushgateway address for metrics
 */
type MetricsConfig struct {
	Pushgateway string `mapstructure:"pushgateway"`
}

type UpgradeConfig struct {
	BaseUrl   string `mapstructure:"base_url"`
	PublicKey string `mapstructure:"public_key"`
}

/**
 * Directory configuration
 * @property {string} base - Base path (default: %APPDATA%/.costrict on Windows, /usr/local/.costrict on Linux)
 * @property {string} bin - Application installation path
 * @property {string} package - Package information file path
 * @property {string} share - Shared path
 * @property {string} cache - Cache path
 * @property {string} logs - Log path
 */
type DirectoryConfig struct {
	Base    string `mapstructure:"base"`
	Bin     string `mapstructure:"bin"`
	Package string `mapstructure:"package"`
	Share   string `mapstructure:"share"`
	Cache   string `mapstructure:"cache"`
	Logs    string `mapstructure:"logs"`
}

var ErrComponentNotFound = errors.New("component not found")

type AppConfig struct {
	Server    ServerConfig    `mapstructure:"server"`
	Log       LogConfig       `mapstructure:"log"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	Upgrade   UpgradeConfig   `mapstructure:"upgrade"`
	Directory DirectoryConfig `mapstructure:"directory"`
}

/**
 * Load application configuration from YAML file
 */
func LoadConfig() (*AppConfig, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg AppConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

var Config AppConfig

func collectConfig(cfg *AppConfig) *AppConfig {
	if cfg.Metrics.Pushgateway == "" {
		cfg.Metrics.Pushgateway = "https://zgsm.sangfor.com/pushgateway"
	}
	if cfg.Upgrade.BaseUrl == "" {
		cfg.Upgrade.BaseUrl = "https://zgsm.sangfor.com/shenma"
	}

	// 设置默认日志配置
	if cfg.Log.Level == "" {
		cfg.Log.Level = "warn"
	}
	if cfg.Log.Path == "" {
		cfg.Log.Path = "console" // 默认输出到控制台
	}

	// Set default directory paths
	if cfg.Directory.Base == "" {
		homeDir := "/root"
		if runtime.GOOS == "windows" {
			homeDir = os.Getenv("USERPROFILE")
		} else {
			homeDir = os.Getenv("HOME")
		}
		cfg.Directory.Base = filepath.Join(homeDir, ".costrict")
	}

	// Set default values for other directories based on base path
	basePath := cfg.Directory.Base
	if cfg.Directory.Bin == "" {
		cfg.Directory.Bin = filepath.Join(basePath, "bin")
	}
	if cfg.Directory.Package == "" {
		cfg.Directory.Package = filepath.Join(basePath, "package")
	}
	if cfg.Directory.Share == "" {
		cfg.Directory.Share = filepath.Join(basePath, "share")
	}
	if cfg.Directory.Cache == "" {
		cfg.Directory.Cache = filepath.Join(basePath, "cache")
	}
	if cfg.Directory.Logs == "" {
		cfg.Directory.Logs = filepath.Join(basePath, "logs")
	}

	return cfg
}

func init() {
	cfg, err := LoadConfig()
	if err == nil {
		Config = *cfg
	}
	collectConfig(&Config)
}
