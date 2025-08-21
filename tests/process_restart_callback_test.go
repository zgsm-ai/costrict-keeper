package tests

import (
	"testing"
	"time"

	"costrict-keeper/internal/logger"
	"costrict-keeper/services"
)

/**
 * TestProcessRestartCallback 测试进程重启回调机制
 * @description
 * - 创建一个进程实例并设置重启回调
 * - 模拟进程重启场景
 * - 验证回调函数是否被正确调用
 * - 验证回调函数接收到的进程信息是否包含更新后的PID
 */
func TestProcessRestartCallback(t *testing.T) {
	// 初始化日志
	logger.InitLogger("console", "info", false, 50*1024*1024) // 50MB

	// 创建进程管理器
	pm := services.GetProcessManager()

	// 创建一个简单的测试进程实例（使用ping命令，它会自然退出）
	process := services.NewProcessInstance(
		"test-process",
		"ping",
		"ping",
		[]string{"-n", "1", "127.0.0.1"}, // Windows ping命令，ping一次后退出
	)

	// 修改默认配置以适应测试需求
	process.WorkDir = ""
	process.AutoRestart = true
	process.MaxRestartCount = 3
	process.RestartDelay = 1 * time.Second

	// 设置回调函数来验证功能
	callbackCalled := false
	var callbackProcess *services.ProcessInstance

	process.SetRestartCallback(func(p *services.ProcessInstance) {
		callbackCalled = true
		callbackProcess = p
		t.Logf("回调被调用: 进程=%s, 新PID=%d, 重启次数=%d", p.ProcessName, p.Pid, p.RestartCount)
	})

	// 启动进程
	err := pm.StartProcess(process)
	if err != nil {
		t.Fatalf("启动进程失败: %v", err)
	}

	// 等待足够时间让进程重启并触发回调
	time.Sleep(5 * time.Second)

	// 验证回调是否被调用
	if !callbackCalled {
		t.Error("进程重启回调函数未被调用")
	}

	// 验证回调接收到的进程信息
	if callbackProcess == nil {
		t.Error("回调接收到的进程实例为nil")
	} else {
		if callbackProcess.ProcessName != "test-process" {
			t.Errorf("回调接收到的进程名称错误: 期望=test-process, 实际=%s", callbackProcess.ProcessName)
		}
		if callbackProcess.Pid <= 0 {
			t.Errorf("回调接收到的PID无效: %d", callbackProcess.Pid)
		}
		if callbackProcess.RestartCount <= 0 {
			t.Errorf("回调接收到的重启次数无效: %d", callbackProcess.RestartCount)
		}
	}

	// 清理：停止进程
	pm.StopProcess(process)

	t.Log("进程重启回调机制测试完成")
}

/**
 * TestProcessRestartCallbackWithoutAutoRestart 测试不自动重启时的回调行为
 * @description
 * - 创建一个不自动重启的进程实例
 * - 设置重启回调
 * - 验证回调函数不会被调用
 */
func TestProcessRestartCallbackWithoutAutoRestart(t *testing.T) {
	// 初始化日志
	logger.InitLogger("console", "info", false, 50*1024*1024) // 50MB

	// 创建进程管理器
	pm := services.GetProcessManager()

	// 创建一个不自动重启的进程实例
	process := services.NewProcessInstance(
		"test-process-no-restart",
		"ping",
		"ping",
		[]string{"-n", "1", "127.0.0.1"},
	)

	// 修改默认配置以适应测试需求
	process.AutoRestart = false

	// 设置回调函数
	callbackCalled := false

	process.SetRestartCallback(func(p *services.ProcessInstance) {
		callbackCalled = true
		t.Log("不自动重启的进程触发了回调，这是意外的")
	})

	// 启动进程
	err := pm.StartProcess(process)
	if err != nil {
		t.Fatalf("启动进程失败: %v", err)
	}

	// 等待进程自然退出
	time.Sleep(3 * time.Second)

	// 验证回调没有被调用
	if callbackCalled {
		t.Error("不自动重启的进程错误地触发了回调函数")
	}

	// 清理
	pm.StopProcess(process)

	t.Log("不自动重启进程的回调行为测试完成")
}
