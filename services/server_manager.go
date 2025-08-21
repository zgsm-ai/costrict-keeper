package services

import (
	"context"
	"time"

	"costrict-keeper/internal/config"
	"costrict-keeper/internal/logger"
)

type Server struct {
	cfg       *config.AppConfig
	service   *ServiceManager
	component *ComponentManager
	tunnel    *TunnelManager
	processor *ProcessManager
}

func NewServer(cfg *config.AppConfig) *Server {
	return &Server{
		cfg:       cfg,
		service:   GetServiceManager(),
		component: GetComponentManager(),
		tunnel:    GetTunnelManager(),
		processor: GetProcessManager(),
	}
}

func (s *Server) Services() *ServiceManager {
	return s.service
}

func (s *Server) Components() *ComponentManager {
	return s.component
}

func (s *Server) StartAllService() {
	s.component.UpgradeAll()
	s.service.StartAll(context.Background())
}

func (s *Server) StartMonitoring() {
	ticker := time.NewTicker(300 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.service.CheckServices()
		s.processor.CheckProcesses()
	}
}

func (s *Server) StartReportMetrics() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.ReportMetrics(); err != nil {
			logger.Errorf("Metrics reporting error: %v", err)
		}
	}
}

func (s *Server) StartLogReporting() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.ReportLogs(); err != nil {
			logger.Errorf("Log reporting error: %v", err)
		}
	}
}

func (s *Server) ReportLogs() error {
	// 实现日志上报逻辑
	return nil
}

func (s *Server) ReportMetrics() error {
	// 实现指标上报逻辑
	// if err := CollectAndPushMetrics(s.cfg.Cloud.PushgatewayUrl); err != nil {
	// 	logger.Errorf("Report Metrics error: %v", err)
	// }
	return nil
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
func (s *Server) StopAllService(ctx context.Context) error {
	var last error

	// 停止所有服务
	s.processor.SetAutoRestart(false)
	s.service.StopAll()
	if err := s.tunnel.CloseAll(); err != nil {
		last = err
	}
	return last
}
