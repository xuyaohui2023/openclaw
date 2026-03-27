package handler

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// PortStatus represents the running state of a port.
type PortStatus string

const (
	PortStatusRunning  PortStatus = "running"  // 启动：端口已监听，连接成功
	PortStatusStarting PortStatus = "starting" // 启动中：连接超时，进程可能正在初始化
	PortStatusStopped  PortStatus = "stopped"  // 未启动：连接被拒绝，端口未监听
)

type portCheckResponse struct {
	Port   int        `json:"port"`
	Status PortStatus `json:"status"`
	Msg    string     `json:"msg"`
}

// probePort returns the PortStatus for the given TCP address.
// Detection logic:
//   - connection succeeds          → running
//   - connection refused (ECONNREFUSED) → stopped
//   - timeout or other transient error  → starting
func probePort(addr string) PortStatus {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err == nil {
		conn.Close()
		return PortStatusRunning
	}

	// Walk the error chain to find the underlying OS error.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		// Unwrap to syscall error if possible.
		var syscallErr syscall.Errno
		if errors.As(opErr.Err, &syscallErr) {
			if syscallErr == syscall.ECONNREFUSED {
				return PortStatusStopped
			}
		}
		// Timeout means something is bound but not yet accepting.
		if opErr.Timeout() {
			return PortStatusStarting
		}
	}

	// Fallback: treat unknown errors as starting (transient).
	return PortStatusStarting
}

// PortCheckHandler checks the running state of port 18789 on the local machine.
//
//	GET /api/v1/port-check
//
// Response status field:
//
//	"running"  — port is listening and accepting connections (启动)
//	"starting" — connection timed out; process is likely initialising (启动中)
//	"stopped"  — connection refused; nothing is listening on the port (未启动)
func PortCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		const port = 18789
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		status := probePort(addr)

		var msg string
		switch status {
		case PortStatusRunning:
			msg = "port is active and accepting connections"
		case PortStatusStarting:
			msg = "port is not yet ready, process may be starting"
		case PortStatusStopped:
			msg = "port is not listening"
		}

		writeJSON(w, http.StatusOK, portCheckResponse{
			Port:   port,
			Status: status,
			Msg:    msg,
		})
	}
}
