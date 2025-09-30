package rpc

import (
	"bytes"
	"costrict-keeper/internal/env"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// HTTPClient 定义HTTP客户端接口
type HTTPClient interface {
	Get(path string, params map[string]interface{}) (*HTTPResponse, error)
	Post(path string, data interface{}) (*HTTPResponse, error)
	Put(path string, data interface{}) (*HTTPResponse, error)
	Patch(path string, data interface{}) (*HTTPResponse, error)
	Delete(path string, params map[string]interface{}) (*HTTPResponse, error)
	Close() error
	IsConnected() bool
}

// HTTPConfig 定义HTTP客户端配置
type HTTPConfig struct {
	ServerName string        `json:"server_name"` // 服务器名称，用于创建唯一的通讯端点
	Timeout    time.Duration `json:"timeout"`     // 默认超时时间
	BaseURL    string        `json:"base_url"`    // 基础URL
}

// DefaultHTTPConfig 返回默认HTTP客户端配置
func DefaultHTTPConfig() *HTTPConfig {
	return &HTTPConfig{
		ServerName: "costrict",
		Timeout:    5 * time.Second,
		BaseURL:    "http://localhost",
	}
}

// HTTPResponse 定义HTTP响应结构
type HTTPResponse struct {
	StatusCode int                    `json:"status_code"`
	Headers    map[string][]string    `json:"headers"`
	Body       map[string]interface{} `json:"body"`
	Text       string                 `json:"text"`
	Error      string                 `json:"error"`
}

// buildURL 构建完整的URL
func buildURL(baseURL, path string, params map[string]interface{}) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// 添加路径
	if u.Path == "" {
		u.Path = path
	} else {
		// 确保路径以/结尾，然后拼接
		if !strings.HasSuffix(u.Path, "/") {
			u.Path += "/"
		}
		u.Path += path
	}

	// 添加查询参数
	if params != nil {
		q := u.Query()
		for key, value := range params {
			switch v := value.(type) {
			case string:
				q.Set(key, v)
			case int, int8, int16, int32, int64:
				q.Set(key, fmt.Sprintf("%d", v))
			case uint, uint8, uint16, uint32, uint64:
				q.Set(key, fmt.Sprintf("%d", v))
			case float32, float64:
				q.Set(key, fmt.Sprintf("%f", v))
			case bool:
				q.Set(key, fmt.Sprintf("%t", v))
			default:
				q.Set(key, fmt.Sprintf("%v", v))
			}
		}
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
}

// serializeData 序列化请求数据
func serializeData(data interface{}) (io.Reader, error) {
	if data == nil {
		return nil, nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize data: %w", err)
	}

	return bytes.NewReader(jsonData), nil
}

// deserializeResponse 反序列化响应数据
func deserializeResponse(resp *http.Response) (*HTTPResponse, error) {
	httpResp := &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	defer resp.Body.Close()

	// 如果响应体为空，直接返回
	if len(body) == 0 {
		return httpResp, nil
	}
	httpResp.Text = string(body)
	// 尝试解析JSON
	if err := json.Unmarshal(body, &httpResp.Body); err != nil {
		httpResp.Error = err.Error()
	} else {
		if errorMsg, exists := httpResp.Body["error"]; exists {
			if errorStr, ok := errorMsg.(string); ok {
				httpResp.Error = errorStr
			}
		}
	}
	return httpResp, nil
}

/**
 * Get full path for Unix socket
 * @param {string} socketName - Name of the socket file
 * @param {string} socketDir - Directory for socket file (optional, uses default if empty)
 * @returns {string} Full path to socket file
 * @description
 * - Constructs socket path using provided directory or platform-specific default
 * - Handles cross-platform path construction
 * - Returns path in format: {socketDir}/{socketName}
 * @example
 * path := GetSocketPath("costrict.sock", "")
 * fmt.Printf("Socket path: %s", path)
 */
func GetSocketPath(socketName string, socketDir string) string {
	if socketDir == "" {
		socketDir = filepath.Join(env.CostrictDir, "run")
	}
	return filepath.Join(socketDir, socketName)
}
