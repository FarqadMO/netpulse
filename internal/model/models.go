// Package model defines core data structures for netpulse.
package model

import "time"

// IPRecord represents a public IP address record with metadata.
type IPRecord struct {
	ID        int64     `json:"id"`
	IP        string    `json:"ip"`
	ASN       string    `json:"asn"`
	ISP       string    `json:"isp"`
	Country   string    `json:"country"`
	City      string    `json:"city"`
	Timestamp time.Time `json:"timestamp"`
}

// TraceResult represents a complete traceroute result.
type TraceResult struct {
	ID        int64      `json:"id"`
	Target    string     `json:"target"`
	Timestamp time.Time  `json:"timestamp"`
	Hops      []TraceHop `json:"hops"`
}

// TraceHop represents a single hop in a traceroute.
type TraceHop struct {
	ID        int64   `json:"id"`
	TraceID   int64   `json:"trace_id"`
	HopNum    int     `json:"hop_num"`
	IP        string  `json:"ip"`
	Hostname  string  `json:"hostname"`
	LatencyMs float64 `json:"latency_ms"`
	Lost      bool    `json:"lost"`
}

// ScanHost represents a discovered host from ping sweep.
type ScanHost struct {
	ID       int64     `json:"id"`
	IP       string    `json:"ip"`
	Hostname string    `json:"hostname"`
	Alive    bool      `json:"alive"`
	LatencyMs float64  `json:"latency_ms"`
	LastSeen time.Time `json:"last_seen"`
}

// ScanPort represents an open port on a host.
type ScanPort struct {
	ID       int64     `json:"id"`
	HostID   int64     `json:"host_id"`
	Port     int       `json:"port"`
	Protocol string    `json:"protocol"`
	Service  string    `json:"service"`
	State    string    `json:"state"`
	Banner   string    `json:"banner"`
	LastSeen time.Time `json:"last_seen"`
}

// DaemonStatus represents the current state of the daemon.
type DaemonStatus struct {
	Running     bool      `json:"running"`
	PID         int       `json:"pid"`
	StartTime   time.Time `json:"start_time"`
	Uptime      string    `json:"uptime"`
	CurrentIP   string    `json:"current_ip"`
	LastCheck   time.Time `json:"last_check"`
	JobsRunning int       `json:"jobs_running"`
}

// ProbeStatus represents the status of a probe job.
type ProbeStatus struct {
	Name       string    `json:"name"`
	LastRun    time.Time `json:"last_run"`
	NextRun    time.Time `json:"next_run"`
	LastResult string    `json:"last_result"`
	ErrorCount int       `json:"error_count"`
}

// Anomaly represents a detected network anomaly.
type Anomaly struct {
	ID          int64     `json:"id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
	Data        string    `json:"data"`
}

// ReportOptions defines options for report generation.
type ReportOptions struct {
	Since      time.Time `json:"since"`
	Until      time.Time `json:"until"`
	Format     string    `json:"format"`
	OutputPath string    `json:"output_path"`
	IncludeIP  bool      `json:"include_ip"`
	IncludeTrace bool    `json:"include_trace"`
	IncludeScan bool     `json:"include_scan"`
}
