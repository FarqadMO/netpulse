package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/user/netpulse/internal/storage"
	"github.com/user/netpulse/internal/tui"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the terminal dashboard",
	Long: `Launch an interactive terminal dashboard showing live network status.

The dashboard shows:
- Current IP status
- Traceroute status
- Scan status
- Latest anomalies

Use arrow keys to navigate, 'q' to quit.`,
	RunE: runUI,
}

func runUI(cmd *cobra.Command, args []string) error {
	// Initialize database
	db, err := storage.Initialize(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	
	app := tui.NewApp(db, cfg)
	return app.Run()
}
