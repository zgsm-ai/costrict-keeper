package config

import (
	"costrict-keeper/internal/env"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

/**
 * Client authentication configuration
 * @property {string} id - Client unique identifier
 * @property {string} name - Client display name
 * @property {string} access_token - JWT access token for authentication
 * @property {string} machine_id - Machine unique identifier
 * @property {string} base_url - Base URL for API endpoints
 */
type ClientConfig struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AccessToken string `json:"access_token"`
	MachineID   string `json:"machine_id"`
	BaseUrl     string `json:"base_url"`
}

var (
	clientConfig ClientConfig
	clientLock   sync.RWMutex
	clientLoaded bool
)

/**
 * Load client configuration from auth.json file
 * @returns {error} Returns error if loading fails, nil on success
 * @description
 * - Loads client authentication configuration from .costrict/share/auth.json
 * - File contains client ID, name, access token, machine ID and base URL
 * - Configuration is cached in memory for subsequent calls
 * @throws
 * - File not found error (os.Stat, os.Open)
 * - JSON decoding error (json.NewDecoder)
 * @example
 * err := LoadClientConfig()
 * if err != nil {
 *     log.Fatal("Failed to load client config:", err)
 * }
 */
func LoadClientConfig() error {
	authPath := filepath.Join(env.CostrictDir, "share", "auth.json")

	// Check if file exists
	if _, err := os.Stat(authPath); os.IsNotExist(err) {
		return fmt.Errorf("auth config file not found: %s", authPath)
	}

	file, err := os.Open(authPath)
	if err != nil {
		return fmt.Errorf("failed to open auth config file: %w", err)
	}
	defer file.Close()

	var newConfig ClientConfig
	if err := json.NewDecoder(file).Decode(&newConfig); err != nil {
		return fmt.Errorf("failed to decode auth config: %w", err)
	}

	clientLock.Lock()
	defer clientLock.Unlock()

	clientConfig = newConfig
	clientLoaded = true

	return nil
}

/**
 * Get client configuration instance
 * @returns {ClientConfig} Returns client configuration instance
 * @description
 * - Returns cached client configuration
 * - If configuration is not loaded, attempts to load it first
 * - Returns empty config if loading fails
 * @example
 * config := GetClientConfig()
 * if config.ID == "" {
 *     log.Println("Client not configured")
 * }
 */
func GetClientConfig() ClientConfig {
	clientLock.RLock()
	if clientLoaded {
		defer clientLock.RUnlock()
		return clientConfig
	}
	clientLock.RUnlock()

	// Try to load config if not loaded yet
	if err := LoadClientConfig(); err != nil {
		// Return empty config on error
		return ClientConfig{}
	}

	clientLock.RLock()
	defer clientLock.RUnlock()
	return clientConfig
}

/**
 * Check if client is configured
 * @returns {bool} Returns true if client is properly configured, false otherwise
 * @description
 * - Checks if client configuration has been loaded and contains required fields
 * - Required fields: ID, AccessToken, MachineID
 * @example
 * if IsClientConfigured() {
 *     // Proceed with authenticated operations
 * }
 */
func IsClientConfigured() bool {
	config := GetClientConfig()
	return config.ID != "" && config.AccessToken != "" && config.MachineID != ""
}

/**
 * Get authentication headers for HTTP requests
 * @returns {map[string]string} Returns map of header names and values
 * @description
 * - Returns standard authentication headers including Authorization bearer token
 * - Headers include: Authorization, Content-Type, Accept
 * @example
 * headers := GetAuthHeaders()
 * for key, value := range headers {
 *     req.Header.Set(key, value)
 * }
 */
func GetAuthHeaders() map[string]string {
	config := GetClientConfig()
	headers := make(map[string]string)

	if config.AccessToken != "" {
		headers["Authorization"] = "Bearer " + config.AccessToken
	}

	headers["Content-Type"] = "application/json"
	headers["Accept"] = "application/json"

	return headers
}

/**
 * Get base URL for API requests
 * @returns {string} Returns base URL or empty string if not configured
 * @description
 * - Returns the configured base URL for API endpoints
 * - Used to construct full API request URLs
 * @example
 * baseURL := GetBaseURL()
 * if baseURL != "" {
 *     apiURL := baseURL + "/api/v1/endpoint"
 * }
 */
func GetBaseURL() string {
	config := GetClientConfig()
	return config.BaseUrl
}

/**
 * Get client machine ID
 * @returns {string} Returns machine ID or empty string if not configured
 * @description
 * - Returns the unique machine identifier from client configuration
 * - Used for machine-specific operations and authentication
 * @example
 * machineID := GetMachineID()
 * if machineID != "" {
 *     // Use machine ID for machine-specific requests
 * }
 */
func GetMachineID() string {
	config := GetClientConfig()
	return config.MachineID
}

/**
 * Get client display name
 * @returns {string} Returns client name or empty string if not configured
 * @description
 * - Returns the human-readable client name from configuration
 * - Used for display purposes and logging
 * @example
 * clientName := GetClientName()
 * log.Printf("Client: %s", clientName)
 */
func GetClientName() string {
	config := GetClientConfig()
	return config.Name
}

/**
 * Get client unique identifier
 * @returns {string} Returns client ID or empty string if not configured
 * @description
 * - Returns the unique client identifier from configuration
 * - Used for client-specific operations and identification
 * @example
 * clientID := GetClientID()
 * if clientID != "" {
 *     // Use client ID for client-specific requests
 * }
 */
func GetClientID() string {
	config := GetClientConfig()
	return config.ID
}
