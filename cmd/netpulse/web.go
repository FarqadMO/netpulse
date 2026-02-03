package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/user/netpulse/internal/storage"
	"github.com/user/netpulse/internal/web"
)

var webPort int

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the web dashboard",
	Long: `Start a lightweight web dashboard for viewing network monitoring data.

The web server provides:
- Live IP history and graphs
- Traceroute visualizations
- Ping sweep and port scan results
- Downloadable reports

Examples:
  netpulse web
  netpulse web --port 8080`,
	RunE: runWeb,
}

func init() {
	webCmd.Flags().IntVarP(&webPort, "port", "p", 8080, "Web server port")
}

func runWeb(cmd *cobra.Command, args []string) error {
	// Initialize database
	db, err := storage.Initialize(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	
	fmt.Printf("Starting web server on http://localhost:%d\n", webPort)
	fmt.Println("Press Ctrl+C to stop")
	
	srv := web.NewServer(db, cfg, webPort)
	return srv.Start()
}
