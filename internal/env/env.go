package env

import (
	"os"
	"path/filepath"
	"runtime"
)

var Daemon bool = false
var ListenPort int = 0

// (default: %USERPROFILE%/.costrict on Windows, $HOME/.costrict on Linux)
var CostrictDir string = GetCostrictDir()

/**
 * Get costrict directory path
 * @returns {string} Returns costrict directory path
 */
func GetCostrictDir() string {
	homeDir := "/root"
	if runtime.GOOS == "windows" {
		homeDir = os.Getenv("USERPROFILE")
	} else {
		homeDir = os.Getenv("HOME")
	}
	return filepath.Join(homeDir, ".costrict")
}
