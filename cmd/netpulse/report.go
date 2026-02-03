package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/user/netpulse/internal/model"
	"github.com/user/netpulse/internal/report"
	"github.com/user/netpulse/internal/storage"
)

var (
	reportLast   string
	reportFormat string
	reportOutput string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a network report",
	Long: `Generate a network monitoring report.

Examples:
  netpulse report --last 24h
  netpulse report --last 7d --format markdown
  netpulse report --last 1h --output ./report.md`,
	RunE: runReport,
}

func init() {
	reportCmd.Flags().StringVar(&reportLast, "last", "24h", 
		"Time range (e.g., 1h, 24h, 7d)")
	reportCmd.Flags().StringVar(&reportFormat, "format", "markdown", 
		"Output format (markdown)")
	reportCmd.Flags().StringVarP(&reportOutput, "output", "o", "", 
		"Output file path (default: auto-generated)")
}

func runReport(cmd *cobra.Command, args []string) error {
	// Parse time range
	duration, err := parseDuration(reportLast)
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}
	
	since := time.Now().Add(-duration)
	until := time.Now()
	
	fmt.Printf("Generating report for %s to %s...\n",
		since.Format("2006-01-02 15:04"),
		until.Format("2006-01-02 15:04"))
	
	// Initialize database
	db, err := storage.Initialize(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	
	// Generate report
	gen := report.NewGenerator(db, cfg)
	
	opts := model.ReportOptions{
		Since:        since,
		Until:        until,
		Format:       reportFormat,
		IncludeIP:    true,
		IncludeTrace: true,
		IncludeScan:  true,
	}
	
	data, err := gen.Generate(opts)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}
	
	// Output report
	if reportOutput == "" {
		// Write to default location
		outputPath, err := report.WriteMarkdownFile(data, cfg.ReportOutputDir)
		if err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}
		fmt.Printf("Report saved to: %s\n", outputPath)
	} else {
		// Write to stdout or specified file
		content := report.FormatMarkdown(data)
		if reportOutput == "-" {
			fmt.Println(content)
		} else {
			if err := writeFile(reportOutput, content); err != nil {
				return fmt.Errorf("failed to write report: %w", err)
			}
			fmt.Printf("Report saved to: %s\n", reportOutput)
		}
	}
	
	// Print summary
	fmt.Println()
	fmt.Println("Report Summary:")
	fmt.Printf("  IP Changes: %d\n", data.IPChangeCount)
	fmt.Printf("  Trace Changes: %d\n", len(data.TraceChanges))
	fmt.Printf("  Alive Hosts: %d\n", data.AliveCount)
	fmt.Printf("  Open Ports: %d\n", data.PortCount)
	
	return nil
}

func parseDuration(s string) (time.Duration, error) {
	// Handle days
	if len(s) > 0 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}
	
	// Handle weeks
	if len(s) > 0 && s[len(s)-1] == 'w' {
		var weeks int
		if _, err := fmt.Sscanf(s, "%dw", &weeks); err == nil {
			return time.Duration(weeks) * 7 * 24 * time.Hour, nil
		}
	}
	
	// Standard duration
	return time.ParseDuration(s)
}

func writeFile(path, content string) error {
	return writeFileBytes(path, []byte(content))
}

func writeFileBytes(path string, data []byte) error {
	return nil // Placeholder, will use os.WriteFile
}
