package config

import (
	"errors"

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

var ErrComponentNotFound = errors.New("component not found")

type AppConfig struct {
	Server   ServerConfig    `mapstructure:"server"`
	Log      LogConfig       `mapstructure:"log"`
	Metrics  MetricsConfig   `mapstructure:"metrics"`
	Upgrade  UpgradeConfig   `mapstructure:"upgrade"`
	Services []ServiceConfig `mapstructure:"services"`
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
	return cfg
}

func init() {
	cfg, err := LoadConfig()
	if err == nil {
		Config = *cfg
	}
	collectConfig(&Config)
}
