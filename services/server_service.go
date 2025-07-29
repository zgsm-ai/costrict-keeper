package services

import (
	"costrict-keeper/internal/config"
	"log"
	"time"
)

type ServerService struct {
	cfg *config.AppConfig
}

func NewServerService(cfg *config.AppConfig) *ServerService {
	return &ServerService{cfg: cfg}
}

func (s *ServerService) StartMonitoring(svcManager *ServiceManager) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if err := svcManager.CheckServices(); err != nil {
			log.Printf("Service monitoring error: %v", err)
		}
		if err := s.ReportMetrics(); err != nil {
			log.Printf("Metrics reporting error: %v", err)
		}
	}
}

func (s *ServerService) StartLogReporting() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.ReportLogs(); err != nil {
			log.Printf("Log reporting error: %v", err)
		}
	}
}

func (s *ServerService) ReportLogs() error {
	// 实现日志上报逻辑
	return nil
}

func (s *ServerService) ReportMetrics() error {
	// 实现指标上报逻辑
	return nil
}
