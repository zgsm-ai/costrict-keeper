package server

import (
	"costrict-keeper/internal/env"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// ListenerConfig 定义监听器配置
/**
 * @typedef {Object} ListenerConfig
 * @property {number} TcpAddr - TCP地址，""表示不监听TCP
 * @property {string} SocketName - Unix socket名称，空表示不监听Unix socket
 * @property {string} SocketDir - Unix socket目录，为空时使用默认目录
 */
type ListenerConfig struct {
	TcpAddr    string // TCP地址，""表示不监听TCP
	SocketName string // Unix socket名称，空表示不监听Unix socket
	SocketDir  string // Unix socket目录，为空时使用默认目录
}

// DefaultListenerConfig 返回默认监听器配置
/**
 * Returns default listener configuration
 * @returns {ListenerConfig} Default listener configuration with TCP port 8080 and socket name "costrict.sock"
 * @description
 * - Creates a default configuration for cross-platform server listening
 * - Sets TCP port to 8080 for HTTP server
 * - Sets socket name for Unix socket communication
 * - Leaves socket directory empty to use platform-specific default
 * @example
 * cfg := DefaultListenerConfig()
 * fmt.Printf("Default port: %d, Socket: %s", cfg.TCPPort, cfg.SocketName)
 */
func DefaultListenerConfig() *ListenerConfig {
	return &ListenerConfig{
		TcpAddr:    "127.0.0.1:8080",
		SocketName: "costrict.sock",
		SocketDir:  "",
	}
}

// CreateListeners 创建TCP和Unix socket监听器
/**
 * Create TCP and Unix socket listeners for cross-platform support
 * @param {ListenerConfig} cfg - Listener configuration
 * @returns {[]net.Listener} Array of created listeners
 * @returns {string} Unix socket path if created
 * @returns {error} Error if listener creation fails
 * @description
 * - Creates TCP listener if TCPPort > 0
 * - Creates Unix socket listener if SocketName is not empty
 * - Automatically determines platform-specific socket directory
 * - Cleans up existing socket files before creating new ones
 * - Sets appropriate file permissions for Unix socket
 * - Supports Windows, Linux, and Darwin platforms
 * @throws
 * - TCP listener creation errors
 * - Unix socket listener creation errors
 * - Socket file cleanup errors
 * @example
 * cfg := DefaultListenerConfig()
 * listeners, socketPath, err := CreateListeners(cfg)
 * if err != nil {
 *     log.Fatal(err)
 * }
 * defer func() {
 *     for _, listener := range listeners {
 *         listener.Close()
 *     }
 * }()
 */
func CreateListeners(cfg *ListenerConfig) ([]net.Listener, string, error) {
	var listeners []net.Listener
	var socketPath string

	// 创建TCP监听器
	if cfg.TcpAddr != "" {
		tcpListener, err := net.Listen("tcp", cfg.TcpAddr)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create TCP listener on %s: %w", cfg.TcpAddr, err)
		}
		listeners = append(listeners, tcpListener)
	}

	// 创建Unix socket监听器
	if cfg.SocketName != "" {
		// 确定socket目录
		socketDir := cfg.SocketDir
		if socketDir == "" {
			socketDir = filepath.Join(env.CostrictDir, "run")
		}

		socketPath = filepath.Join(socketDir, cfg.SocketName)

		// 删除已存在的socket文件
		if _, err := os.Stat(socketPath); err == nil {
			os.Remove(socketPath)
		}

		unixListener, err := net.Listen("unix", socketPath)
		if err != nil {
			// 关闭已创建的TCP监听器
			for _, listener := range listeners {
				listener.Close()
			}
			return nil, "", fmt.Errorf("failed to create Unix socket listener on %s: %w", socketPath, err)
		}

		// 设置socket文件权限
		os.Chmod(socketPath, 0777)

		listeners = append(listeners, unixListener)
	}

	return listeners, socketPath, nil
}
