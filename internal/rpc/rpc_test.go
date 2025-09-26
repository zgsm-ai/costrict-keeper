package rpc

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"costrict-keeper/internal/config"
	"costrict-keeper/internal/env"
	"costrict-keeper/internal/logger"
)

/**
 * Initialize test environment
 * @description
 * - Sets up configuration for testing
 * - Initializes logger system with config settings
 * - Called automatically when test package is loaded
 */
func init() {
	// 初始化配置
	cfg := config.App()

	// 初始化日志系统
	logger.InitLogger(cfg.Log.Path, cfg.Log.Level, false, cfg.Log.MaxSize)
}

/**
 * Test HTTP client creation functionality
 * @param {*testing.T} t - Testing framework instance
 * @description
 * - Creates default HTTP configuration
 * - Sets server name and timeout for testing
 * - Instantiates new HTTP client with test config
 * - Verifies client is not connected initially
 * - Ensures proper cleanup with defer Close()
 * @example
 * // Run this test with: go test -v -run TestHTTPClientCreation
 */
func TestHTTPClientCreation(t *testing.T) {
	// 创建配置
	config := DefaultHTTPConfig()
	config.ServerName = "test-http-client"
	config.Timeout = 5 * time.Second

	// 创建客户端
	client := NewHTTPClient(config)
	defer client.Close()

	// 检查初始连接状态
	if client.IsConnected() {
		t.Error("Client should not be connected initially")
	}
}

/**
 * Test HTTP client with mock server functionality
 * @param {*testing.T} t - Testing framework instance
 * @description
 * - Creates mock HTTP server with different endpoint handlers
 * - Supports GET, POST, PUT, PATCH, DELETE HTTP methods
 * - Tests various API endpoints with different response codes
 * - Validates response status codes and body content
 * - Uses custom HTTP client for testing without Unix socket
 * @example
 * // Run this test with: go test -v -run TestHTTPClientWithMockServer
 */
func TestHTTPClientWithMockServer(t *testing.T) {
	// 创建模拟HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 根据请求方法和路径返回不同的响应
		switch r.Method {
		case "GET":
			if r.URL.Path == "/api/test" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message": "test response"}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		case "POST":
			if r.URL.Path == "/api/create" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"id": 123, "status": "created"}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		case "PUT":
			if r.URL.Path == "/api/update/123" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id": 123, "updated": true}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		case "PATCH":
			if r.URL.Path == "/api/patch/123" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id": 123, "patched": true}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		case "DELETE":
			if r.URL.Path == "/api/delete/123" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"deleted": true}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	// 创建配置，使用模拟服务器的URL
	config := DefaultHTTPConfig()
	config.BaseURL = server.URL
	config.Timeout = 5 * time.Second

	// 创建一个自定义的客户端，直接使用HTTP客户端而不通过Unix socket
	client := &httpClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		transport: &http.Transport{},
		connected: true, // 直接设置为已连接状态
	}
	defer client.Close()

	// 测试GET请求
	resp, err := client.Get("/api/test", nil)
	if err != nil {
		t.Fatalf("Failed to send GET request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if resp.Body["message"] != "test response" {
		t.Errorf("Expected message 'test response', got %v", resp.Body["message"])
	}

	// 测试POST请求
	postData := map[string]interface{}{
		"name":  "test item",
		"value": 42,
	}
	resp, err = client.Post("/api/create", postData)
	if err != nil {
		t.Fatalf("Failed to send POST request: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	if resp.Body["id"] != float64(123) {
		t.Errorf("Expected id 123, got %v", resp.Body["id"])
	}

	// 测试PUT请求
	putData := map[string]interface{}{
		"name": "updated item",
	}
	resp, err = client.Put("/api/update/123", putData)
	if err != nil {
		t.Fatalf("Failed to send PUT request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// 测试PATCH请求
	patchData := map[string]interface{}{
		"value": 100,
	}
	resp, err = client.Patch("/api/patch/123", patchData)
	if err != nil {
		t.Fatalf("Failed to send PATCH request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// 测试DELETE请求
	resp, err = client.Delete("/api/delete/123", nil)
	if err != nil {
		t.Fatalf("Failed to send DELETE request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if !resp.Body["deleted"].(bool) {
		t.Errorf("Expected deleted to be true")
	}
}

/**
 * Test HTTP client with query parameters functionality
 * @param {*testing.T} t - Testing framework instance
 * @description
 * - Creates mock HTTP server that handles query parameters
 * - Server validates query parameters and returns them in response
 * - Tests GET request with query parameters using custom HTTP client
 * - Validates response status code and query parameter values
 * - Ensures proper parameter passing and response parsing
 * @example
 * // Run this test with: go test -v -run TestHTTPClientWithQueryParams
 */
func TestHTTPClientWithQueryParams(t *testing.T) {
	// 创建模拟HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查查询参数
		if r.URL.Path == "/api/search" {
			query := r.URL.Query()
			name := query.Get("name")
			status := query.Get("status")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"query_name": "` + name + `", "query_status": "` + status + `"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 创建配置
	config := DefaultHTTPConfig()
	config.BaseURL = server.URL
	config.Timeout = 5 * time.Second

	// 创建一个自定义的客户端，直接使用HTTP客户端而不通过Unix socket
	client := &httpClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		transport: &http.Transport{},
		connected: true, // 直接设置为已连接状态
	}
	defer client.Close()

	// 测试带查询参数的GET请求
	params := map[string]interface{}{
		"name":   "test",
		"status": "active",
	}
	resp, err := client.Get("/api/search", params)
	if err != nil {
		t.Fatalf("Failed to send GET request with params: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if resp.Body["query_name"] != "test" {
		t.Errorf("Expected query_name 'test', got %v", resp.Body["query_name"])
	}

	if resp.Body["query_status"] != "active" {
		t.Errorf("Expected query_status 'active', got %v", resp.Body["query_status"])
	}
}

/**
 * Test socket path generation functionality
 * @param {*testing.T} t - Testing framework instance
 * @description
 * - Tests default socket path generation using Costrict directory
 * - Validates socket path format and directory structure
 * - Tests custom socket directory functionality
 * - Ensures proper path joining and directory handling
 * @example
 * // Run this test with: go test -v -run TestSocketPathGeneration
 */
func TestSocketPathGeneration(t *testing.T) {
	// 测试默认socket目录（现在使用Costrict目录）
	socketPath := GetSocketPath("test.sock", "")
	expectedPath := filepath.Join(env.CostrictDir, "run", "test.sock")

	if socketPath != expectedPath {
		t.Errorf("Expected socket path %s, got %s", expectedPath, socketPath)
	}

	// 测试自定义socket目录
	customDir := "/tmp/custom"
	socketPath = GetSocketPath("test.sock", customDir)
	expectedPath = filepath.Join(customDir, "test.sock")

	if socketPath != expectedPath {
		t.Errorf("Expected socket path %s, got %s", expectedPath, socketPath)
	}
}

/**
 * Test Costrict project socket path functionality
 * @param {*testing.T} t - Testing framework instance
 * @description
 * - Tests socket path generation specific to Costrict project
 * - Creates temporary directory to simulate Costrict environment
 * - Creates required run directory structure
 * - Validates socket path generation in Costrict context
 * - Restores original Costrict directory after test
 * @example
 * // Run this test with: go test -v -run TestCostrictSocketPath
 */
func TestCostrictSocketPath(t *testing.T) {
	// 测试Costrict项目的socket路径生成
	socketName := "costrict.sock"

	// 创建临时目录模拟Costrict目录
	oldCostrictDir := env.CostrictDir
	tmpDir := t.TempDir()
	env.CostrictDir = tmpDir
	defer func() {
		env.CostrictDir = oldCostrictDir
	}()

	// 创建run目录
	runDir := filepath.Join(tmpDir, "run")
	err := os.MkdirAll(runDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create run directory: %v", err)
	}

	socketPath := GetSocketPath(socketName, "")
	expectedPath := filepath.Join(runDir, socketName)

	if socketPath != expectedPath {
		t.Errorf("Expected socket path %s, got %s", expectedPath, socketPath)
	}
}

/**
 * Benchmark HTTP client performance
 * @param {*testing.B} b - Benchmark testing framework instance
 * @description
 * - Creates mock HTTP server for benchmark testing
 * - Sets up HTTP client with server URL and timeout
 * - Performs warm-up requests before benchmarking
 * - Runs parallel benchmark tests to measure performance
 * - Measures HTTP request throughput and response times
 * @example
 * // Run this benchmark with: go test -bench=BenchmarkHTTPClient -benchmem
 */
func BenchmarkHTTPClient(b *testing.B) {
	// 创建模拟HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "benchmark response"}`))
	}))
	defer server.Close()

	// 创建配置
	config := DefaultHTTPConfig()
	config.BaseURL = server.URL
	config.Timeout = 5 * time.Second

	// 创建一个自定义的客户端，直接使用HTTP客户端而不通过Unix socket
	client := &httpClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		transport: &http.Transport{},
		connected: true, // 直接设置为已连接状态
	}

	// 预热
	for i := 0; i < 100; i++ {
		client.Get("/api/benchmark", nil)
	}

	// 性能测试
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Get("/api/benchmark", nil)
			if err != nil {
				b.Fatalf("HTTP request failed: %v", err)
			}
		}
	})
}
