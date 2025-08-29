package utils

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
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
	// 根据操作系统选择不同的进程枚举方式
	if runtime.GOOS == "windows" {
		return killProcessWindows(processName)
	} else {
		return killProcessUnix(processName)
	}
}

/**
 * Kill processes on Windows system
 * @param {[]string} targetProcesses - List of process names to kill
 * @returns {error} Returns error if process killing fails, nil on success
 * @description
 * - Uses tasklist command to enumerate processes
 * - Parses output to find target process PIDs
 * - Kills each found process using taskkill command
 * @throws
 * - Command execution errors
 * - Process kill errors
 */

func killProcessWindows(processName string) error {
	log.Printf("Looking for process: %s\n", processName)

	selfPid := os.Getpid()

	// 使用tasklist命令查找进程
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s*", processName), "/FO", "CSV", "/NH")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to list processes for %s: %v\n", processName, err)
		return err
	}

	// 解析输出，获取PID
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// CSV格式: "进程名","PID","会话名","会话#","内存使用"
		fields := strings.Split(line, "\",\"")
		if len(fields) >= 2 {
			// 移除引号
			procName := strings.Trim(fields[0], "\"")
			pidStr := strings.Trim(fields[1], "\"")

			// 检查进程名是否匹配
			if strings.Contains(strings.ToLower(procName), strings.ToLower(processName)) {
				pid, err := strconv.Atoi(pidStr)
				if err != nil {
					log.Printf("Failed to parse PID %s for process %s: %v\n", pidStr, procName, err)
					continue
				}
				if pid == selfPid {
					continue
				}

				// 杀死进程
				log.Printf("Killing process %s (PID: %d)\n", procName, pid)
				killCmd := exec.Command("taskkill", "/F", "/PID", pidStr)
				if err := killCmd.Run(); err != nil {
					log.Printf("Failed to kill process %s (PID: %d): %v\n", procName, pid, err)
				} else {
					log.Printf("Successfully killed process %s (PID: %d)\n", procName, pid)
				}
			}
		}
	}
	return nil
}

/**
 * Kill processes on Unix system (Linux/macOS)
 * @param {[]string} targetProcesses - List of process names to kill
 * @returns {error} Returns error if process killing fails, nil on success
 * @description
 * - Uses ps command to enumerate processes
 * - Parses output to find target process PIDs
 * - Kills each found process using kill command
 * @throws
 * - Command execution errors
 * - Process kill errors
 */
func killProcessUnix(processName string) error {
	log.Printf("Looking for process: %s\n", processName)

	selfPid := os.Getpid()
	// 使用ps命令查找进程
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to list processes for %s: %v\n", processName, err)
		return err
	}

	// 解析输出，获取PID
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 11 {
			pidStr := fields[1]
			procName := fields[10]

			// 检查进程名是否匹配
			if strings.Contains(strings.ToLower(procName), strings.ToLower(processName)) {
				pid, err := strconv.Atoi(pidStr)
				if err != nil {
					log.Printf("Failed to parse PID %s for process %s: %v\n", pidStr, procName, err)
					continue
				}
				if pid == selfPid {
					continue
				}

				// 杀死进程
				log.Printf("Killing process %s (PID: %d)\n", procName, pid)
				killCmd := exec.Command("kill", "-9", pidStr)
				if err := killCmd.Run(); err != nil {
					log.Printf("Failed to kill process %s (PID: %d): %v\n", procName, pid, err)
				} else {
					log.Printf("Successfully killed process %s (PID: %d)\n", procName, pid)
				}
			}
		}
	}
	return nil
}
