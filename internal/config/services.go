package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

/**
 * Service configuration
 * @property {string} name - Service name
 * @property {VersionConfig} versions - Version compatibility range
 * @property {string} startup - Startup command
 * @property {string} protocol - Network protocol
 * @property {int} port - Service port
 * @property {string} metrics - Metrics endpoint path
 */
type ServiceConfig struct {
	Name     string        `mapstructure:"name" json:"name"`
	Versions VersionConfig `mapstructure:"versions" json:"versions"`
	Startup  string        `mapstructure:"startup" json:"startup"`
	Protocol string        `mapstructure:"protocol" json:"protocol"`
	Port     int           `mapstructure:"port" json:"port"`
	Metrics  string        `mapstructure:"metrics" json:"metrics,omitempty"`
}

type VersionConfig struct {
	Lowest  string `mapstructure:"lowest" json:"lowest"`
	Highest string `mapstructure:"highest" json:"highest"`
}

type RemoteServicesConfig struct {
	Configuration string          `json:"configuration"`
	Platform      string          `json:"platform"`
	Arch          string          `json:"arch"`
	Version       string          `json:"version"`
	Services      []ServiceConfig `json:"services"`
}

func LoadRemoteServicesConfig(url string) (*RemoteServicesConfig, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote config: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var config RemoteServicesConfig
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}
