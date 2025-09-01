package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"costrict-keeper/internal/env"
	"costrict-keeper/internal/logger"
	"costrict-keeper/internal/utils"
)

/**
 * ProcessInstance 进程实例信息
 * @property {string} instanceName - 进程实例名称，唯一标识
 * @property {string} processName - 进程列表显示的进程名，processName+pid可以确定一个进程身份，放误杀
 * @property {string} command - 执行命令
 * @property {[]string} args - 命令参数
 * @property {string} workDir - 工作目录
 * @property {int} pid - 进程ID
 * @property {string} status - 进程状态: running/exited/stopped/error
 * @property {int} restartCount - 重启次数
 * @property {time.Time} startTime - 启动时间
 * @property {time.Time} lastExitTime - 最后退出时间
 * @property {string} lastExitReason - 最后退出原因
 * @property {bool} autoRestart - 是否自动重启
 * @property {int} maxRestartCount - 最大重启次数
 * @property {time.Duration} restartDelay - 重启延迟
 * @property {func(*ProcessInstance)} RestartCallback - 进程重启回调函数
 */
type ProcessInstance struct {
	InstanceName    string                 `json:"instanceName"`
	ProcessName     string                 `json:"processName"`
	Command         string                 `json:"command"`
	Args            []string               `json:"args"`
	WorkDir         string                 `json:"workDir"`
	AutoRestart     bool                   `json:"autoRestart"`
	MaxRestartCount int                    `json:"maxRestartCount"`
	RestartDelay    time.Duration          `json:"restartDelay"`
	RestartCallback func(*ProcessInstance) `json:"-"` // 进程重启回调函数
	Pid             int                    `json:"pid"`
	Status          string                 `json:"status"`
	RestartCount    int                    `json:"restartCount"`
	StartTime       time.Time              `json:"startTime"`
	LastExitTime    time.Time              `json:"lastExitTime"`
	LastExitReason  string                 `json:"lastExitReason"`
	cancelFunc      context.CancelFunc     `json:"-"`
	process         *os.Process            `json:"-"` // 统一的进程对象，用于Wait()
}

/**
 * ProcessManager 进程管理器
 * @property {map[string]*ProcessInstance} processes - 进程实例映射
 * @property {sync.RWMutex} mutex - 读写锁
 */
type ProcessManager struct {
	processes   map[string]*ProcessInstance
	mutex       sync.RWMutex
	autoRestart bool
}

var processManager *ProcessManager

/**
 * GetProcessManager 获取进程管理器单例
 * @returns {ProcessManager} 返回进程管理器实例
 * @description
 * - 如果进程管理器已存在，直接返回
 * - 如果不存在，创建新的进程管理器实例
 */
func GetProcessManager() *ProcessManager {
	if processManager != nil {
		return processManager
	}

	pm := &ProcessManager{
		processes:   make(map[string]*ProcessInstance),
		autoRestart: true,
	}

	processManager = pm
	return pm
}

/**
 * NewProcessInstance 创建新的进程实例
 * @param {string} idName - 进程标识名称，可以唯一确定一个进程，即使它重启过
 * @param {string} command - 执行命令
 * @param {[]string} args - 命令参数
 * @returns {ProcessInstance} 返回创建的进程实例
 * @description
 * - 创建并初始化一个新的进程实例
 * - 设置默认的进程状态和属性
 */
func NewProcessInstance(instanceName, processName, command string, args []string) *ProcessInstance {
	return &ProcessInstance{
		InstanceName:    instanceName,
		ProcessName:     processName,
		Command:         command,
		Args:            args,
		WorkDir:         "",
		AutoRestart:     true,
		MaxRestartCount: 7,
		RestartDelay:    time.Second,
		RestartCallback: nil,
		Status:          "exited",
		RestartCount:    0,
	}
}

/**
 * SetRestartCallback 设置进程重启回调函数
 * @param {func(*ProcessInstance)} callback - 重启回调函数
 * @description
 * - 设置进程重启时的回调函数
 * - 当进程重启或重启成功后，会调用此回调函数通知Owner更新关键信息
 * - 回调函数会接收更新后的ProcessInstance作为参数，包含新的PID等信息
 * @example
 * proc.SetRestartCallback(func(p *ProcessInstance) {
 *     fmt.Printf("Process %s restarted with new PID: %d\n", p.InstanceName, p.Pid)
 * })
 */
func (p *ProcessInstance) SetRestartCallback(callback func(*ProcessInstance)) {
	p.RestartCallback = callback
}

func (pm *ProcessManager) SetAutoRestart(autoRestart bool) bool {
	old := pm.autoRestart
	pm.autoRestart = autoRestart
	return old
}

/**
 * StartProcess 启动进程
 * @param {ProcessInstance} proc - 进程实例
 * @returns {error} 返回错误信息
 * @description
 * - 启动指定进程
 * - 自动将进程添加到管理器中
 * - 使用协程监控进程状态
 * - 如果进程配置了自动重启，会在进程退出时自动重启
 * - 更新进程状态
 */
func (pm *ProcessManager) StartProcess(proc *ProcessInstance) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.processes[proc.InstanceName]; exists {
		return fmt.Errorf("process '%s' already exists", proc.InstanceName)
	}
	if proc.Status == "running" {
		return fmt.Errorf("process '%s' is already running", proc.InstanceName)
	}
	if err := pm.startProcess(proc); err != nil {
		return err
	}
	// 将进程添加到管理器
	pm.processes[proc.InstanceName] = proc
	return nil
}

/**
* AttachProcess 根据PID附加到现有进程
* @param {ProcessInstance} proc - 进程实例
* @param {int} pid - 进程ID
* @returns {error} 返回错误信息
* @description
* - 根据PID查找并附加到现有进程
* - 获取进程基本信息并更新进程实例
* - 将进程添加到管理器中
* - 启动协程监控进程退出
* - 如果进程配置了自动重启，会在进程退出时自动重启
* @throws
* - 进程不存在或无法访问
* - 进程名称已存在
* - 获取进程信息失败
 */
func (pm *ProcessManager) AttachProcess(proc *ProcessInstance, pid int) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// 检查进程名称是否已存在
	if _, exists := pm.processes[proc.InstanceName]; exists {
		logger.Warnf("Process name '%s' already exists", proc.InstanceName)
		return fmt.Errorf("process name '%s' already exists", proc.InstanceName)
	}

	// 查找进程对象
	processObj, err := utils.FindProcess(proc.ProcessName, pid)
	if err != nil {
		logger.Warnf("Failed to find process '%s' with PID %d: %v", proc.ProcessName, pid, err)
		return fmt.Errorf("failed to find process '%s' with PID %d: %v", proc.ProcessName, pid, err)
	}

	// 更新进程实例
	proc.Pid = pid
	proc.Status = "running"
	proc.RestartCount = 0
	proc.StartTime = time.Now()
	proc.process = processObj // 保存进程对象

	// 将进程添加到管理器
	pm.processes[proc.InstanceName] = proc

	// 启动协程监控进程
	go pm.monitorProcess(proc)

	logger.Infof("Process '%s' attached (PID: %d, NAME: %s)", proc.InstanceName, pid, proc.ProcessName)
	return nil
}

/**
 * StopProcess 停止进程
 * @param {ProcessInstance} proc - 进程实例
 * @returns {error} 返回错误信息
 * @description
 * - 停止指定名称的进程
 * - 取消进程上下文，终止进程
 * - 自动从管理器中移除进程
 * - 更新进程状态
 */
func (pm *ProcessManager) StopProcess(proc *ProcessInstance) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if proc.Status != "running" {
		return fmt.Errorf("process '%s' is not running", proc.InstanceName)
	}
	err := pm.stopProcess(proc)
	delete(pm.processes, proc.InstanceName)
	logger.Infof("Process '%s' removed from manager", proc.InstanceName)
	return err
}

func (pm *ProcessManager) GetInstances() []*ProcessInstance {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	var procs []*ProcessInstance
	for _, proc := range pm.processes {
		procs = append(procs, proc)
	}
	return procs
}

func (pm *ProcessManager) MonitorProcesses() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for _, proc := range pm.processes {
		if proc.Status != "running" {
			continue
		}
		if proc.Pid == 0 {
			continue
		}
		_, err := utils.FindProcess(proc.ProcessName, proc.Pid)
		if err != nil {
			pm.doProcessStopped(proc, err)
		}
	}
}

func (pm *ProcessManager) CheckProcesses() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for _, proc := range pm.processes {
		if proc.Status != "running" {
			continue
		}
		if proc.Pid == 0 {
			continue
		}
		_, err := utils.FindProcess(proc.ProcessName, proc.Pid)
		if err != nil {
			pm.doProcessStopped(proc, err)
		}
	}
}

func (pm *ProcessManager) startProcess(proc *ProcessInstance) error {
	fullCommand := proc.Command
	for _, arg := range proc.Args {
		fullCommand += " " + arg
	}
	logger.Infof("Executing command: %s", fullCommand)

	// 创建上下文用于控制进程
	ctx, cancel := context.WithCancel(context.Background())
	proc.cancelFunc = cancel

	// 创建命令
	cmd := exec.CommandContext(ctx, proc.Command, proc.Args...)

	// 设置工作目录
	if proc.WorkDir != "" {
		cmd.Dir = proc.WorkDir
	}

	// 设置进程属性，使子进程在父进程退出后继续运行
	if !env.Daemon {
		utils.SetNewPG(cmd)
	}

	if err := cmd.Start(); err != nil {
		proc.Status = "error"
		proc.Pid = 0
		proc.LastExitReason = fmt.Sprintf("start failed: %v", err)
		return fmt.Errorf("failed to start process '%s': %v", proc.InstanceName, err)
	}

	// 更新进程状态
	proc.process = cmd.Process // 保存进程对象，用于统一Wait()
	proc.Pid = cmd.Process.Pid
	proc.Status = "running"
	proc.StartTime = time.Now()

	logger.Infof("Process '%s' started (PID: %d)", proc.InstanceName, proc.Pid)

	// 启动协程监控进程
	go pm.monitorProcess(proc)
	return nil
}

/**
 * stopProcess 停止进程的内部实现
 * @param {ProcessInstance} proc - 进程实例
 * @returns {error} 返回错误信息
 * @description
 * - 取消进程上下文，终止进程
 * - 等待进程退出
 * - 更新进程状态
 */
func (pm *ProcessManager) stopProcess(proc *ProcessInstance) error {
	if proc.cancelFunc != nil {
		proc.cancelFunc()
	}

	proc.Status = "stopped"
	proc.LastExitTime = time.Now()
	proc.LastExitReason = "stopped by user"

	if proc.process != nil {
		if err := proc.process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process '%s': %v", proc.InstanceName, err)
		}
		proc.process.Wait()
	}

	logger.Infof("Process '%s' stopped", proc.InstanceName)
	return nil
}

/**
 * monitorProcess 监控进程状态的协程
 * @param {ProcessInstance} proc - 进程实例
 * @description
 * - 使用协程监控进程状态
 * - 统一使用process.Wait()等待进程退出
 * - 如果进程配置了自动重启，在进程退出时自动重启
 * - 更新进程状态并记录退出原因
 */
func (pm *ProcessManager) monitorProcess(proc *ProcessInstance) {
	_, err := proc.process.Wait()

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.doProcessStopped(proc, err)
}

func (pm *ProcessManager) doProcessStopped(proc *ProcessInstance, err error) {
	if proc.Status == "stopped" {
		logger.Infof("Process '%s' stopped by user", proc.InstanceName)
		return
	}
	proc.LastExitTime = time.Now()
	if err != nil {
		proc.LastExitReason = fmt.Sprintf("exited with error: %v", err)
		proc.Status = "error"
		logger.Errorf("Process '%s' exited with error: %v", proc.InstanceName, err)
	} else {
		proc.LastExitReason = "exited normally"
		proc.Status = "exited"
		logger.Infof("Process '%s' exited normally", proc.InstanceName)
	}

	// 检查是否需要自动重启
	if pm.autoRestart && proc.AutoRestart {
		pm.restartProcess(proc)
	} else {
		if !pm.autoRestart {
			logger.Infof("No automatic restart required")
		}
		if proc.AutoRestart {
			logger.Infof("Process [%s] does not automatically restart", proc.InstanceName)
		}
	}
}

/**
 * restartProcess 重启进程
 * @param {ProcessInstance} proc - 进程实例
 * @description
 * - 检查重启次数是否超过限制
 * - 增加重启计数
 * - 延迟重启进程
 * - 对于附加的进程，无法重启，只记录日志
 */
func (pm *ProcessManager) restartProcess(proc *ProcessInstance) {
	// 检查重启次数是否超过限制
	if proc.MaxRestartCount > 0 && proc.RestartCount >= proc.MaxRestartCount {
		logger.Warnf("Process '%s' has reached maximum restart count (%d), not restarting",
			proc.InstanceName, proc.MaxRestartCount)
		return
	}
	proc.RestartCount++

	logger.Infof("Process '%s' will restart in %v (restart count: %d)",
		proc.InstanceName, proc.RestartDelay, proc.RestartCount)

	// 延迟重启
	time.AfterFunc(proc.RestartDelay, func() {
		pm.mutex.Lock()
		defer pm.mutex.Unlock()

		// 检查进程是否仍然存在且状态为停止
		if curProc, exists := pm.processes[proc.InstanceName]; exists {
			if proc.Status == "stopped" {
				logger.Infof("Process '%s' stopped by user, needn't restart", proc.InstanceName)
				return
			}
			// 重新创建命令和上下文
			pm.startProcess(curProc)
			// 调用重启回调函数通知Owner更新关键信息
			if curProc.RestartCallback != nil {
				logger.Infof("Calling restart callback for process '%s'", curProc.InstanceName)
				curProc.RestartCallback(curProc)
			}
		}
	})
}
