package services

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type LogService struct {
	cloudHost string
}

func NewLogService(cfg *viper.Viper) *LogService {
	return &LogService{
		cloudHost: cfg.GetString("cloud.host"),
	}
}

func (ls *LogService) UploadLog(filePath string, serviceName string) (string, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("日志文件不存在: %s", filePath)
	}

	dest := fmt.Sprintf("cloud://%s/%s/%s-%s.log",
		ls.cloudHost,
		serviceName,
		filepath.Base(filePath),
		time.Now().Format("20060102"))

	// 模拟上传到云端
	fmt.Printf("上传日志: %s -> %s\n", filePath, dest)
	return dest, nil
}
