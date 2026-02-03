// Package util provides common utilities for netpulse.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	DataDir         string        `mapstructure:"data_dir"`
	LogLevel        string        `mapstructure:"log_level"`
	LogFile         string        `mapstructure:"log_file"`
	
	// Probe intervals
	IPCheckInterval    time.Duration `mapstructure:"ip_check_interval"`
	TraceInterval      time.Duration `mapstructure:"trace_interval"`
	PingSweepInterval  time.Duration `mapstructure:"ping_sweep_interval"`
	PortScanInterval   time.Duration `mapstructure:"port_scan_interval"`
	
	// Traceroute targets
	TraceTargets []string `mapstructure:"trace_targets"`
	
	// Ping sweep settings
	SweepSubnet     string `mapstructure:"sweep_subnet"`
	SweepConcurrency int   `mapstructure:"sweep_concurrency"`
	SweepTimeout    time.Duration `mapstructure:"sweep_timeout"`
	
	// Port scan settings
	ScanPorts       []int  `mapstructure:"scan_ports"`
	ScanConcurrency int    `mapstructure:"scan_concurrency"`
	ScanTimeout     time.Duration `mapstructure:"scan_timeout"`
	
	// Report settings
	ReportOutputDir string `mapstructure:"report_output_dir"`
	
	// Web server
	WebPort int `mapstructure:"web_port"`
	
	// Adaptive intervals
	StableIntervalMultiplier float64 `mapstructure:"stable_interval_multiplier"`
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".netpulse")
	
	return &Config{
		DataDir:         dataDir,
		LogLevel:        "info",
		LogFile:         filepath.Join(dataDir, "netpulse.log"),
		
		IPCheckInterval:   5 * time.Minute,
		TraceInterval:     15 * time.Minute,
		PingSweepInterval: 30 * time.Minute,
		PortScanInterval:  1 * time.Hour,
		
		TraceTargets: []string{
			"8.8.8.8",      // Google DNS
			"1.1.1.1",      // Cloudflare DNS
			"185.97.0.1",   // Middle East target
		},
		
		SweepSubnet:      "192.168.1.0/24",
		SweepConcurrency: 50,
		SweepTimeout:     2 * time.Second,
		
		ScanPorts:        GetTopPorts(50),
		ScanConcurrency:  20,
		ScanTimeout:      3 * time.Second,
		
		ReportOutputDir: filepath.Join(dataDir, "reports"),
		WebPort:         8080,
		
		StableIntervalMultiplier: 2.0,
	}
}

// LoadConfig loads configuration from file and environment.
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()
	
	// Ensure config directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}
	
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(cfg.DataDir)
	viper.AddConfigPath(".")
	
	// Set defaults in viper
	viper.SetDefault("data_dir", cfg.DataDir)
	viper.SetDefault("log_level", cfg.LogLevel)
	viper.SetDefault("ip_check_interval", cfg.IPCheckInterval)
	viper.SetDefault("trace_interval", cfg.TraceInterval)
	viper.SetDefault("trace_targets", cfg.TraceTargets)
	viper.SetDefault("sweep_subnet", cfg.SweepSubnet)
	viper.SetDefault("sweep_concurrency", cfg.SweepConcurrency)
	viper.SetDefault("scan_ports", cfg.ScanPorts)
	viper.SetDefault("scan_concurrency", cfg.ScanConcurrency)
	viper.SetDefault("web_port", cfg.WebPort)
	
	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}
	
	// Unmarshal into config struct
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return cfg, nil
}

// GetTopPorts returns the top N most common ports.
func GetTopPorts(n int) []int {
	topPorts := []int{
		21, 22, 23, 25, 53, 80, 110, 111, 135, 139,
		143, 443, 445, 993, 995, 1723, 3306, 3389, 5432, 5900,
		8080, 8443, 8888, 27017, 6379, 11211, 1433, 1521, 5984, 9200,
		2181, 9092, 6443, 10250, 2379, 4443, 7443, 8000, 8001, 8002,
		9000, 9001, 9090, 9091, 9443, 10000, 10443, 15672, 27018, 27019,
	}
	
	if n > len(topPorts) {
		n = len(topPorts)
	}
	return topPorts[:n]
}

// EnsureDir ensures a directory exists.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
