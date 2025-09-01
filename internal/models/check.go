package models

import (
	"time"
)

// ServiceCheckResult 服务检查结果
// @Description 服务健康状态检查结果
type ServiceCheckResult struct {
	Name      string `json:"name" example:"costrict" description:"服务名称"`
	Status    string `json:"status" example:"running" description:"服务状态"`
	Pid       int    `json:"pid" example:"1234" description:"进程ID"`
	Port      int    `json:"port" example:"8080" description:"服务端口"`
	StartTime string `json:"startTime" example:"2024-01-01T10:00:00Z" description:"启动时间"`
	Healthy   bool   `json:"healthy" example:"true" description:"是否健康"`
}

// ProcessCheckResult 进程检查结果
// @Description 进程状态检查结果
type ProcessCheckResult struct {
	InstanceName   string `json:"instanceName" example:"service-costrict" description:"进程实例名称"`
	ProcessName    string `json:"processName" example:"costrict" description:"进程名称"`
	Status         string `json:"status" example:"running" description:"进程状态"`
	Pid            int    `json:"pid" example:"1234" description:"进程ID"`
	RestartCount   int    `json:"restartCount" example:"0" description:"重启次数"`
	StartTime      string `json:"startTime" example:"2024-01-01T10:00:00Z" description:"启动时间"`
	LastExitTime   string `json:"lastExitTime" example:"2024-01-01T09:00:00Z" description:"最后退出时间"`
	LastExitReason string `json:"lastExitReason" example:"exited normally" description:"最后退出原因"`
	AutoRestart    bool   `json:"autoRestart" example:"true" description:"是否自动重启"`
}

// TunnelCheckResult 隧道检查结果
// @Description 隧道状态检查结果
type TunnelCheckResult struct {
	Name        string `json:"name" example:"myapp" description:"隧道名称"`
	LocalPort   int    `json:"localPort" example:"8080" description:"本地端口"`
	MappingPort int    `json:"mappingPort" example:"30001" description:"映射端口"`
	Status      string `json:"status" example:"running" description:"隧道状态"`
	Pid         int    `json:"pid" example:"1235" description:"隧道进程ID"`
	CreatedTime string `json:"createdTime" example:"2024-01-01T10:00:00Z" description:"创建时间"`
}

// ComponentCheckResult 组件检查结果
// @Description 组件状态检查结果
type ComponentCheckResult struct {
	Name          string `json:"name" example:"costrict" description:"组件名称"`
	LocalVersion  string `json:"localVersion" example:"1.0.0" description:"本地版本"`
	RemoteVersion string `json:"remoteVersion" example:"1.1.0" description:"远程版本"`
	Installed     bool   `json:"installed" example:"true" description:"是否已安装"`
	NeedUpgrade   bool   `json:"needUpgrade" example:"true" description:"是否需要升级"`
}

// MidnightRoosterCheckResult 半夜鸡叫检查结果
// @Description 半夜鸡叫自动升级检查结果
type MidnightRoosterCheckResult struct {
	Status          string    `json:"status" example:"active" description:"检查状态"`
	NextCheckTime   time.Time `json:"nextCheckTime" example:"2024-01-02T03:30:00Z" description:"下次检查时间"`
	LastCheckTime   time.Time `json:"lastCheckTime" example:"2024-01-01T03:30:00Z" description:"最后检查时间"`
	ComponentsCount int       `json:"componentsCount" example:"5" description:"组件总数"`
	UpgradesNeeded  int       `json:"upgradesNeeded" example:"2" description:"需要升级的组件数"`
}

// CheckResponse 检查API响应结构
// @Description 系统检查API响应数据结构
type CheckResponse struct {
	Timestamp       time.Time                  `json:"timestamp" example:"2024-01-01T10:00:00Z" description:"检查时间戳"`
	Services        []ServiceCheckResult       `json:"services" description:"服务检查结果列表"`
	Processes       []ProcessCheckResult       `json:"processes" description:"进程检查结果列表"`
	Tunnels         []TunnelCheckResult        `json:"tunnels" description:"隧道检查结果列表"`
	Components      []ComponentCheckResult     `json:"components" description:"组件检查结果列表"`
	MidnightRooster MidnightRoosterCheckResult `json:"midnightRooster" description:"半夜鸡叫检查结果"`
	OverallStatus   string                     `json:"overallStatus" example:"healthy" description:"总体状态"`
	TotalChecks     int                        `json:"totalChecks" example:"10" description:"总检查项数"`
	PassedChecks    int                        `json:"passedChecks" example:"8" description:"通过检查项数"`
	FailedChecks    int                        `json:"failedChecks" example:"2" description:"失败检查项数"`
}
