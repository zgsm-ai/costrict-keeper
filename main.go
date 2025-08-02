package main

import (
	_ "costrict-keeper/cmd"
	"costrict-keeper/cmd/root"
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/logger"
	"os"
)

func main() {
	// 检查是否是服务器模式
	isServerMode := len(os.Args) > 1 && os.Args[1] == "server"

	// 根据运行模式初始化日志系统
	if isServerMode {
		logger.InitLoggerWithMode(&config.Config.Log, true)
	} else {
		logger.InitLoggerWithMode(&config.Config.Log, false)
	}

	if err := root.RootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
	os.Exit(0)
}
