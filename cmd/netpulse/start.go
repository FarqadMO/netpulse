package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/user/netpulse/internal/daemon"
	"github.com/user/netpulse/internal/storage"
	"github.com/user/netpulse/internal/util"
	"github.com/user/netpulse/internal/web"
)

var (
	foreground   bool
	withWeb      bool
	startWebPort int
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the netpulse daemon",
	Long:  "Start the netpulse daemon in the background to monitor network status.",
	RunE:  runStart,
}

func init() {
	startCmd.Flags().BoolVarP(&foreground, "foreground", "f", false, 
		"Run in foreground instead of daemonizing")
	startCmd.Flags().BoolVar(&withWeb, "with-web", false,
		"Also start the web dashboard server")
	startCmd.Flags().IntVar(&startWebPort, "web-port", 8080,
		"Port for web server (when using --with-web)")
}

func runStart(cmd *cobra.Command, args []string) error {
	// Check if already running
	running, pid := daemon.CheckRunning(cfg.DataDir)
	if running {
		fmt.Printf("Daemon is already running (PID %d)\n", pid)
		return nil
	}
	
	if foreground {
		return runForeground()
	}
	
	return runDaemon()
}

func runForeground() error {
	fmt.Println("Starting netpulse in foreground mode...")
	
	d, err := daemon.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}
	
	if err := d.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}
	
	// Start web server if requested
	if withWeb {
		go func() {
			db, err := storage.Initialize(cfg.DataDir)
			if err != nil {
				util.Error("Failed to open database for web server: %v", err)
				return
			}
			
			srv := web.NewServer(db, cfg, startWebPort)
			fmt.Printf("Web dashboard: http://localhost:%d\n", startWebPort)
			if err := srv.Start(); err != nil {
				util.Error("Web server error: %v", err)
			}
		}()
	}
	
	fmt.Println("NetPulse daemon started. Press Ctrl+C to stop.")
	
	// Wait for daemon to finish
	d.Wait()
	
	return nil
}

func runDaemon() error {
	// Re-execute self in background
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	
	// Prepare arguments
	args := []string{"start", "--foreground"}
	if cfgFile != "" {
		args = append(args, "--config", cfgFile)
	}
	if withWeb {
		args = append(args, "--with-web", "--web-port", fmt.Sprintf("%d", startWebPort))
	}
	
	// Create log file for daemon output
	logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	
	// Start background process
	procAttr := &os.ProcAttr{
		Dir:   "/",
		Env:   os.Environ(),
		Files: []*os.File{nil, logFile, logFile},
		Sys: &syscall.SysProcAttr{
			Setsid: true,
		},
	}
	
	proc, err := os.StartProcess(executable, append([]string{executable}, args...), procAttr)
	if err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start daemon process: %w", err)
	}
	
	// Detach from parent
	if err := proc.Release(); err != nil {
		util.Warn("Failed to release process: %v", err)
	}
	
	fmt.Printf("NetPulse daemon started (PID %d)\n", proc.Pid)
	fmt.Printf("Logs: %s\n", cfg.LogFile)
	if withWeb {
		fmt.Printf("Web dashboard: http://localhost:%d\n", startWebPort)
	}
	
	return nil
}

// For cross-platform compatibility, we also support a simpler approach
func runDaemonSimple() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	
	args := []string{"start", "--foreground"}
	cmd := exec.Command(executable, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}
	
	fmt.Printf("NetPulse daemon started (PID %d)\n", cmd.Process.Pid)
	return nil
}

