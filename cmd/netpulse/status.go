package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/user/netpulse/internal/daemon"
	"github.com/user/netpulse/internal/storage"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long:  "Show the current status of the netpulse daemon and latest results.",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		MarginBottom(1)
	
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86"))
	
	runningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")).
		Bold(true)
	
	stoppedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	
	// Check daemon status
	running, pid := daemon.CheckRunning(cfg.DataDir)
	
	fmt.Println(titleStyle.Render("NetPulse Status"))
	fmt.Println()
	
	// Daemon status
	fmt.Print(labelStyle.Render("Daemon: "))
	if running {
		fmt.Println(runningStyle.Render(fmt.Sprintf("Running (PID %d)", pid)))
	} else {
		fmt.Println(stoppedStyle.Render("Stopped"))
	}
	
	// Try to read status file for more details
	if sf, err := daemon.ReadStatusFile(cfg.DataDir); err == nil {
		fmt.Print(labelStyle.Render("Started: "))
		fmt.Println(valueStyle.Render(sf.StartTime))
		
		fmt.Print(labelStyle.Render("Uptime: "))
		fmt.Println(valueStyle.Render(sf.Uptime))
		
		if sf.CurrentIP != "" {
			fmt.Print(labelStyle.Render("Current IP: "))
			fmt.Println(valueStyle.Render(sf.CurrentIP))
		}
		
		if len(sf.Jobs) > 0 {
			fmt.Println()
			fmt.Println(titleStyle.Render("Jobs"))
			
			for _, job := range sf.Jobs {
				statusStr := "idle"
				if job.Running {
					statusStr = "running"
				}
				fmt.Printf("  %s: %s (last: %s, errors: %d)\n",
					labelStyle.Render(job.Name),
					valueStyle.Render(statusStr),
					job.LastRun.Format("15:04:05"),
					job.ErrorCount)
			}
		}
	}
	
	// Get database stats
	db, err := storage.Initialize(cfg.DataDir)
	if err == nil {
		fmt.Println()
		fmt.Println(titleStyle.Render("Database Stats"))
		
		ipStorage := storage.NewIPStorage(db)
		if count, err := ipStorage.Count(); err == nil {
			fmt.Printf("  %s %s\n",
				labelStyle.Render("IP records:"),
				valueStyle.Render(fmt.Sprintf("%d", count)))
		}
		
		scanStorage := storage.NewScanStorage(db)
		if count, err := scanStorage.CountAliveHosts(); err == nil {
			fmt.Printf("  %s %s\n",
				labelStyle.Render("Alive hosts:"),
				valueStyle.Render(fmt.Sprintf("%d", count)))
		}
		if count, err := scanStorage.CountOpenPorts(); err == nil {
			fmt.Printf("  %s %s\n",
				labelStyle.Render("Open ports:"),
				valueStyle.Render(fmt.Sprintf("%d", count)))
		}
		
		// Show latest IP
		if latest, err := ipStorage.GetLatest(); err == nil && latest != nil {
			fmt.Println()
			fmt.Println(titleStyle.Render("Latest IP"))
			fmt.Printf("  %s %s\n",
				labelStyle.Render("IP:"),
				valueStyle.Render(latest.IP))
			fmt.Printf("  %s %s\n",
				labelStyle.Render("ASN:"),
				valueStyle.Render(latest.ASN))
			fmt.Printf("  %s %s\n",
				labelStyle.Render("ISP:"),
				valueStyle.Render(latest.ISP))
			fmt.Printf("  %s %s\n",
				labelStyle.Render("Last check:"),
				valueStyle.Render(latest.Timestamp.Format("2006-01-02 15:04:05")))
		}
	}
	
	return nil
}
