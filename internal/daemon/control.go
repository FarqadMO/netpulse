package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// CheckRunning checks if the daemon is already running.
func CheckRunning(dataDir string) (bool, int) {
	pidFile := filepath.Join(dataDir, "netpulse.pid")
	
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false, 0
	}
	
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return false, 0
	}
	
	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0
	}
	
	// Send signal 0 to check if process is running
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false, 0
	}
	
	return true, pid
}

// SendStop sends a stop signal to the running daemon.
func SendStop(dataDir string) error {
	running, pid := CheckRunning(dataDir)
	if !running {
		return fmt.Errorf("daemon is not running")
	}
	
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}
	
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send signal: %w", err)
	}
	
	return nil
}

// StatusFile holds serialized daemon status.
type StatusFile struct {
	Running   bool   `json:"running"`
	PID       int    `json:"pid"`
	StartTime string `json:"start_time"`
	Uptime    string `json:"uptime"`
	CurrentIP string `json:"current_ip,omitempty"`
	Jobs      []JobStatus `json:"jobs"`
}

// WriteStatusFile writes the daemon status to a file.
func WriteStatusFile(dataDir string, status *DaemonStatus, currentIP string) error {
	statusFile := filepath.Join(dataDir, "status.json")
	
	sf := StatusFile{
		Running:   status.Running,
		PID:       status.PID,
		StartTime: status.StartTime.Format("2006-01-02 15:04:05"),
		Uptime:    status.Uptime.String(),
		CurrentIP: currentIP,
		Jobs:      status.Jobs,
	}
	
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(statusFile, data, 0644)
}

// ReadStatusFile reads the daemon status from a file.
func ReadStatusFile(dataDir string) (*StatusFile, error) {
	statusFile := filepath.Join(dataDir, "status.json")
	
	data, err := os.ReadFile(statusFile)
	if err != nil {
		return nil, err
	}
	
	var sf StatusFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}
	
	return &sf, nil
}
