package utils

import (
	"fmt"
	"net"
	"time"
)

func CheckPortAvailable(port int) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", fmt.Sprintf("%d", port)), timeout)
	if err != nil {
		// 连接失败，说明端口可用
		return true
	}
	if conn != nil {
		conn.Close()
		// 连接成功，说明端口已被占用
		return false
	}
	return true
}
