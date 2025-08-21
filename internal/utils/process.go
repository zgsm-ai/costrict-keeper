package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

func FindProcess(processName string, pid int) error {
	// 检查进程是否存在
	running, err := IsProcessRunning(pid)
	if err != nil {
		return err
	}
	if !running {
		return fmt.Errorf("process with PID %d is not running", pid)
	}

	// 获取进程名
	name, err := GetProcessName(pid)
	if err != nil {
		return fmt.Errorf("failed to get process name for PID %d: %v", pid, err)
	}

	// 比较进程名（不区分大小写）
	baseName := filepath.Base(name)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	if !strings.EqualFold(name, processName) &&
		!strings.EqualFold(baseName, processName) &&
		!strings.EqualFold(nameWithoutExt, processName) {
		return fmt.Errorf("process name mismatch: expected '%s', got '%s'", processName, name)
	}
	return nil
}

// KillProcess 根据进程名和PID杀死进程
func KillProcess(processName string, pid int) error {
	// 检查进程是否存在
	if err := FindProcess(processName, pid); err != nil {
		return err
	}

	// 进程名和PID都匹配，杀死进程
	return KillProcessByPID(pid)
}
