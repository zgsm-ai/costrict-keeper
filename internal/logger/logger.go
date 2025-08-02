package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"costrict-keeper/internal/config"
)

var (
	defaultLogger *Logger
)

// Logger 日志结构体
type Logger struct {
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
}

// LogLevel 日志级别类型
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// GetLogLevelFromString 将字符串转换为日志级别
func GetLogLevelFromString(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return WARN // 默认级别
	}
}

// InitLogger 初始化日志系统
func InitLogger(cfg *config.LogConfig) {
	var output io.Writer

	// 根据配置设置输出位置
	if cfg.Path == "console" || cfg.Path == "" {
		output = os.Stdout
	} else {
		// 确保日志目录存在
		logDir := cfg.Path[:strings.LastIndex(cfg.Path, "/")]
		if logDir == "" {
			logDir = "."
		}
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "创建日志目录失败: %v\n", err)
			output = os.Stdout
		} else {
			file, err := os.OpenFile(cfg.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				// 在日志系统初始化失败时，暂时使用标准错误输出
				fmt.Fprintf(os.Stderr, "打开日志文件失败: %v\n", err)
				output = os.Stdout
			} else {
				output = file
			}
		}
	}

	// 获取日志级别
	logLevel := GetLogLevelFromString(cfg.Level)

	// 创建不同级别的日志器
	flags := log.LstdFlags | log.Lshortfile

	defaultLogger = &Logger{
		debugLogger: log.New(io.Discard, "DEBUG: ", flags),
		infoLogger:  log.New(io.Discard, "INFO: ", flags),
		warnLogger:  log.New(io.Discard, "WARN: ", flags),
		errorLogger: log.New(io.Discard, "ERROR: ", flags),
	}

	// 根据级别设置输出
	if logLevel <= DEBUG {
		defaultLogger.debugLogger.SetOutput(output)
	}
	if logLevel <= INFO {
		defaultLogger.infoLogger.SetOutput(output)
	}
	if logLevel <= WARN {
		defaultLogger.warnLogger.SetOutput(output)
	}
	if logLevel <= ERROR {
		defaultLogger.errorLogger.SetOutput(output)
	}
}

// InitLoggerWithMode 根据运行模式初始化日志系统
// isServerMode: true表示HTTP服务器模式，false表示CLI模式
func InitLoggerWithMode(cfg *config.LogConfig, isServerMode bool) {
	var output io.Writer

	// 根据配置设置输出位置
	if cfg.Path == "console" || cfg.Path == "" {
		// 如果没有指定日志路径，使用默认路径
		logPath := config.Config.Directory.Logs + "/costrict-keeper.log"
		output = setupLogFileOutput(logPath)
	} else {
		output = setupLogFileOutput(cfg.Path)
	}

	// 如果是服务器模式，同时输出到控制台
	if isServerMode {
		output = io.MultiWriter(os.Stdout, output)
	}

	// 获取日志级别
	logLevel := GetLogLevelFromString(cfg.Level)

	// 创建不同级别的日志器
	flags := log.LstdFlags | log.Lshortfile

	defaultLogger = &Logger{
		debugLogger: log.New(io.Discard, "DEBUG: ", flags),
		infoLogger:  log.New(io.Discard, "INFO: ", flags),
		warnLogger:  log.New(io.Discard, "WARN: ", flags),
		errorLogger: log.New(io.Discard, "ERROR: ", flags),
	}

	// 根据级别设置输出
	if logLevel <= DEBUG {
		defaultLogger.debugLogger.SetOutput(output)
	}
	if logLevel <= INFO {
		defaultLogger.infoLogger.SetOutput(output)
	}
	if logLevel <= WARN {
		defaultLogger.warnLogger.SetOutput(output)
	}
	if logLevel <= ERROR {
		defaultLogger.errorLogger.SetOutput(output)
	}
}

// setupLogFileOutput 设置日志文件输出
func setupLogFileOutput(logPath string) io.Writer {
	// 确保日志目录存在
	logDir := logPath[:strings.LastIndex(logPath, "/")]
	if logDir == "" {
		logDir = "."
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建日志目录失败: %v\n", err)
		return os.Stdout
	}

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// 在日志系统初始化失败时，暂时使用标准错误输出
		fmt.Fprintf(os.Stderr, "打开日志文件失败: %v\n", err)
		return os.Stdout
	}

	return file
}

// Debug 输出调试日志
func Debug(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.debugLogger.Println(v...)
	}
}

// Debugf 输出格式化调试日志
func Debugf(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.debugLogger.Printf(format, v...)
	}
}

// Info 输出信息日志
func Info(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.infoLogger.Println(v...)
	}
}

// Infof 输出格式化信息日志
func Infof(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.infoLogger.Printf(format, v...)
	}
}

// Warn 输出警告日志
func Warn(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.warnLogger.Println(v...)
	}
}

// Warnf 输出格式化警告日志
func Warnf(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.warnLogger.Printf(format, v...)
	}
}

// Error 输出错误日志
func Error(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.errorLogger.Println(v...)
	}
}

// Errorf 输出格式化错误日志
func Errorf(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.errorLogger.Printf(format, v...)
	}
}

// Fatal 输出致命错误日志并退出程序
func Fatal(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.errorLogger.Fatal(v...)
	} else {
		// 在日志系统未初始化时，使用标准错误输出
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", v...)
		os.Exit(1)
	}
}

// Fatalf 输出格式化致命错误日志并退出程序
func Fatalf(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.errorLogger.Fatalf(format, v...)
	} else {
		// 在日志系统未初始化时，使用标准错误输出
		fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", v...)
		os.Exit(1)
	}
}
