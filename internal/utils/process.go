package utils

import (
	"fmt"
	"os"
	"strings"
)

func FindProcess(processName string, pid int) (*os.Process, error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, err
	}

	// 获取进程名
	name, err := GetProcessName(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get process name for PID %d: %v", pid, err)
	}

	// 比较进程名（不区分大小写）
	if strings.EqualFold(name, processName) {
		return proc, nil
	}
	return nil, fmt.Errorf("process name mismatch: expected '%s', got '%s'", processName, name)
}

// KillProcess 根据进程名和PID杀死进程
func KillProcess(processName string, pid int) error {
	proc, err := FindProcess(processName, pid)
	if err != nil {
		return err
	}
	if err = proc.Kill(); err != nil {
		return err
	}
	return nil
}

/**
 * Kill processes with specified names: costrict, codebase-indexer, cotun
 * @returns {error} Returns error if process killing fails, nil on success
 * @description
 * - Defines target process names
 * - Finds and kills processes by name for Windows and Unix systems
 * - Logs success and failure for each process
 * @throws
 * - Process enumeration errors
 * - Process kill errors
 */
func KillSpecifiedProcesses(targetProcesses []string) error {
	var last error
	for _, processName := range targetProcesses {
		if err := KillSpecifiedProcess(processName); err != nil {
			last = err
		}
	}
	return last
}

func KillSpecifiedProcess(processName string) error {
	// 在Windows系统上，killSpecifiedProcess函数在process_windows.go中定义
	// 在Unix系统上，killSpecifiedProcess函数在process_unix.go中定义
	return killSpecifiedProcess(processName)
}
