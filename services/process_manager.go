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
 * @property {int} maxRestartCount - 最大重启次数
 */
type ProcessInstance struct {
	InstanceName    string                 `json:"instanceName"`
	ProcessName     string                 `json:"processName"`
	Command         string                 `json:"command"`
	Args            []string               `json:"args"`
	WorkDir         string                 `json:"workDir"`
	ExitedCallback  func(*ProcessInstance) `json:"-"` // 监测到进程退出的回调函数
	MaxRestartCount int                    `json:"maxRestartCount"`
	Pid             int                    `json:"pid"`
	Status          string                 `json:"status"`
	RestartCount    int                    `json:"restartCount"`
	StartTime       time.Time              `json:"startTime"`
	LastExitTime    time.Time              `json:"lastExitTime"`
	LastExitReason  string                 `json:"lastExitReason"`
	cancelFunc      context.CancelFunc
	process         *os.Process // 统一的进程对象，用于Wait()
	mutex           sync.RWMutex
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
		MaxRestartCount: 7,
		RestartCount:    0,
		Status:          "exited",
	}
}

/**
 * SetExitedCallback 设置进程退出的事件回调
 * @param {func(*ProcessInstance)} callback - 进程退出的事件回调
 * @description
 * - 设置进程退出时的回调函数
 * - 当监测到进程异常退出时，会调用此回调函数通知Owner处理异常
 * - 回调函数会接收发生异常的ProcessInstance作为参数
 * @example
 * proc.SetExitedCallback(func(p *ProcessInstance) {
 *     fmt.Printf("Process %s exited\n", p.InstanceName)
 * })
 */
func (p *ProcessInstance) SetExitedCallback(callback func(*ProcessInstance)) {
	p.ExitedCallback = callback
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
func (proc *ProcessInstance) AttachProcess(pid int) error {
	proc.mutex.Lock()
	defer proc.mutex.Unlock()

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

	logger.Infof("Process '%s' attached (PID: %d, NAME: %s)", proc.InstanceName, pid, proc.ProcessName)
	// 启动协程监控进程
	go proc.monitorProcess()
	return nil
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
func (proc *ProcessInstance) StartProcess() error {
	proc.mutex.Lock()
	defer proc.mutex.Unlock()

	if proc.Status == "running" {
		return fmt.Errorf("process '%s' is already running", proc.InstanceName)
	}
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
	go proc.monitorProcess()
	return nil
}

/**
 * StopProcess 停止进程
 * @param {ProcessInstance} proc - 进程实例
 * @returns {error} 返回错误信息
 * @description
 * - 停止指定进程
 * - 取消进程上下文，终止进程
 * - 自动从管理器中移除进程
 * - 更新进程状态
 */
func (proc *ProcessInstance) StopProcess() error {
	proc.mutex.Lock()
	defer proc.mutex.Unlock()

	if proc.Status != "running" {
		return fmt.Errorf("process '%s' is not running", proc.InstanceName)
	}
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
func (proc *ProcessInstance) monitorProcess() {
	_, err := proc.process.Wait()

	proc.mutex.Lock()
	defer proc.mutex.Unlock()

	proc.doProcessExited(err)
}

func (proc *ProcessInstance) doProcessExited(err error) {
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
	if proc.ExitedCallback != nil {
		proc.ExitedCallback(proc)
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
func (proc *ProcessInstance) RestartProcess() {
	// 检查重启次数是否超过限制
	if proc.MaxRestartCount > 0 && proc.RestartCount >= proc.MaxRestartCount {
		logger.Warnf("Process '%s' has reached maximum restart count (%d), not restarting",
			proc.InstanceName, proc.MaxRestartCount)
		return
	}
	proc.RestartCount++

	logger.Infof("Process '%s' will restart in %v (restart count: %d)",
		proc.InstanceName, time.Second, proc.RestartCount)

	// 延迟重启
	time.AfterFunc(time.Second, func() {
		proc.mutex.Lock()
		defer proc.mutex.Unlock()

		if proc.Status == "stopped" {
			logger.Infof("Process '%s' stopped by user, needn't restart", proc.InstanceName)
			return
		}
		proc.StartProcess()
	})
}
