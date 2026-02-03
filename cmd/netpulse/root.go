package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/user/netpulse/internal/util"
)

var (
	cfgFile string
	cfg     *util.Config
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "netpulse",
	Short: "Network monitoring and analysis tool",
	Long: `NetPulse is a comprehensive network monitoring tool that tracks:
- Public IP changes and ASN/ISP information
- Traceroute paths to multiple targets
- Local network hosts via ping sweep
- Open ports on discovered hosts

It runs as a background daemon and provides reports with Mermaid diagrams.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", 
		"config file (default is $HOME/.netpulse/config.yaml)")
	rootCmd.PersistentFlags().String("log-level", "info", 
		"log level (debug, info, warn, error)")
	
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	
	// Add subcommands
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(webCmd)
	rootCmd.AddCommand(uiCmd)
	rootCmd.AddCommand(versionCmd)
	
	// Add shell completion
	rootCmd.AddCommand(completionCmd)
}

func initConfig() {
	var err error
	cfg, err = util.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	
	// Initialize logger
	util.InitLogger(cfg.LogLevel, cfg.LogFile)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("netpulse version 1.0.0")
	},
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for netpulse.

To load completions:

Bash:
  $ source <(netpulse completion bash)

Zsh:
  $ source <(netpulse completion zsh)

Fish:
  $ netpulse completion fish | source

PowerShell:
  PS> netpulse completion powershell | Out-String | Invoke-Expression
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}
