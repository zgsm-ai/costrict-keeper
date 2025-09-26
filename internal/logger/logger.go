package logger

import (
	"costrict-keeper/internal/env"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	defaultLogger *Logger
)

// sizeLimitedWriter 日志文件大小限制写入器
type sizeLimitedWriter struct {
	filePath string
	maxSize  int64
	file     *os.File
	mu       sync.Mutex
}

// Logger 日志结构体
type Logger struct {
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	logWriter   *sizeLimitedWriter
}

// LogLevel 日志级别类型
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

/**
 * Create a new size limited writer for log file rotation
 * @param {string} filePath - Path to the log file
 * @param {int64} maxSize - Maximum size of log file in bytes before rotation
 * @returns {sizeLimitedWriter} Returns a new sizeLimitedWriter instance
 * @description
 * - Creates a new writer that automatically rotates log files when they reach maxSize
 * - Rotated files will have timestamp suffix (e.g., costrict.log.20240101-150405)
 * - Thread-safe implementation using mutex
 */
func newSizeLimitedWriter(filePath string, maxSize int64) (*sizeLimitedWriter, error) {
	w := &sizeLimitedWriter{
		filePath: filePath,
		maxSize:  maxSize,
	}

	if err := w.rotateIfNeeded(); err != nil {
		return nil, err
	}

	return w, nil
}

/**
 * Write implements io.Writer interface with size checking and rotation
 * @param {[]byte} p - Data to write
 * @returns {int} Returns number of bytes written
 * @returns {error} Returns error if write operation fails
 * @description
 * - Checks file size before writing
 * - Automatically rotates file if size limit exceeded
 * - Thread-safe operation using mutex lock
 */
func (w *sizeLimitedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if rotation is needed before writing
	if err := w.rotateIfNeeded(); err != nil {
		return 0, err
	}

	return w.file.Write(p)
}

/**
 * Close the underlying file
 * @returns {error} Returns error if close operation fails
 */
func (w *sizeLimitedWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

/**
 * Check file size and rotate if necessary
 * @returns {error} Returns error if rotation fails
 * @description
 * - Checks current file size against maxSize limit
 * - If limit exceeded, renames current file with timestamp
 * - Creates new file for continued logging
 */
func (w *sizeLimitedWriter) rotateIfNeeded() error {
	// Check if file exists and get its size
	if w.file != nil {
		fileInfo, err := w.file.Stat()
		if err != nil {
			return err
		}

		if fileInfo.Size() >= w.maxSize {
			// Close current file
			if err := w.file.Close(); err != nil {
				return err
			}

			// Rename current file with timestamp
			timestamp := time.Now().Format("20060102-150405")
			backupPath := w.filePath + "." + timestamp
			if err := os.Rename(w.filePath, backupPath); err != nil {
				return err
			}
		} else {
			// File is within size limit, no rotation needed
			return nil
		}
	}

	// Create/open log file
	file, err := os.OpenFile(w.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	w.file = file
	return nil
}

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

// InitLogger 根据运行模式初始化日志系统
// isServerMode: true表示HTTP服务器模式，false表示CLI模式
func InitLogger(logPath, level string, isServerMode bool, maxSize int64) {
	var output io.Writer

	// 根据配置设置输出位置
	if logPath == "console" || logPath == "" {
		// 如果没有指定日志路径，使用默认路径
		logPath := filepath.Join(env.CostrictDir, "logs", "costrict.log")
		output = setupLogFileOutput(logPath, maxSize)
	} else {
		output = setupLogFileOutput(logPath, maxSize)
	}

	// 如果是服务器模式，同时输出到控制台
	if isServerMode {
		output = io.MultiWriter(os.Stdout, output)
	}

	// 获取日志级别
	logLevel := GetLogLevelFromString(level)

	// 创建不同级别的日志器
	flags := log.LstdFlags

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
func setupLogFileOutput(logPath string, maxSize int64) io.Writer {
	// 确保日志目录存在
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建日志目录失败: %v\n", err)
		return os.Stdout
	}

	writer, err := newSizeLimitedWriter(logPath, maxSize)
	if err != nil {
		// 在日志系统初始化失败时，暂时使用标准错误输出
		fmt.Fprintf(os.Stderr, "创建日志写入器失败: %v\n", err)
		return os.Stdout
	}

	return writer
}

// Debug 输出调试日志
func Debug(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.debugLogger.Println(v...)
	} else {
		log.Println(v...)
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
	} else {
		log.Println(v...)
	}
}

// Infof 输出格式化信息日志
func Infof(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.infoLogger.Printf(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

// Warn 输出警告日志
func Warn(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.warnLogger.Println(v...)
	} else {
		log.Println(v...)
	}
}

// Warnf 输出格式化警告日志
func Warnf(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.warnLogger.Printf(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

// Error 输出错误日志
func Error(v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.errorLogger.Println(v...)
	} else {
		log.Println(v...)
	}
}

// Errorf 输出格式化错误日志
func Errorf(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.errorLogger.Printf(format, v...)
	} else {
		log.Printf(format, v...)
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
