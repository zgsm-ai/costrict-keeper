package services

import (
	"bufio"
	"bytes"
	"costrict-keeper/internal/config"
	"costrict-keeper/internal/env"
	"costrict-keeper/internal/logger"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type LogService struct {
	logUrl string
}

type UploadLogArgs struct {
	ClientID    string `json:"client_id"`
	UserID      string `json:"user_id"`
	FileName    string `json:"file_name"`
	FirstLineNo int64  `json:"first_line_no"`
	LastLineNo  int64  `json:"end_line_no"`
}

func NewLogService() *LogService {
	return &LogService{
		logUrl: config.Cloud().LogUrl,
	}
}

func uploadBuffer(r io.Reader, filePath string, targetURL string) error {
	au := config.GetAuthConfig()
	args := &UploadLogArgs{
		ClientID: au.MachineID,
		UserID:   au.ID,
		FileName: filepath.Base(filePath),
	}
	data, err := json.Marshal(&args)
	if err != nil {
		return err
	}
	// 创建表单文件
	body := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(body)
	fileWriter, err := multipartWriter.CreateFormFile("logfile", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %v", err)
	}

	// 将文件内容复制到表单文件部分
	if _, err := io.Copy(fileWriter, r); err != nil {
		return fmt.Errorf("failed to copy file to form: %v", err)
	}
	if err := multipartWriter.WriteField("args", string(data)); err != nil {
		return err
	}
	multipartWriter.Close()

	// 创建请求
	request, err := http.NewRequest("POST", targetURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+config.GetAuthConfig().AccessToken)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("failed to upload file: %s", response.Status)
	}
	return nil
}

func uploadFile(filePath string, targetURL string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	return uploadBuffer(file, filePath, targetURL)
}

func getFileErrors(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// 创建一个切片来存储包含 'ERROR' 的行
	var errorLines []string

	// 使用 bufio.Scanner 逐行读取文件
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 检查行是否包含 'ERROR'
		if strings.Contains(line, "ERROR") {
			errorLines = append(errorLines, line)
		}
	}

	// 检查是否在读取文件时发生错误
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	return errorLines, nil
}

func (ls *LogService) UploadErrors() error {
	directory := filepath.Join(env.CostrictDir, "logs")

	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' not exist", directory)
	}

	// 读取目录下的所有文件
	files, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("directory '%s' read failed: %v", directory, err)
	}

	var errorLogs []string

	// 遍历所有文件，上传日志文件
	for _, file := range files {
		if file.IsDir() {
			continue // 跳过子目录
		}
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".log") {
			continue
		}
		filePath := filepath.Join(directory, file.Name())
		lines, err := getFileErrors(filePath)
		if err != nil {
			continue
		}
		errorLogs = append(errorLogs, lines...)
	}
	var logText string
	if len(errorLogs) > 0 {
		logText = strings.Join(errorLogs, "\n")
	} else {
		logText = "no errors"
	}

	lastErrorFile := filepath.Join(env.CostrictDir, "logs", "last-upload-errors")
	lastErrorContent, err := os.ReadFile(lastErrorFile)
	if err == nil && string(lastErrorContent) == logText {
		return nil
	}
	buf := bytes.NewReader([]byte(logText))
	err = uploadBuffer(buf, "last-upload-errors", ls.logUrl)
	if err != nil {
		return err
	}
	err = os.WriteFile(lastErrorFile, []byte(logText), 0664)
	return err
}

/**
 * Upload single log file to cloud storage
 * @param {string} filePath - Path to the log file to upload
 * @param {string} serviceName - Name of the service for organizing logs on server
 * @returns {string} Returns destination path in cloud storage
 * @returns {error} Returns error if upload fails, nil on success
 * @description
 * - Checks if the specified log file exists using os.Stat
 * - Generates cloud destination path with timestamp
 * - Simulates upload operation (currently just prints to console)
 * - Format: {logurl}/{serviceName}/{filename}-{timestamp}.log
 * @throws
 * - File not found errors (os.Stat)
 * - File path generation errors
 */
func (ls *LogService) UploadFile(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Warnf("Failed to upload log file '%s'", filePath)
		return fmt.Errorf("log file is not exist: %s", filePath)
	}
	if err := uploadFile(filePath, ls.logUrl); err != nil {
		logger.Warnf("Failed to upload log file '%s', error: %v", filePath, err.Error())
		return err
	}
	logger.Infof("Upload log file '%s' to '%s'", filePath, ls.logUrl)
	return nil
}

/**
* Upload log files from specified directory to server
* @param {string} directory - Path to the directory containing log files to upload
* @param {string} serviceName - Name of the service for organizing logs on server
* @returns {string} Destination path for the uploaded directory
* @returns {error} Error if any operation fails
* @description
* - Validates that the specified directory exists
* - Reads all files from the specified directory
* - Filters for .log files only
* - Uploads each file using UploadFile method
* - Returns destination path for the uploaded directory
* @throws
* - Directory access errors (os.ReadDir)
* - File upload errors (UploadFile)
 */
func (ls *LogService) UploadDirectory(directory string) error {
	// 检查目录是否存在
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return fmt.Errorf("指定的目录不存在: %s", directory)
	}

	// 读取目录下的所有文件
	files, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("读取目录失败: %v", err)
	}

	var uploadedFiles []string
	var uploadErrors []string

	// 遍历所有文件，上传日志文件
	for _, file := range files {
		if file.IsDir() {
			continue // 跳过子目录
		}

		// 只处理.log文件
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".log") {
			continue
		}

		filePath := filepath.Join(directory, file.Name())
		err := ls.UploadFile(filePath)
		if err != nil {
			uploadErrors = append(uploadErrors, filePath)
			continue
		}

		uploadedFiles = append(uploadedFiles, filePath)
	}

	// 如果有上传错误，返回错误信息
	if len(uploadErrors) > 0 {
		return fmt.Errorf("部分文件上传失败: %s", strings.Join(uploadErrors, "; "))
	}

	// 如果没有日志文件，返回提示信息
	if len(uploadedFiles) == 0 {
		return fmt.Errorf("指定的目录中没有找到日志文件: %s", directory)
	}

	return nil
}
