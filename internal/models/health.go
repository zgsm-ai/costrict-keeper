package models

// HealthResponse 健康检查响应结构
// @Description 健康检查API响应数据结构
type HealthResponse struct {
	Version   string  `json:"version" example:"1.0.0" description:"服务版本"`
	StartTime string  `json:"startTime" example:"2024-01-01T10:00:00Z" description:"启动时间"`
	Status    string  `json:"status" example:"UP" description:"健康状态"`
	Uptime    string  `json:"uptime" example:"1h30m45s" description:"运行时长"`
	Metrics   Metrics `json:"metrics" description:"关键指标"`
}

// Metrics 关键指标结构
// @Description 系统关键指标数据结构
type Metrics struct {
	TotalRequests      int64 `json:"totalRequests" example:"1000" description:"总请求数"`
	ErrorRequests      int64 `json:"errorRequests" example:"5" description:"出错请求数"`
	ActiveServices     int   `json:"activeServices" example:"3" description:"活跃服务数"`
	ActiveTunnels      int   `json:"activeTunnels" example:"2" description:"活跃隧道数"`
	TotalComponents    int   `json:"totalComponents" example="5" description:"组件总数"`
	UpgradedComponents int   `json:"upgradedComponents" example:"4" description:"已升级组件数"`
}
