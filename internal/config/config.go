package config

import (
	"costrict-keeper/internal/env"
	"costrict-keeper/internal/utils"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

/**
 * Server configuration parameters
 * @property {string} address - Server listening address (e.g. ":8080")
 * @property {string} mode - Application mode (debug/release/test)
 * @property {int} midnightRoosterStartHour - Midnight rooster start hour (default: 3)
 * @property {int} midnightRoosterEndHour - Midnight rooster end hour (default: 5)
 * @property {int} monitoringInterval - Monitoring execution interval in seconds (default: 300)
 * @property {int} metricsReportInterval - Metrics report execution interval in seconds, <=0 means disable (default: 300)
 * @property {int} logReportInterval - Log report execution interval in seconds, <=0 means disable (default: 0)
 */
type ServerConfig struct {
	Address                  string `json:"address"`
	Mode                     string `json:"mode"`
	MidnightRoosterStartHour int    `json:"midnight_rooster_start_hour"`
	MidnightRoosterEndHour   int    `json:"midnight_rooster_end_hour"`
	MonitoringInterval       int    `json:"monitoring_interval"`
	MetricsReportInterval    int    `json:"metrics_report_interval"`
	LogReportInterval        int    `json:"log_report_interval"`
}

type ServiceConfig struct {
	MinPort int `json:"min_port"`
	MaxPort int `json:"max_port"`
}

type TunnelConfig struct {
	ProcessName string   `json:"process_name"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	Timeout     int      `json:"timeout"`
}

/**
 * Logging configuration
 * @property {string} level - Log level (debug/info/warn/error)
 * @property {string} path - Log file path
 * @property {int64} maxSize - Maximum log file size in bytes (default: 52428800, which is 50MB)
 */
type LogConfig struct {
	Level   string `json:"level"`
	Path    string `json:"path"`
	MaxSize int64  `json:"maxSize"`
}

type CloudConfig struct {
	BaseUrl        string `json:"base_url"`
	PushgatewayUrl string `json:"pushgateway_url"`
	TunManagerUrl  string `json:"tunman_url"`
	TunnelUrl      string `json:"tunnel_url"`
	UpgradeUrl     string `json:"upgrade_url"`
	PublicKey      string `json:"public_key"`
}

var ErrComponentNotFound = errors.New("component not found")

type AppConfig struct {
	Server  ServerConfig  `json:"server"`
	Service ServiceConfig `json:"service"`
	Tunnel  TunnelConfig  `json:"tunnel"`
	Log     LogConfig     `json:"log"`
	Cloud   CloudConfig   `json:"cloud"`
}

var (
	cfgData *AppConfig
	cfgLock sync.RWMutex
)

/**
 * Load application configuration from JSON file
 * @param {string} configPath - Path to configuration file
 * @returns {error} Returns error if loading fails, nil on success
 */
func (cfg *AppConfig) loadConfig(configPath string) error {
	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var newConfig AppConfig
	if err := json.NewDecoder(file).Decode(&newConfig); err != nil {
		return err
	}
	newConfig.correctConfig()
	*cfg = newConfig
	return nil
}

func (cfg *AppConfig) correctConfig() {
	if cfg.Server.Address == "" {
		cfg.Server.Address = ":8999"
	}
	if cfg.Server.MidnightRoosterStartHour == 0 {
		cfg.Server.MidnightRoosterStartHour = 3
	}
	if cfg.Server.MidnightRoosterEndHour == 0 {
		cfg.Server.MidnightRoosterEndHour = 5
	}
	if cfg.Server.MonitoringInterval == 0 {
		cfg.Server.MonitoringInterval = 300
	}
	if cfg.Server.MetricsReportInterval == 0 {
		cfg.Server.MetricsReportInterval = 300
	}
	// LogReportInterval 默认为 0，表示不上报日志
	if cfg.Cloud.BaseUrl == "" {
		cfg.Cloud.BaseUrl = GetBaseURL()
		if cfg.Cloud.BaseUrl == "" {
			cfg.Cloud.BaseUrl = "https://zgsm.sangfor.com"
		}
	}
	if cfg.Cloud.PushgatewayUrl == "" {
		cfg.Cloud.PushgatewayUrl = fmt.Sprintf("%s/pushgateway", cfg.Cloud.BaseUrl)
	}
	if cfg.Cloud.UpgradeUrl == "" {
		cfg.Cloud.UpgradeUrl = fmt.Sprintf("%s/costrict", cfg.Cloud.BaseUrl)
	}
	if cfg.Cloud.TunnelUrl == "" {
		cfg.Cloud.TunnelUrl = fmt.Sprintf("%s/ws", cfg.Cloud.BaseUrl)
	}
	if cfg.Cloud.TunManagerUrl == "" {
		cfg.Cloud.TunManagerUrl = fmt.Sprintf("%s/tunnel-manager/api/v1", cfg.Cloud.BaseUrl)
	}
	if cfg.Service.MinPort == 0 {
		cfg.Service.MinPort = 9000
	}
	if cfg.Service.MaxPort == 0 {
		cfg.Service.MaxPort = cfg.Service.MinPort + 1000
	}
	if cfg.Tunnel.ProcessName == "" {
		cfg.Tunnel.ProcessName = "cotun"
	}
	if cfg.Tunnel.Command == "" {
		cfg.Tunnel.Command = "{{.ProcessPath}}"
	}
	if len(cfg.Tunnel.Args) == 0 {
		cfg.Tunnel.Args = []string{
			"client",
			"--auth",
			"costrict:zgsm@costrict.ai",
			"{{.RemoteAddr}}",
			"R:{{.MappingPort}}:127.0.0.1:{{.LocalPort}}",
		}
	}
	// 设置默认日志配置
	if cfg.Log.Level == "" {
		cfg.Log.Level = "debug"
	}
	if cfg.Log.Path == "" {
		cfg.Log.Path = "console" // 默认输出到控制台
	}
	if cfg.Log.MaxSize == 0 {
		cfg.Log.MaxSize = 50 * 1024 * 1024 // 默认50M
	}
}

/**
 * Load configuration from specified path
 * @param {string} configPath - Path to configuration file
 * @returns {error} Returns error if loading fails, nil on success
 */
func LoadConfigFromPath(configPath string) error {
	var cfg AppConfig
	if err := cfg.loadConfig(configPath); err != nil {
		return err
	}
	cfg.correctConfig()

	cfgLock.Lock()
	defer cfgLock.Unlock()
	cfgData = &cfg
	utils.SetAvailablePortRange(cfg.Service.MinPort, cfg.Service.MaxPort)
	return nil
}

/**
 * Get configuration instance
 * @returns {AppConfig} Returns configuration instance
 */
func Get() *AppConfig {
	if cfgData != nil {
		return cfgData
	}
	var cfg AppConfig
	cfgFile := filepath.Join(env.CostrictDir, "config", "costrict.json")
	cfg.loadConfig(cfgFile)
	cfg.correctConfig()

	cfgLock.Lock()
	defer cfgLock.Unlock()
	cfgData = &cfg
	utils.SetAvailablePortRange(cfg.Service.MinPort, cfg.Service.MaxPort)
	return cfgData
}
