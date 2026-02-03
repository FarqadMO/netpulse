// Package report generates network monitoring reports.
package report

import (
	"fmt"
	"time"

	"github.com/user/netpulse/internal/model"
	"github.com/user/netpulse/internal/storage"
	"github.com/user/netpulse/internal/util"
)

// Generator creates network monitoring reports.
type Generator struct {
	db     *storage.DB
	config *util.Config
}

// NewGenerator creates a new report generator.
func NewGenerator(db *storage.DB, cfg *util.Config) *Generator {
	return &Generator{
		db:     db,
		config: cfg,
	}
}

// ReportData holds all data for a report.
type ReportData struct {
	GeneratedAt time.Time
	Since       time.Time
	Until       time.Time
	
	// IP Section
	IPRecords     []model.IPRecord
	IPChangeCount int
	CurrentIP     *model.IPRecord
	
	// Trace Section
	Traces         []model.TraceResult
	TracesByTarget map[string][]model.TraceResult
	
	// Scan Section
	AliveHosts    []model.ScanHost
	AliveCount    int
	OpenPorts     []model.ScanPort
	PortCount     int
	
	// Anomalies (simplified)
	IPChanges      []IPChange
	TraceChanges   []TraceChange
	PortChanges    []PortChange
}

// IPChange represents an IP address change.
type IPChange struct {
	OldIP     string
	NewIP     string
	Timestamp time.Time
}

// TraceChange represents a traceroute path change.
type TraceChange struct {
	Target    string
	OldHops   []string
	NewHops   []string
	Added     []string
	Removed   []string
	Timestamp time.Time
}

// PortChange represents a port state change.
type PortChange struct {
	Host      string
	Port      int
	OldState  string
	NewState  string
	Timestamp time.Time
}

// Generate creates a report for the specified time range.
func (g *Generator) Generate(opts model.ReportOptions) (*ReportData, error) {
	data := &ReportData{
		GeneratedAt:    time.Now(),
		Since:          opts.Since,
		Until:          opts.Until,
		TracesByTarget: make(map[string][]model.TraceResult),
	}
	
	// Get IP history
	ipStorage := storage.NewIPStorage(g.db)
	
	records, err := ipStorage.GetHistory(opts.Since)
	if err != nil {
		return nil, fmt.Errorf("failed to get IP history: %w", err)
	}
	data.IPRecords = records
	
	// Get current IP
	current, err := ipStorage.GetLatest()
	if err == nil {
		data.CurrentIP = current
	}
	
	// Calculate IP changes
	data.IPChanges = g.detectIPChanges(records)
	data.IPChangeCount = len(data.IPChanges)
	
	// Get traces
	traceStorage := storage.NewTraceStorage(g.db)
	traces, err := traceStorage.GetAllHistory(opts.Since)
	if err != nil {
		return nil, fmt.Errorf("failed to get traces: %w", err)
	}
	data.Traces = traces
	
	// Group traces by target
	for _, trace := range traces {
		data.TracesByTarget[trace.Target] = append(data.TracesByTarget[trace.Target], trace)
	}
	
	// Detect trace changes
	for target, traces := range data.TracesByTarget {
		changes := g.detectTraceChanges(target, traces)
		data.TraceChanges = append(data.TraceChanges, changes...)
	}
	
	// Get scan results
	scanStorage := storage.NewScanStorage(g.db)
	
	hosts, err := scanStorage.GetAliveHosts()
	if err == nil {
		data.AliveHosts = hosts
		data.AliveCount = len(hosts)
	}
	
	// Get open ports for all hosts
	for _, host := range hosts {
		ports, err := scanStorage.GetHostPorts(host.ID)
		if err == nil {
			for i := range ports {
				// Attach host IP to port for reporting
				data.OpenPorts = append(data.OpenPorts, ports[i])
			}
		}
	}
	data.PortCount = len(data.OpenPorts)
	
	return data, nil
}

func (g *Generator) detectIPChanges(records []model.IPRecord) []IPChange {
	var changes []IPChange
	
	for i := 0; i < len(records)-1; i++ {
		if records[i].IP != records[i+1].IP {
			changes = append(changes, IPChange{
				OldIP:     records[i+1].IP,
				NewIP:     records[i].IP,
				Timestamp: records[i].Timestamp,
			})
		}
	}
	
	return changes
}

func (g *Generator) detectTraceChanges(target string, traces []model.TraceResult) []TraceChange {
	var changes []TraceChange
	
	for i := 0; i < len(traces)-1; i++ {
		curr := traces[i]
		prev := traces[i+1]
		
		currHops := getHopIPs(curr.Hops)
		prevHops := getHopIPs(prev.Hops)
		
		if !equalHops(currHops, prevHops) {
			added, removed := diffHops(prevHops, currHops)
			changes = append(changes, TraceChange{
				Target:    target,
				OldHops:   prevHops,
				NewHops:   currHops,
				Added:     added,
				Removed:   removed,
				Timestamp: curr.Timestamp,
			})
		}
	}
	
	return changes
}

func getHopIPs(hops []model.TraceHop) []string {
	ips := make([]string, 0, len(hops))
	for _, hop := range hops {
		if !hop.Lost && hop.IP != "" {
			ips = append(ips, hop.IP)
		}
	}
	return ips
}

func equalHops(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func diffHops(old, new []string) (added, removed []string) {
	oldSet := make(map[string]bool)
	newSet := make(map[string]bool)
	
	for _, h := range old {
		oldSet[h] = true
	}
	for _, h := range new {
		newSet[h] = true
	}
	
	for h := range newSet {
		if !oldSet[h] {
			added = append(added, h)
		}
	}
	for h := range oldSet {
		if !newSet[h] {
			removed = append(removed, h)
		}
	}
	
	return
}
