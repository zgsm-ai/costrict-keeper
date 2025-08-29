package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"costrict-keeper/internal/config"
	"costrict-keeper/internal/env"
	"costrict-keeper/internal/logger"
	"costrict-keeper/internal/models"
	"costrict-keeper/internal/utils"
)

// PortAllocationRequest 端口分配请求
type PortAllocationRequest struct {
	ClientId   string `json:"clientId"`
	AppName    string `json:"appName"`
	ClientPort int    `json:"clientPort"`
}

// PortAllocationResponse 端口分配响应
type PortAllocationResponse struct {
	ClientId    string `json:"clientId"`
	AppName     string `json:"appName"`
	ClientPort  int    `json:"clientPort"`
	MappingPort int    `json:"mappingPort"`
}

type PortQueryResponse struct {
	MappingPort int `json:"mappingPort"`
}

type TunnelArgs struct {
	LocalPort   int
	MappingPort int
	RemoteAddr  string
	ProcessName string
	ProcessPath string
}

type TunnelInstance struct {
	models.Tunnel
	proc *ProcessInstance
}

type TunnelManager struct {
	daemon     bool
	tunnelsDir string
	tunnels    []*TunnelInstance
	pm         *ProcessManager
}

var tunnelManager *TunnelManager

/**
 * Get singleton instance of TunnelManager
 * @returns {*TunnelManager} Returns the singleton TunnelManager instance
 * @description
 * - Implements singleton pattern to ensure only one TunnelManager exists
 * - Initializes tunnel manager with cache directory, daemon mode and process manager
 * - Loads existing tunnel cache on first creation
 * - Returns existing instance if already initialized
 * @example
 * tunnelMgr := GetTunnelManager()
 * tunnel, err := tunnelMgr.StartTunnel("myapp", 8080)
 */
func GetTunnelManager() *TunnelManager {
	if tunnelManager != nil {
		return tunnelManager
	}
	tm := &TunnelManager{
		tunnelsDir: filepath.Join(env.CostrictDir, "cache", "tunnels"),
		daemon:     env.Daemon,
		pm:         GetProcessManager(),
	}
	tm.loadCache()
	tunnelManager = tm
	return tunnelManager
}

/**
 * Create new tunnel instance with default values
 * @param {string} name - Application name for the tunnel
 * @param {int} port - Local port number for the tunnel
 * @returns {*TunnelInstance} Returns new tunnel instance with initialized values
 * @description
 * - Creates new tunnel with specified name and port
 * - Initializes default values: mapping port 0, HTTP protocol, stopped status
 * - Sets creation time to current time and PID to 0
 * - Tunnel is not started yet, just created with initial configuration
 * @example
 * tunnel := tunnelMgr.newTunnel("myapp", 8080)
 * // Returns: TunnelInstance with name="myapp", localPort=8080, status=stopped
 */
func newTunnel(name string, port int) *TunnelInstance {
	return &TunnelInstance{
		Tunnel: models.Tunnel{
			Name:        name,
			LocalPort:   port,
			MappingPort: 0,
			Protocol:    "http",
			Status:      models.StatusStopped,
			CreatedTime: time.Now(),
			Pid:         0,
		},
	}
}

/**
 * Get title string for tunnel instance
 * @returns {string} Returns formatted title string
 * @description
 * - Creates formatted title with name, local port, and mapping port
 * - Format: {name}:{localPort}->{mappingPort}
 * - Used for logging and display purposes
 * @private
 * @example
 * title := tunnelInstance.getTitle()
 * // Returns: "myapp:8080->9000"
 */
func (ti *TunnelInstance) getTitle() string {
	return fmt.Sprintf("%s:%d->%d", ti.Name, ti.LocalPort, ti.MappingPort)
}

/**
 * Generate cache file name for tunnel instance
 * @param {*TunnelInstance} tun - Tunnel instance to generate cache file name for
 * @returns {string} Returns the full path to the cache file
 * @description
 * - Constructs cache file path using tunnel name and local port
 * - File name format: {name}-{port}.json
 * - Cache files are stored in CostrictDir/cache/tunnels directory
 * @example
 * fname := tunnelMgr.getCacheFname(tunnelInstance)
 * // Returns: /path/to/costrict/cache/tunnels/myapp-8080.json
 */
func (tun *TunnelInstance) getCacheFname() string {
	return filepath.Join(env.CostrictDir, "cache", "tunnels", fmt.Sprintf("%s-%d.json", tun.Name, tun.LocalPort))
}

/**
 * Request port mapping from tunnel manager service
 * @param {*TunnelInstance} tun - Tunnel instance to request mapping for
 * @returns {error} Returns error if request fails, nil on success
 * @description
 * - Creates HTTP client and prepares port allocation request
 * - Includes machine ID, app name and client port in request body
 * - Adds authentication headers from config
 * - Sends POST request to tunnel manager service
 * - Handles HTTP response and error statuses
 * - Parses JSON response and updates tunnel mapping port
 * - Logs detailed error information on failures
 * @throws
 * - JSON marshaling errors for request body
 * - HTTP request creation errors
 * - Network request errors
 * - Non-200 HTTP status codes
 * - JSON parsing errors for response
 * @example
 * err := tm.allocMappingPort(tunnel)
 * if err != nil {
 *     log.Printf("Failed to get port mapping: %v", err)
 *     return err
 * }
 */
func (tun *TunnelInstance) allocMappingPort() error {
	client := &http.Client{}
	tun.MappingPort = 0

	// 创建请求 body
	requestBody := PortAllocationRequest{
		ClientId:   config.GetMachineID(),
		AppName:    tun.Name,
		ClientPort: tun.LocalPort,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", config.Get().Cloud.TunManagerUrl+"/ports", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 获取认证请求头，包含Authorization字段
	authHeaders := config.GetAuthHeaders()
	for key, value := range authHeaders {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("allocMappingPort failed - URL: %s, Body: %s, Error: %v", req.URL.String(), string(jsonBody), err)
		return fmt.Errorf("failed to request tunnel-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 读取响应体内容
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Errorf("Failed to read response body: %v", err)
		} else {
			logger.Errorf("Failed to request URL: %s, Body: %s, Status Code: %d, Response Body: %s", req.URL.String(), string(jsonBody), resp.StatusCode, string(bodyBytes))
		}
		return fmt.Errorf("tunnel-manager returned error status code: %d", resp.StatusCode)
	}

	var result PortAllocationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	tun.MappingPort = result.MappingPort
	logger.Infof("Successfully applied for port mapping, result: %+v", result)
	return nil
}

/**
 * Check single tunnel port mapping for changes
 * @param {string} clientId - Client ID for authentication
 * @param {string} appName - Application name to check
 * @param {int} port - Port number to check
 * @returns {(bool, error)} Returns (needsRestart, error) where needsRestart indicates if tunnel needs restart
 * @description
 * - Sends GET request to tunnel-manager API to get port allocation records
 * - Retrieves local tunnel instance using app name and port
 * - Compares mappingPort from API response with local tunnel MappingPort
 * - Returns true if ports are inconsistent (needs restart), false if consistent
 * - Returns error if tunnel is not found or HTTP request fails
 * - Logs detailed information about check results
 * @throws
 * - HTTP request creation errors
 * - Network request errors
 * - Non-200 HTTP status codes
 * - JSON parsing errors for response
 * - Tunnel not found errors
 * @example
 * needsRestart, err := tm.checkTunnel(tun)
 * if err != nil {
 *     log.Printf("Failed to check tunnel: %v", err)
 *     return false, err
 * }
 * if needsRestart {
 *     log.Printf("Tunnel needs restart")
 * }
 */
func (tun *TunnelInstance) checkTunnel() (bool, error) {
	client := &http.Client{}

	// 构建 API URL
	url := fmt.Sprintf("%s/ports/%s/%s", config.Get().Cloud.TunManagerUrl, config.GetMachineID(), tun.Name)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// 获取认证请求头，包含Authorization字段
	authHeaders := config.GetAuthHeaders()
	for key, value := range authHeaders {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("checkTunnel failed - URL: %s, Error: %v", url, err)
		return false, fmt.Errorf("failed to request tunnel-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 读取响应体内容
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Errorf("Failed to read response body: %v", err)
		} else {
			logger.Errorf("Failed to request URL: %s, Status Code: %d, Response Body: %s", url, resp.StatusCode, string(bodyBytes))
		}
		if resp.StatusCode == http.StatusNotFound {
			return true, nil
		}
		return false, fmt.Errorf("tunnel-manager returned error status code: %d", resp.StatusCode)
	}

	// 解析响应
	var result PortQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	logger.Debugf("Successfully retrieved port allocation records for app [%s], new: %d",
		tun.getTitle(), result.MappingPort)

	// 检查端口映射是否一致
	if tun.MappingPort != result.MappingPort {
		logger.Warnf("Port mapping inconsistent for tunnel [%s, PID:%d], old: %d, new: %d",
			tun.getTitle(), tun.Pid, tun.MappingPort, result.MappingPort)
		return true, nil
	} else {
		return false, nil
	}
}

/**
 * Start tunnel process and initialize connection
 * @param {*TunnelInstance} tunnel - Tunnel instance to start
 * @returns {error} Returns error if any step fails, nil on success
 * @description
 * - Sets tunnel status to error initially (for safety)
 * - Requests port mapping from tunnel manager service
 * - Creates process instance with tunnel configuration
 * - Sets restart callback to update PID and save tunnel on restart
 * - Starts tunnel process via process manager
 * - Updates tunnel status, PID and creation time on success
 * - Saves tunnel state to cache via defer function
 * - Logs successful tunnel creation with details
 * @throws
 * - Port mapping request errors
 * - Process instance creation errors
 * - Process start errors
 * @example
 * err := tm.startTunnel(tunnelInstance)
 * if err != nil {
 *     log.Printf("Failed to start tunnel: %v", err)
 *     return err
 * }
 */
func (tunnel *TunnelInstance) startTunnel() error {
	var err error

	defer func() {
		tunnel.saveTunnel()
	}()
	tunnel.Status = models.StatusError

	if err := tunnel.allocMappingPort(); err != nil {
		logger.Errorf("Allocate mapping port failed: %v", err)
		return err
	}

	tunnel.proc, err = tunnel.createProcessInstance()
	if err != nil {
		logger.Errorf("Failed to get command info: %v", err)
		return fmt.Errorf("failed to get command info: %w", err)
	}
	tunnel.proc.SetRestartCallback(func(pi *ProcessInstance) {
		tunnel.Pid = pi.Pid
		tunnel.saveTunnel()
	})
	if err := GetProcessManager().StartProcess(tunnel.proc); err != nil {
		return err
	}
	tunnel.Status = models.StatusRunning
	tunnel.Pid = tunnel.proc.Pid
	tunnel.CreatedTime = tunnel.proc.StartTime

	logger.Infof("Successfully created tunnel (%s), process: %s (PID: %d)",
		tunnel.getTitle(), tunnel.proc.ProcessName, tunnel.Pid)
	return nil
}

/**
 * Stop tunnel process and clean up resources
 * @description
 * - Stops tunnel process via process manager if it exists
 * - Logs success or failure of tunnel stop operation
 * - Frees the local port used by the tunnel
 * - Cleans up tunnel cache and state
 * - Updates tunnel status to stopped and resets PID
 * - Used for graceful tunnel shutdown
 * @private
 * @example
 * tunnelInstance.stopTunnel()
 */
func (tunnel *TunnelInstance) stopTunnel() {
	// 优先使用记录的进程名和PID关闭进程
	if tunnel.proc != nil {
		if err := GetProcessManager().StopProcess(tunnel.proc); err != nil {
			logger.Errorf("Failed to close the tunnel (%s) (PID: %d, NAME: %s): %v",
				tunnel.getTitle(), tunnel.Pid, tunnel.proc.ProcessName, err)
		} else {
			logger.Infof("Successfully closed the tunnel (%s) (PID: %d, NAME: %s)",
				tunnel.getTitle(), tunnel.Pid, tunnel.proc.ProcessName)
		}
	} else {
		logger.Infof("Tunnel (%s) process has stopped", tunnel.getTitle())
	}
	utils.FreePort(tunnel.LocalPort)
	tunnel.cleanTunnel()

	tunnel.Status = models.StatusStopped
	tunnel.Pid = 0
	tunnel.proc = nil
}

/**
 * Restart tunnel with new port mapping
 * @param {*TunnelInstance} tunnel - Tunnel instance to restart
 * @returns {error} Returns error if restart operation fails, nil on success
 * @description
 * - Stops the current tunnel process if running
 * - Updates tunnel status to stopped
 * - Starts the tunnel again with new configuration
 * - Handles errors during stop and start operations
 * @throws
 * - Tunnel stop errors
 * - Tunnel start errors
 * @example
 * err := tm.restartTunnel(tunnelInstance)
 * if err != nil {
 *     log.Printf("Failed to restart tunnel: %v", err)
 *     return err
 * }
 */
func (tunnel *TunnelInstance) restartTunnel() error {
	if tunnel.Status == models.StatusRunning {
		tunnel.stopTunnel()
	}
	return tunnel.startTunnel()
}

/**
 * Get process instance for tunnel execution
 * @param {*TunnelInstance} tunnel - Tunnel instance to create process for
 * @returns {(*ProcessInstance, error)} Returns process instance and error if any
 * @description
 * - Reads tunnel configuration from config
 * - Adjusts process name for Windows (.exe extension)
 * - Creates TunnelArgs with tunnel-specific parameters
 * - Uses text/template to process command and arguments from config
 * - Generates command line with substituted template variables
 * - Returns new ProcessInstance with generated command and args
 * - Template variables include: RemoteAddr, MappingPort, LocalPort, ProcessName, ProcessPath
 * @throws
 * - Command line generation errors
 * @example
 * process, err := tunnel.createProcessInstance()
 * if err != nil {
 *     log.Printf("Failed to create process instance: %v", err)
 *     return nil, err
 * }
 * // Use process instance to start tunnel process
 */
func (tunnel *TunnelInstance) createProcessInstance() (*ProcessInstance, error) {
	cfg := config.Get()
	name := cfg.Tunnel.ProcessName
	if runtime.GOOS == "windows" {
		name = fmt.Sprintf("%s.exe", cfg.Tunnel.ProcessName)
	}
	args := TunnelArgs{
		LocalPort:   tunnel.LocalPort,
		MappingPort: tunnel.MappingPort,
		RemoteAddr:  cfg.Cloud.TunnelUrl,
		ProcessName: name,
		ProcessPath: filepath.Join(env.CostrictDir, "bin", name),
	}
	command, cmdArgs, err := utils.GetCommandLine(cfg.Tunnel.Command, cfg.Tunnel.Args, args)
	if err != nil {
		logger.Errorf("Tunnel startup settings are incorrect, setting: %+v", cfg.Tunnel)
		return nil, err
	}
	tunnel.proc = NewProcessInstance("tunnel "+tunnel.Name, name, command, cmdArgs)
	return tunnel.proc, nil
}

/**
 * Save tunnel instance to cache file
 * @param {*TunnelInstance} tun - Tunnel instance to save
 * @returns {error} Returns error if save operation fails, nil on success
 * @description
 * - Creates cache directory if it doesn't exist
 * - Serializes tunnel instance to JSON format
 * - Writes JSON data to cache file with 0644 permissions
 * - Logs error if save operation fails
 * - Uses inner function for better error handling
 * - File path is generated using getCacheFname()
 * @throws
 * - Directory creation errors
 * - JSON serialization errors
 * - File write errors
 * @example
 * err := tm.saveTunnel(tunnelInstance)
 * if err != nil {
 *     log.Printf("Failed to save tunnel: %v", err)
 * }
 */
func (tun *TunnelInstance) saveTunnel() error {
	err := func() error {
		tunnelsDir := filepath.Join(env.CostrictDir, "cache", "tunnels")
		if err := os.MkdirAll(tunnelsDir, 0755); err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}

		data, err := tun.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize tunnel info: %w", err)
		}
		filePath := tun.getCacheFname()
		if err := os.WriteFile(filePath, []byte(data), 0644); err != nil {
			return fmt.Errorf("failed to write tunnel info file: %w", err)
		}
		return nil
	}()
	if err != nil {
		logger.Errorf("Save tunnel failed: %v", err)
	}
	return err
}

/**
 * Clean tunnel cache file
 * @param {*TunnelInstance} tun - Tunnel instance to clean
 * @returns {error} Returns error if file deletion fails, nil on success
 * @description
 * - Generates cache file path using getCacheFname()
 * - Checks if cache file exists using os.Stat()
 * - Removes cache file if it exists
 * - Logs error if deletion fails
 * - Silently returns if file doesn't exist (no error)
 * - Used when closing tunnels to clean up cached data
 * @throws
 * - File deletion errors
 * @example
 * err := tm.cleanTunnel(tunnelInstance)
 * if err != nil {
 *     log.Printf("Failed to clean tunnel cache: %v", err)
 * }
 */
func (tun *TunnelInstance) cleanTunnel() error {
	filePath := tun.getCacheFname()
	if _, err := os.Stat(filePath); err == nil {
		if err := os.Remove(filePath); err != nil {
			logger.Errorf("Failed to delete cache file: %v", err)
			return fmt.Errorf("failed to delete cache file: %w", err)
		}
	}
	return nil
}

/**
 * Load tunnel instance from cache into memory
 * @param {TunnelInstance} tun - Tunnel instance loaded from cache
 * @description
 * - Searches for existing tunnel with same name and port
 * - Updates existing tunnel if found, appends new one if not found
 * - If tunnel has PID > 0, creates and attaches process instance
 * - Logs successful loading with tunnel details
 * - Silently returns on errors during process attachment
 * @example
 * // Called from loadCache when reading tunnel data from disk
 * tm.loadTunnel(cachedTunnel)
 */
func (tun *TunnelInstance) loadTunnel(cached TunnelInstance) error {
	tun.Tunnel = cached.Tunnel
	if cached.Pid > 0 {
		var err error
		tun.proc, err = tun.createProcessInstance()
		if err != nil {
			tun.proc = nil
			tun.Pid = 0
			tun.Status = models.StatusExited
			return err
		}
		if err = GetProcessManager().AttachProcess(tun.proc, tun.Pid); err != nil {
			tun.proc = nil
			tun.Pid = 0
			tun.Status = models.StatusExited
			return err
		}
	}
	logger.Infof("Successfully loaded tunnel (%s,PID:%d) from cache", tun.getTitle(), tun.Pid)
	return nil
}

/**
 * Get tunnel instance by application name and port
 * @param {string} appName - Application name to search for
 * @param {int} port - Port number to match (0 to match any port)
 * @returns {*TunnelInstance} Returns found tunnel instance or nil if not found
 * @description
 * - Iterates through all managed tunnels
 * - Matches by application name (exact match required)
 * - If port > 0, also matches by local port
 * - If port = 0, returns first tunnel with matching app name
 * - Returns nil if no matching tunnel found
 * @example
 * tunnel := tm.getTunnel("myapp", 8080)    // Get specific tunnel
 * tunnel := tm.getTunnel("myapp", 0)       // Get any tunnel for myapp
 */
func (tm *TunnelManager) getTunnel(appName string, port int) *TunnelInstance {
	for _, tun := range tm.tunnels {
		if tun.Name != appName {
			continue
		}
		if port != 0 && tun.LocalPort != port {
			continue
		}
		return tun
	}
	return nil
}

/**
 * Create or retrieve tunnel instance for application
 * @param {string} appName - Application name for the tunnel
 * @param {int} localPort - Local port number for the tunnel
 * @returns {*TunnelInstance} Returns existing or newly created tunnel instance
 * @description
 * - Searches for existing tunnel with matching name and port
 * - Returns existing tunnel if found
 * - Creates new tunnel using newTunnel() if not found
 * - Adds new tunnel to tunnels list
 * - Does not start the tunnel, just creates the instance
 * @example
 * tunnel := tm.createTunnel("myapp", 8080)
 * // Returns existing tunnel if found, or creates new one
 */
func (tm *TunnelManager) createTunnel(appName string, localPort int) *TunnelInstance {
	for i, tun := range tm.tunnels {
		if tun.Name != appName {
			continue
		}
		if tun.LocalPort != localPort {
			continue
		}
		return tm.tunnels[i]
	}
	tunnel := newTunnel(appName, localPort)
	tm.tunnels = append(tm.tunnels, tunnel)
	return tunnel
}

/**
 * Load tunnel instances from cache directory
 * @returns {error} Returns error if cache directory read fails, nil on success
 * @description
 * - Reads all files from tunnels cache directory
 * - Skips directories and continues on read errors
 * - Unmarshals JSON data into TunnelInstance objects
 * - Loads each valid tunnel instance using loadTunnel()
 * - Returns nil if cache directory doesn't exist (first run)
 * - Silently continues on individual file parsing errors
 * @throws
 * - Cache directory access errors (except ENOENT)
 * @example
 * err := tm.loadCache()
 * if err != nil {
 *     log.Printf("Failed to load tunnel cache: %v", err)
 * }
 */
func (tm *TunnelManager) loadCache() error {
	files, err := os.ReadDir(tm.tunnelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read tunnel cache directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(tm.tunnelsDir, file.Name()))
		if err != nil {
			continue
		}

		var cached TunnelInstance
		if err := json.Unmarshal(data, &cached); err != nil {
			continue
		}
		tun := tm.createTunnel(cached.Name, cached.LocalPort)
		if tun.proc != nil {
			logger.Errorf("Tunnel [%s] already exist", tun.getTitle())
			continue
		}
		tun.loadTunnel(cached)
	}
	return nil
}

/**
 * Start tunnel for application
 * @param {string} appName - Application name that will use the tunnel
 * @param {int} port - If specified, indicates the application already occupies this port. If 0, an available port will be automatically allocated
 * @returns {(*models.Tunnel, error)} Returns created tunnel info and error if any
 * @description
 * - Creates or retrieves a tunnel instance for the specified application
 * - If port is 0, tries to get existing tunnel or creates a new one with available port
 * - If port is specified, creates tunnel for that specific port
 * - Requests port mapping from tunnel manager
 * - Starts the tunnel process with appropriate command and arguments
 * - Updates tunnel status and process information
 * @throws
 * - No available port found (findAvailablePort)
 * - Port mapping request failed (allocMappingPort)
 * - Command info generation failed (createProcessInstance)
 * - Tunnel process start failed (cmd.Start)
 * @example
 * tunnel, err := tunnelService.StartTunnel("myapp", 0)
 * if err != nil {
 *     log.Fatal(err)
 * }
 */
/**
 * Start new tunnel for application
 * @param {string} appName - Name of the application for the tunnel
 * @param {int} port - Local port number for the tunnel
 * @returns {TunnelInstance} Returns created tunnel instance
 * @returns {error} Returns error if start fails, nil on success
 * @description
 * - Checks if tunnel already exists for the app and port
 * - Creates new tunnel instance with specified parameters
 * - Starts tunnel process and initializes connection
 * - Adds tunnel to manager and saves to cache
 * - Returns created tunnel instance
 * @throws
 * - Tunnel already exists errors
 * - Tunnel start errors
 * - Cache save errors
 * @example
 * tunnelManager := GetTunnelManager()
 * tunnel, err := tunnelManager.StartTunnel("myapp", 8080)
 * if err != nil {
 *     logger.Error("Failed to start tunnel:", err)
 * }
 */
func (tm *TunnelManager) StartTunnel(appName string, port int) (*TunnelInstance, error) {
	var tunnel *TunnelInstance

	if port == 0 {
		tunnel = tm.getTunnel(appName, 0)
		if tunnel == nil {
			// No existing tunnel found, create new one with available port
			availablePort, err := utils.AllocPort(0)
			if err != nil {
				logger.Fatalf("no available port found: %v", err)
				return nil, fmt.Errorf("no available port found: %w", err)
			}
			tunnel = tm.createTunnel(appName, availablePort)
		}
	} else { //已经指定端口，说明应用已经占据这个端口了
		tunnel = tm.createTunnel(appName, port)
	}
	if tunnel.Status == models.StatusRunning {
		logger.Infof("Tunnel (%s) has been started, PID: %d", tunnel.getTitle(), tunnel.Pid)
		return tunnel, nil
	}
	if err := tunnel.startTunnel(); err != nil {
		logger.Errorf("Start [%s] failed: %v", tunnel.getTitle(), err)
		return nil, err
	}
	return tunnel, nil
}

/**
 * Close tunnel for specified application and port
 * @param {string} appName - Application name
 * @param {int} port - Port number
 * @returns {error} Returns error if close operation fails, nil on success
 * @description
 * - Retrieves tunnel instance using getTunnel()
 * - Returns error if tunnel doesn't exist
 * - Returns immediately if tunnel is not running
 * - Stops tunnel process using recorded process information
 * - Logs success or failure of process termination
 * - Frees the local port for reuse
 * - Removes tunnel cache file
 * - Resets tunnel status to stopped and clears PID and process reference
 * @throws
 * - Tunnel not found errors
 * - Process stop errors
 * - Cache file deletion errors
 * @example
 * err := tm.CloseTunnel("myapp", 8080)
 * if err != nil {
 *     log.Printf("Failed to close tunnel: %v", err)
 * }
 */
func (tm *TunnelManager) CloseTunnel(appName string, port int) error {
	tunnel := tm.getTunnel(appName, port)
	if tunnel == nil {
		return fmt.Errorf("tunnel [%s:%d] not exist", appName, port)
	}
	if tunnel.Status != models.StatusRunning {
		return nil
	}
	tunnel.stopTunnel()
	return nil
}

/**
 * List all managed tunnels
 * @returns {[]*models.Tunnel} Returns slice of tunnel information
 * @description
 * - Creates new slice to hold tunnel data
 * - Iterates through all managed tunnel instances
 * - Extracts Tunnel struct from each TunnelInstance
 * - Returns slice containing all tunnel information
 * - Does not include process instance details, only tunnel metadata
 * @example
 * tunnels := tm.ListTunnels()
 * for _, tunnel := range tunnels {
 *     fmt.Printf("Tunnel: %s:%d->%d\n", tunnel.Name, tunnel.LocalPort, tunnel.MappingPort)
 * }
 */
/**
 * List all managed tunnels
 * @returns {[]models.Tunnel} Returns slice of tunnel information
 * @description
 * - Creates slice of tunnel information from all managed instances
 * - Excludes internal process instance details
 * - Used for API responses and tunnel listing
 * @example
 * tunnelManager := GetTunnelManager()
 * tunnels := tunnelManager.ListTunnels()
 * for _, tunnel := range tunnels {
 *     fmt.Printf("Tunnel: %s:%d->%d", tunnel.Name, tunnel.LocalPort, tunnel.MappingPort)
 * }
 */
func (tm *TunnelManager) ListTunnels() []*models.Tunnel {
	var tunnels []*models.Tunnel
	for _, t := range tm.tunnels {
		tunnels = append(tunnels, &t.Tunnel)
	}
	return tunnels
}

/**
 * Get tunnel information by application name and port
 * @param {string} appName - Application name to search for
 * @param {int} port - Port number to match
 * @returns {(*models.Tunnel, error)} Returns tunnel info and error if any
 * @description
 * - Uses getTunnel() to find tunnel instance
 * - Returns error if tunnel is not found
 * - Returns Tunnel struct (without process instance details) on success
 * - Used by API handlers to provide tunnel information to clients
 * @throws
 * - Tunnel not found errors
 * @example
 * tunnel, err := tm.GetTunnelInfo("myapp", 8080)
 * if err != nil {
 *     log.Printf("Tunnel not found: %v", err)
 *     return nil, err
 * }
 * fmt.Printf("Tunnel: %s:%d -> %d\n", tunnel.Name, tunnel.LocalPort, tunnel.MappingPort)
 */
func (tm *TunnelManager) GetTunnelInfo(appName string, port int) (*models.Tunnel, error) {
	tunnel := tm.getTunnel(appName, port)
	if tunnel == nil {
		return nil, fmt.Errorf("tunnel not found for app [%s]", appName)
	}
	return &tunnel.Tunnel, nil
}

/**
 * Close all running tunnels
 * @returns {error} Returns the last error encountered, or nil if all tunnels closed successfully
 * @description
 * - Iterates through all managed tunnel instances
 * - Skips tunnels that are not in running state
 * - Calls CloseTunnel() for each running tunnel
 * - Logs errors for individual tunnel close failures
 * - Continues closing remaining tunnels even if some fail
 * - Returns the last error encountered (if any)
 * - Used during application shutdown to clean up all active tunnels
 * @example
 * err := tm.CloseAll()
 * if err != nil {
 *     log.Printf("Some tunnels failed to close: %v", err)
 * }
 */
func (tm *TunnelManager) CloseAll() {
	for _, tunnel := range tm.tunnels {
		if tunnel.Status != models.StatusRunning {
			continue
		}
		if err := tm.CloseTunnel(tunnel.Name, tunnel.LocalPort); err != nil {
			logger.Errorf("Failed to close tunnel (%s): %v", tunnel.getTitle(), err)
		}
	}
}

/**
 * Check tunnel port mappings and restart if necessary
 * @param {string} clientId - Client ID for authentication
 * @param {string} appName - Application name to check tunnels for
 * @returns {error} Returns error if check operation fails, nil on success
 * @description
 * - Iterates through all local tunnels and calls checkTunnel() for each
 * - Restarts tunnel if checkTunnel indicates port mapping changed
 * - Logs detailed information about check results and restart operations
 * - Handles tunnel restart errors
 * @throws
 * - Tunnel restart errors
 * @example
 * err := tm.CheckTunnels("client123", "myapp")
 * if err != nil {
 *     log.Printf("Failed to check tunnels: %v", err)
 *     return err
 * }
 */
func (tm *TunnelManager) CheckTunnels() error {
	// 遍历所有本地隧道
	for _, tunnel := range tm.tunnels {
		needsRestart, err := tunnel.checkTunnel()
		if err != nil {
			// 如果检查失败，记录错误但继续检查下一个隧道
			logger.Errorf("Failed to check tunnel [%s:%d]: %v", tunnel.Name, tunnel.LocalPort, err)
			continue
		}

		// 如果需要重启，则重启隧道
		if needsRestart {
			logger.Warnf("Restarting tunnel [%s:%d] due to port mapping change", tunnel.Name, tunnel.LocalPort)

			// 重启隧道
			if err := tunnel.restartTunnel(); err != nil {
				logger.Errorf("Failed to restart tunnel [%s:%d]: %v", tunnel.Name, tunnel.LocalPort, err)
				return fmt.Errorf("failed to restart tunnel [%s:%d]: %w", tunnel.Name, tunnel.LocalPort, err)
			}

			logger.Infof("Successfully restarted tunnel [%s:%d]", tunnel.Name, tunnel.LocalPort)
		}
	}

	return nil
}
