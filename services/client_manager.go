package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"costrict-keeper/internal/config"
	"costrict-keeper/internal/env"
	"costrict-keeper/internal/logger"
)

// WellKnownConfig 定义 well-known.json 文件的结构
type WellKnownConfig struct {
	ServerURL string `json:"server_url"`
	Port      int    `json:"port"`
	Version   string `json:"version"`
}

// ClientManager 管理与 costrict 服务器的客户端交互
type ClientManager struct {
	cfg       *config.AppConfig
	client    *http.Client
	serverURL string
}

var (
	clientManager *ClientManager
	clientOnce    sync.Once
)

// GetClientManager 获取客户端管理器单例
func GetClientManager() *ClientManager {
	clientOnce.Do(func() {
		clientManager = &ClientManager{
			cfg:    config.Get(),
			client: &http.Client{Timeout: 30 * time.Second},
		}
	})
	return clientManager
}

/**
 * 检测 costrict 服务器是否正在运行
 * @returns {bool} Returns true if server is running, false otherwise
 * @description
 * - 通过检查默认端口连接性来检测服务器状态
 * - 默认检查端口 8999（从配置中获取）
 * - 使用 TCP 连接测试端口可用性
 * - 超时设置为 5 秒
 * @example
 * manager := GetClientManager()
 * if manager.IsServerRunning() {
 *     logger.Info("Costrict server is running")
 * }
 */
func (cm *ClientManager) IsServerRunning() bool {
	address := cm.cfg.Server.Address
	if address == "" {
		address = ":8999" // 默认地址
	}

	// 如果地址以 : 开头，添加 localhost
	if address[0] == ':' {
		address = "localhost" + address
	}

	// 创建连接检查
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		logger.Debugf("Failed to connect to server at %s: %v", address, err)
		return false
	}
	defer conn.Close()

	logger.Debugf("Successfully connected to server at %s", address)
	return true
}

/**
 * 从 well-known.json 文件获取服务器配置
 * @returns {WellKnownConfig, error} Returns server configuration and error if any
 * @description
 * - 从 costrict 目录下的 share/well-known.json 文件读取服务器配置
 * - 文件包含服务器 URL、端口和版本信息
 * - 如果文件不存在或解析失败，返回错误
 * @throws
 * - File not found error (os.Open)
 * - JSON decoding error (json.Unmarshal)
 * @example
 * manager := GetClientManager()
 * config, err := manager.GetWellKnownConfig()
 * if err != nil {
 *     logger.Errorf("Failed to get well-known config: %v", err)
 *     return
 * }
 * logger.Infof("Server URL: %s", config.ServerURL)
 */
func (cm *ClientManager) GetWellKnownConfig() (*WellKnownConfig, error) {
	wellKnownPath := filepath.Join(env.CostrictDir, "share", "well-known.json")

	// 检查文件是否存在
	if _, err := os.Stat(wellKnownPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("well-known.json file not found at %s", wellKnownPath)
	}

	// 读取文件内容
	file, err := os.Open(wellKnownPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open well-known.json: %w", err)
	}
	defer file.Close()

	// 解析 JSON 内容
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read well-known.json: %w", err)
	}

	var wellKnownConfig WellKnownConfig
	if err := json.Unmarshal(data, &wellKnownConfig); err != nil {
		return nil, fmt.Errorf("failed to parse well-known.json: %w", err)
	}

	// 验证必要字段
	if wellKnownConfig.ServerURL == "" {
		return nil, fmt.Errorf("server_url is required in well-known.json")
	}

	logger.Debugf("Loaded well-known config: %+v", wellKnownConfig)
	return &wellKnownConfig, nil
}

/**
 * 设置服务器 URL
 * @param {string} url - Server URL to set
 * @description
 * - 设置用于 HTTP 请求的服务器基础 URL
 * - 自动添加 http:// 前缀（如果不存在）
 * - 后续的所有 API 请求都将使用此 URL
 * @example
 * manager := GetClientManager()
 * manager.SetServerURL("http://localhost:8999")
 */
func (cm *ClientManager) SetServerURL(url string) {
	if url == "" {
		return
	}

	// 确保 URL 以 http:// 或 https:// 开头
	if url[:7] != "http://" && url[:8] != "https://" {
		url = "http://" + url
	}

	cm.serverURL = url
	logger.Debugf("Server URL set to: %s", url)
}

/**
 * 获取当前服务器 URL
 * @returns {string} Returns current server URL
 * @description
 * - 返回当前设置的服务器 URL
 * - 如果未设置，尝试从 well-known.json 获取
 * - 如果都失败，返回默认 URL
 * @example
 * manager := GetClientManager()
 * url := manager.GetServerURL()
 * logger.Infof("Using server URL: %s", url)
 */
func (cm *ClientManager) GetServerURL() string {
	if cm.serverURL != "" {
		return cm.serverURL
	}

	// 尝试从 well-known.json 获取
	if wellKnownConfig, err := cm.GetWellKnownConfig(); err == nil {
		cm.SetServerURL(wellKnownConfig.ServerURL)
		return cm.serverURL
	}

	// 使用默认配置
	defaultURL := "http://localhost:8999"
	if cm.cfg.Server.Address != "" {
		if cm.cfg.Server.Address[0] == ':' {
			defaultURL = "http://localhost" + cm.cfg.Server.Address
		} else {
			defaultURL = "http://" + cm.cfg.Server.Address
		}
	}

	cm.SetServerURL(defaultURL)
	return cm.serverURL
}

/**
 * 发送 HTTP GET 请求到 costrict 服务器
 * @param {string} endpoint - API endpoint path (e.g., "/api/v1/health")
 * @returns {[]byte, error} Returns response body and error if any
 * @description
 * - 向指定的 API 端点发送 GET 请求
 * - 自动添加认证头（从客户端配置获取）
 * - 处理 HTTP 错误状态码
 * - 超时设置为 30 秒
 * @throws
 * - HTTP request errors (client.Do)
 * - Non-2xx status codes
 * @example
 * manager := GetClientManager()
 * data, err := manager.GetRequest("/healthz")
 * if err != nil {
 *     logger.Errorf("Request failed: %v", err)
 *     return
 * }
 * logger.Infof("Response: %s", string(data))
 */
func (cm *ClientManager) GetRequest(endpoint string) ([]byte, error) {
	url := cm.GetServerURL() + endpoint

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 添加认证头
	headers := config.GetAuthHeaders()
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := cm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	logger.Debugf("GET %s successful: %d", endpoint, resp.StatusCode)
	return body, nil
}

/**
 * 发送 HTTP POST 请求到 costrict 服务器
 * @param {string} endpoint - API endpoint path (e.g., "/api/v1/data")
 * @param {interface{}} data - Request body data to be JSON encoded
 * @returns {[]byte, error} Returns response body and error if any
 * @description
 * - 向指定的 API 端点发送 POST 请求
 * - 自动将请求体序列化为 JSON
 * - 自动添加认证头和 Content-Type 头
 * - 处理 HTTP 错误状态码
 * - 超时设置为 30 秒
 * @throws
 * - JSON encoding errors (json.Marshal)
 * - HTTP request errors (client.Do)
 * - Non-2xx status codes
 * @example
 * manager := GetClientManager()
 * data := map[string]interface{}{"key": "value"}
 * response, err := manager.PostRequest("/api/v1/data", data)
 * if err != nil {
 *     logger.Errorf("Request failed: %v", err)
 *     return
 * }
 * logger.Infof("Response: %s", string(response))
 */
func (cm *ClientManager) PostRequest(endpoint string, data interface{}) ([]byte, error) {
	url := cm.GetServerURL() + endpoint

	// 序列化请求体
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 添加认证头
	headers := config.GetAuthHeaders()
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := cm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	logger.Debugf("POST %s successful: %d", endpoint, resp.StatusCode)
	return body, nil
}

/**
 * 发送 HTTP PUT 请求到 costrict 服务器
 * @param {string} endpoint - API endpoint path (e.g., "/api/v1/data/123")
 * @param {interface{}} data - Request body data to be JSON encoded
 * @returns {[]byte, error} Returns response body and error if any
 * @description
 * - 向指定的 API 端点发送 PUT 请求
 * - 自动将请求体序列化为 JSON
 * - 自动添加认证头和 Content-Type 头
 * - 处理 HTTP 错误状态码
 * - 超时设置为 30 秒
 * @throws
 * - JSON encoding errors (json.Marshal)
 * - HTTP request errors (client.Do)
 * - Non-2xx status codes
 * @example
 * manager := GetClientManager()
 * data := map[string]interface{}{"key": "updated_value"}
 * response, err := manager.PutRequest("/api/v1/data/123", data)
 * if err != nil {
 *     logger.Errorf("Request failed: %v", err)
 *     return
 * }
 * logger.Infof("Response: %s", string(response))
 */
func (cm *ClientManager) PutRequest(endpoint string, data interface{}) ([]byte, error) {
	url := cm.GetServerURL() + endpoint

	// 序列化请求体
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 添加认证头
	headers := config.GetAuthHeaders()
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := cm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	logger.Debugf("PUT %s successful: %d", endpoint, resp.StatusCode)
	return body, nil
}

/**
 * 发送 HTTP DELETE 请求到 costrict 服务器
 * @param {string} endpoint - API endpoint path (e.g., "/api/v1/data/123")
 * @returns {[]byte, error} Returns response body and error if any
 * @description
 * - 向指定的 API 端点发送 DELETE 请求
 * - 自动添加认证头
 * - 处理 HTTP 错误状态码
 * - 超时设置为 30 秒
 * @throws
 * - HTTP request errors (client.Do)
 * - Non-2xx status codes
 * @example
 * manager := GetClientManager()
 * response, err := manager.DeleteRequest("/api/v1/data/123")
 * if err != nil {
 *     logger.Errorf("Request failed: %v", err)
 *     return
 * }
 * logger.Infof("Delete successful: %s", string(response))
 */
func (cm *ClientManager) DeleteRequest(endpoint string) ([]byte, error) {
	url := cm.GetServerURL() + endpoint

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 添加认证头
	headers := config.GetAuthHeaders()
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := cm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	logger.Debugf("DELETE %s successful: %d", endpoint, resp.StatusCode)
	return body, nil
}

/**
 * 获取服务器健康状态
 * @returns {map[string]interface{}, error} Returns health status and error if any
 * @description
 * - 调用 /healthz 端点获取服务器健康状态
 * - 自动解析 JSON 响应
 * - 包含服务版本、启动时间、健康状态等信息
 * @throws
 * - Request errors (from GetRequest)
 * - JSON parsing errors (json.Unmarshal)
 * @example
 * manager := GetClientManager()
 * health, err := manager.GetHealthStatus()
 * if err != nil {
 *     logger.Errorf("Failed to get health status: %v", err)
 *     return
 * }
 * logger.Infof("Server version: %s", health["version"])
 */
func (cm *ClientManager) GetHealthStatus() (map[string]interface{}, error) {
	data, err := cm.GetRequest("/healthz")
	if err != nil {
		return nil, fmt.Errorf("failed to get health status: %w", err)
	}

	var health map[string]interface{}
	if err := json.Unmarshal(data, &health); err != nil {
		return nil, fmt.Errorf("failed to parse health response: %w", err)
	}

	logger.Debugf("Health status: %+v", health)
	return health, nil
}

/**
 * 测试与 costrict 服务器的连接
 * @returns {bool, error} Returns true if connection is successful, false otherwise with error
 * @description
 * - 综合测试服务器连接状态
 * - 首先检查服务器是否正在运行
 * - 然后尝试获取健康状态
 * - 返回连接测试结果和详细信息
 * @throws
 * - Server not running errors
 * - Health check errors
 * @example
 * manager := GetClientManager()
 * connected, err := manager.TestConnection()
 * if err != nil {
 *     logger.Errorf("Connection test failed: %v", err)
 *     return
 * }
 * if connected {
 *     logger.Info("Successfully connected to costrict server")
 * }
 */
func (cm *ClientManager) TestConnection() (bool, error) {
	// 首先检查服务器是否正在运行
	if !cm.IsServerRunning() {
		return false, fmt.Errorf("costrict server is not running")
	}

	// 尝试获取健康状态
	_, err := cm.GetHealthStatus()
	if err != nil {
		return false, fmt.Errorf("health check failed: %w", err)
	}

	logger.Info("Successfully connected to costrict server")
	return true, nil
}
