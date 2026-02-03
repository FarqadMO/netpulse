package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/user/netpulse/internal/daemon"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the netpulse daemon",
	Long:  "Stop the running netpulse daemon gracefully.",
	RunE:  runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	running, pid := daemon.CheckRunning(cfg.DataDir)
	if !running {
		fmt.Println("Daemon is not running")
		return nil
	}
	
	fmt.Printf("Stopping daemon (PID %d)...\n", pid)
	
	if err := daemon.SendStop(cfg.DataDir); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}
	
	// Wait for daemon to stop
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		running, _ := daemon.CheckRunning(cfg.DataDir)
		if !running {
			fmt.Println("Daemon stopped")
			return nil
		}
	}
	
	fmt.Println("Warning: Daemon may not have stopped completely")
	return nil
}
