//go:build !windows

package utils

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"
)

// checkPortConnectable checks if a port is connectable on localhost (POSIX implementation)
func checkPortConnectable(port int) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", fmt.Sprintf("%d", port)), timeout)
	if err != nil {
		return false
	}
	if conn != nil {
		conn.Close()
		return true
	}
	return false
}

// checkPortListenable checks if a port is listenable (POSIX implementation)
func checkPortListenable(port int) bool {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}

	// Create ListenConfig with control function to disable SO_REUSEADDR
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				// Disable SO_REUSEADDR to prevent address reuse
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 0)
			})
		},
	}

	l, err := lc.Listen(context.Background(), "tcp", addr.String())
	if err != nil {
		return false
	}
	defer l.Close()
	return true
}
