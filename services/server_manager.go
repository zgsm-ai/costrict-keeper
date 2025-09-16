package services

import (
	"context"
	"math/rand"
	"os"
	"time"

	"costrict-keeper/internal/config"
	"costrict-keeper/internal/env"
	"costrict-keeper/internal/logger"
	"costrict-keeper/internal/models"
	"costrict-keeper/internal/utils"
)

type Server struct {
	cfg               *config.AppConfig
	service           *ServiceManager
	component         *ComponentManager
	startTime         time.Time
	nextMidnightCheck time.Time
}

/**
 * Create new server instance with all managers
 * @param {config.AppConfig} cfg - Application configuration
 * @returns {Server} Returns new server instance
 * @description
 * - Creates and initializes a new Server instance
 * - Initializes all managers: service, component, tunnel, and process
 * - Sets up the server with provided configuration
 * - Used as the main entry point for server operations
 * @example
 * cfg := config.Get()
 * server := NewServer(cfg)
 * server.StartAllService()
 */
func NewServer(cfg *config.AppConfig) *Server {
	return &Server{
		cfg:       cfg,
		service:   GetServiceManager(),
		component: GetComponentManager(),
		startTime: time.Now(),
	}
}

/**
 * Get service manager instance
 * @returns {ServiceManager} Returns the service manager
 * @description
 * - Returns the service manager associated with this server
 * - Used to access service management operations
 * - Provides access to start, stop, and manage services
 * @example
 * server := NewServer(cfg)
 * serviceManager := server.Services()
 * serviceManager.StartAll(context.Background())
 */
func (s *Server) Services() *ServiceManager {
	return s.service
}

/**
 * Get component manager instance
 * @returns {ComponentManager} Returns the component manager
 * @description
 * - Returns the component manager associated with this server
 * - Used to access component management operations
 * - Provides access to upgrade, remove, and manage components
 * @example
 * server := NewServer(cfg)
 * componentManager := server.Components()
 * components := componentManager.GetComponents()
 */
func (s *Server) Components() *ComponentManager {
	return s.component
}

/**
 * Start all services and upgrade components
 * @description
 * - Stops all currently running services
 * - Upgrades all components to latest versions
 * - Starts all services with background context
 * - Used for initial server startup and full restart
 * @example
 * server := NewServer(cfg)
 * server.StartAllService()
 */
func (s *Server) StartAllService() {
	s.service.StopAll()
	s.cleanRemains()
	s.component.UpgradeAll()
	s.service.StartAll(context.Background())
}

func (s *Server) cleanRemains() {
	utils.KillSpecifiedProcess(COSTRICT_NAME)
	for _, svc := range config.Spec().Components {
		utils.KillSpecifiedProcess(svc.Name)
	}
}

/**
 * Stop all services and tunnels gracefully
 * @param {context.Context} ctx - Context for cancellation and timeout
 * @returns {error} Returns error if any service fails to stop, nil on success
 * @description
 * - Stops all running services managed by ServiceManager
 * - Closes all active tunnels managed by TunnelManager
 * - Uses context for timeout control
 * - Logs any errors encountered during shutdown
 * @throws
 * - Service stop errors
 * - Tunnel close errors
 * @example
 * ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
 * defer cancel()
 * if err := server.StopAllService(ctx); err != nil {
 *     logger.Fatal("Failed to stop services:", err)
 * }
 */
func (s *Server) StopAllService(ctx context.Context) {
	s.service.StopAll()
}

/**
 * Start monitoring services, tunnels, and processes
 * @description
 * - Creates ticker with configured monitoring interval
 * - Periodically checks service health status
 * - Periodically checks tunnel connectivity
 * - Periodically checks process status
 * - Runs indefinitely until server shutdown
 * @example
 * go server.StartMonitoring()
 */
func (s *Server) StartMonitoring() {
	interval := time.Duration(s.cfg.Server.MonitoringInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.service.CheckServices()
	}
}

/**
 * Start periodic metrics reporting
 * @description
 * - Checks if metrics reporting is enabled (interval > 0)
 * - Creates ticker with configured metrics report interval
 * - Periodically calls ReportMetrics to send metrics
 * - Logs errors if metrics reporting fails
 * - Runs indefinitely until server shutdown
 * @example
 * go server.StartReportMetrics()
 */
func (s *Server) StartReportMetrics() {
	interval := s.cfg.Server.MetricsReportInterval
	if interval <= 0 {
		logger.Info("Metrics reporting is disabled (interval <= 0)")
		return
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.ReportMetrics(); err != nil {
			logger.Errorf("Metrics reporting error: %v", err)
		}
	}
}

/**
 * Start periodic log reporting
 * @description
 * - Checks if log reporting is enabled (interval > 0)
 * - Creates ticker with configured log report interval
 * - Periodically calls ReportLogs to send logs
 * - Logs errors if log reporting fails
 * - Runs indefinitely until server shutdown
 * @example
 * go server.StartLogReporting()
 */
func (s *Server) StartLogReporting() {
	interval := s.cfg.Server.LogReportInterval
	if interval <= 0 {
		logger.Info("Log reporting is disabled (interval <= 0)")
		return
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.ReportLogs(); err != nil {
			logger.Errorf("Log reporting error: %v", err)
		}
	}
}

/**
 * Start midnight rooster mechanism for automatic upgrade checking
 * @description
 * - Starts a goroutine that schedules upgrade checks between 3-5 AM
 * - Randomly selects a time within the 3-5 AM window each day
 * - Checks for component upgrades and exits if upgrades are needed
 * - Uses time.Ticker for daily scheduling
 * - Logs scheduling and check operations
 * - Runs indefinitely until server shutdown or upgrade detected
 * @example
 * // This is typically called during server startup
 * server.StartMidnightRooster()
 */
func (s *Server) StartMidnightRooster() {
	// 每天午夜检查一次，计算到明天3-5点之间的随机时间
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	logger.Info("Starting midnight rooster mechanism for upgrade checking")

	// 立即执行第一次检查
	s.scheduleMidnightCheck()

	for range ticker.C {
		s.scheduleMidnightCheck()
	}
}

/**
 * Schedule upgrade check for random time between 3-5 AM
 * @description
 * - Calculates random time between 3:00-5:00 AM
 * - Sets up timer for the calculated time
 * - When timer expires, performs upgrade check
 * - If upgrades are needed, exits the application
 * @private
 */
func (s *Server) scheduleMidnightCheck() {
	now := time.Now()

	// 计算明天的日期
	tomorrow := now.Add(24 * time.Hour)

	// 从配置中获取半夜鸡叫起止时间
	startHour := s.cfg.Server.MidnightRoosterStartHour
	endHour := s.cfg.Server.MidnightRoosterEndHour

	// 设置明天的基础时间（开始小时）
	baseTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), startHour, 0, 0, 0, tomorrow.Location())

	// 在配置的时间范围内随机选择一个时间
	maxMinutes := (endHour - startHour) * 60
	randomMinutes := rand.Intn(maxMinutes) // 0 到 (maxMinutes-1) 分钟
	checkTime := baseTime.Add(time.Duration(randomMinutes) * time.Minute)
	// 保存下一次半夜鸡叫的时间
	s.nextMidnightCheck = checkTime

	// 计算从现在到检查时间的等待时间
	waitDuration := checkTime.Sub(now)

	logger.Infof("Scheduled upgrade check for %s (in %v), time range: %d:00-%d:00",
		checkTime.Format("2006-01-02 15:04:05"), waitDuration, startHour, endHour)

	// 设置定时器
	timer := time.NewTimer(waitDuration)

	go func() {
		<-timer.C
		s.performMidnightCheck()
	}()
}

/**
 * Perform the actual upgrade check
 * @description
 * - Checks all components for available upgrades
 * - If any component needs upgrade, logs the finding and exits the application
 * - Uses os.Exit(0) for clean exit, expecting external process to restart
 * @private
 */
func (s *Server) performMidnightCheck() {
	logger.Info("Performing midnight upgrade check...")

	// 检查所有组件是否需要升级
	needsUpgrade := s.component.CheckComponents()

	if needsUpgrade > 0 {
		logger.Info("Components need upgrade, exiting for restart...")
		// 退出程序，等待外部进程重启
		os.Exit(0)
	} else {
		logger.Info("All components are up to date")
	}
}

/**
 * Report logs to remote server
 * @returns {error} Returns error if report fails, nil on success
 * @description
 * - Implements log reporting logic
 * - Currently returns nil (placeholder implementation)
 * - Should be implemented to send logs to configured remote server
 * @example
 * if err := server.ReportLogs(); err != nil {
 *     logger.Error("Log reporting failed:", err)
 * }
 */
func (s *Server) ReportLogs() error {
	// 实现日志上报逻辑
	return nil
}

/**
* Perform comprehensive system check
* @returns {models.CheckResponse} Returns comprehensive system check results
* @description
* - Performs comprehensive system health check including:
*   - Service health status and running state
*   - Process status and auto-restart information
*   - Tunnel connectivity and mapping status
*   - Component versions and upgrade requirements
*   - Midnight rooster automatic upgrade mechanism status
* - Calculates overall system health status based on all checks
* - Aggregates statistics for total, passed, and failed checks
* - Used for system monitoring and health assessment
* @example
* server := NewServer(cfg)
* checkResult := server.Check()
* fmt.Printf("System status: %s, Passed: %d/%d\n",
*     checkResult.OverallStatus, checkResult.PassedChecks, checkResult.TotalChecks)
 */
func (s *Server) Check() models.CheckResponse {
	response := models.CheckResponse{
		Timestamp: time.Now(),
	}

	// 检查服务
	s.service.CheckServices()
	var serviceResults []models.ServiceCheckResult
	for _, svc := range s.service.GetInstances(false) {
		healthy := svc.IsHealthy()

		tunResult := models.TunnelCheckResult{}
		if svc.tun != nil {
			tunResult.Pid = svc.tun.Pid
			tunResult.CreatedTime = svc.tun.CreatedTime.Format(time.RFC3339)
			tunResult.Status = string(svc.tun.Status)
			tunResult.Ports = svc.tun.Pairs
		}
		serviceResults = append(serviceResults, models.ServiceCheckResult{
			Name:           svc.Name,
			Status:         string(svc.Status),
			Pid:            svc.Pid,
			Port:           svc.Port,
			StartTime:      svc.StartTime,
			Healthy:        healthy,
			RestartCount:   svc.proc.RestartCount,
			LastExitTime:   svc.proc.LastExitTime.Format(time.RFC3339),
			LastExitReason: svc.proc.LastExitReason,
			ProcessName:    svc.proc.ProcessName,
			Tunnel:         tunResult,
		})
	}
	response.Services = serviceResults

	// 检查组件
	upgradesNeeded := s.component.CheckComponents()
	var components []models.ComponentCheckResult
	for _, cpn := range s.component.GetComponents(true) {
		components = append(components, models.ComponentCheckResult{
			Name:          cpn.Spec.Name,
			LocalVersion:  cpn.LocalVersion,
			RemoteVersion: cpn.RemoteVersion,
			Installed:     cpn.Installed,
			NeedUpgrade:   cpn.NeedUpgrade,
		})
	}
	response.Components = components

	response.MidnightRooster = models.MidnightRoosterCheckResult{
		Status:          "active",
		NextCheckTime:   s.nextMidnightCheck,
		LastCheckTime:   time.Now(), // 简化处理
		ComponentsCount: len(components),
		UpgradesNeeded:  upgradesNeeded,
	}

	// 计算总体状态
	response.TotalChecks = len(serviceResults) + len(components)
	response.PassedChecks = 0
	response.FailedChecks = 0

	// 统计服务检查结果
	for _, svc := range serviceResults {
		if svc.Healthy && svc.Status == "running" {
			response.PassedChecks++
		} else {
			response.FailedChecks++
		}
	}

	// 统计组件检查结果
	for _, cpn := range components {
		if cpn.Installed && !cpn.NeedUpgrade {
			response.PassedChecks++
		} else {
			response.FailedChecks++
		}
	}

	// 确定总体状态
	if response.FailedChecks == 0 {
		response.OverallStatus = "healthy"
	} else if response.FailedChecks < response.TotalChecks/2 {
		response.OverallStatus = "warning"
	} else {
		response.OverallStatus = "error"
	}

	return response
}

/**
 * Report metrics to remote server
 * @returns {error} Returns error if report fails, nil on success
 * @description
 * - Implements metrics reporting logic
 * - Currently returns nil (placeholder implementation)
 * - Should be implemented to send metrics to pushgateway
 * - Contains commented out CollectAndPushMetrics call
 * @example
 * if err := server.ReportMetrics(); err != nil {
 *     logger.Error("Metrics reporting failed:", err)
 * }
 */
func (s *Server) ReportMetrics() error {
	// 实现指标上报逻辑
	// if err := CollectAndPushMetrics(s.cfg.Cloud.PushgatewayUrl); err != nil {
	// 	logger.Errorf("Report Metrics error: %v", err)
	// }
	return nil
}

/**
* Get health check response for the server
* @returns {models.HealthResponse} Returns health check response with server status and metrics
* @description
* - Calculates server uptime from start time
* - Collects service statistics (active services count)
* - Collects tunnel statistics (active tunnels count)
* - Collects component statistics (total and upgraded components count)
* - Builds comprehensive health response with all metrics
* - Used for health check endpoint and monitoring
* @example
* server := NewServer(cfg)
* health := server.GetHealthz()
* fmt.Printf("Server status: %s, Uptime: %s\n", health.Status, health.Uptime)
 */
func (s *Server) GetHealthz() models.HealthResponse {
	// 计算服务运行时间
	uptime := time.Since(s.startTime)

	// 获取服务统计信息
	activeServices := 0
	activeTunnels := 0
	for _, svc := range s.service.GetInstances(false) {
		if svc.Status == models.StatusRunning {
			activeServices++
			tun := svc.GetTunnel()
			if tun != nil && tun.Status == models.StatusRunning {
				activeTunnels += len(tun.Pairs)
			}
		}
	}

	// 获取组件统计信息
	components := s.component.GetComponents(true)
	totalComponents := len(components)
	upgradedComponents := 0
	for _, cpn := range components {
		if cpn.Installed {
			upgradedComponents++
		}
	}

	// 构建响应
	response := models.HealthResponse{
		Version:   env.Version,
		StartTime: s.startTime.Format(time.RFC3339),
		Status:    "UP",
		Uptime:    uptime.String(),
		Metrics: models.Metrics{
			TotalRequests:      GetTotalRequestCount(),
			ErrorRequests:      GetTotalErrorCount(),
			ActiveServices:     activeServices,
			ActiveTunnels:      activeTunnels,
			TotalComponents:    totalComponents,
			UpgradedComponents: upgradedComponents,
		},
	}

	return response
}
