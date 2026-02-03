package web

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/user/netpulse/internal/storage"
	"github.com/user/netpulse/internal/util"
)

// AnalyticsHandlers provides analytics API endpoints.
type AnalyticsHandlers struct {
	db     *storage.DB
	config *util.Config
}

// NewAnalyticsHandlers creates analytics handlers.
func NewAnalyticsHandlers(db *storage.DB, cfg *util.Config) *AnalyticsHandlers {
	return &AnalyticsHandlers{db: db, config: cfg}
}

// TopologyData represents network topology for visualization.
type TopologyData struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

// TopologyNode represents a node in the network graph.
type TopologyNode struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Type     string  `json:"type"` // "source", "hop", "target"
	AvgLatency float64 `json:"avg_latency"`
	HitCount int     `json:"hit_count"`
}

// TopologyEdge represents a connection between nodes.
type TopologyEdge struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Weight int     `json:"weight"` // number of times this path was used
	AvgLatency float64 `json:"avg_latency"`
}

// LatencyPoint represents a latency measurement over time.
type LatencyPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Target    string    `json:"target"`
	HopNum    int       `json:"hop_num"`
	IP        string    `json:"ip"`
	LatencyMs float64   `json:"latency_ms"`
}

// GetTopology returns network topology data for visualization.
func (h *AnalyticsHandlers) GetTopology(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	since := time.Now().Add(-24 * time.Hour)
	
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if d, err := time.ParseDuration(sinceStr); err == nil {
			since = time.Now().Add(-d)
		}
	}
	
	traceStorage := storage.NewTraceStorage(h.db)
	var traces []struct {
		Target string
		Hops   []struct {
			IP string
			LatencyMs float64
			HopNum int
		}
	}
	
	// Get traces based on filter
	if target != "" {
		rawTraces, err := traceStorage.GetHistory(target, since)
		if err != nil {
			writeJSON(w, TopologyData{})
			return
		}
		for _, t := range rawTraces {
			trace := struct {
				Target string
				Hops   []struct {
					IP string
					LatencyMs float64
					HopNum int
				}
			}{Target: t.Target}
			for _, hop := range t.Hops {
				if !hop.Lost {
					trace.Hops = append(trace.Hops, struct {
						IP string
						LatencyMs float64
						HopNum int
					}{IP: hop.IP, LatencyMs: hop.LatencyMs, HopNum: hop.HopNum})
				}
			}
			traces = append(traces, trace)
		}
	} else {
		rawTraces, err := traceStorage.GetAllHistory(since)
		if err != nil {
			writeJSON(w, TopologyData{})
			return
		}
		for _, t := range rawTraces {
			trace := struct {
				Target string
				Hops   []struct {
					IP string
					LatencyMs float64
					HopNum int
				}
			}{Target: t.Target}
			for _, hop := range t.Hops {
				if !hop.Lost {
					trace.Hops = append(trace.Hops, struct {
						IP string
						LatencyMs float64
						HopNum int
					}{IP: hop.IP, LatencyMs: hop.LatencyMs, HopNum: hop.HopNum})
				}
			}
			traces = append(traces, trace)
		}
	}
	
	// Build topology
	topology := h.buildTopology(traces)
	writeJSON(w, topology)
}

func (h *AnalyticsHandlers) buildTopology(traces []struct {
	Target string
	Hops   []struct {
		IP string
		LatencyMs float64
		HopNum int
	}
}) TopologyData {
	nodeMap := make(map[string]*TopologyNode)
	edgeMap := make(map[string]*TopologyEdge)
	
	// Add source node
	nodeMap["source"] = &TopologyNode{
		ID:    "source",
		Label: "Your Network",
		Type:  "source",
	}
	
	for _, trace := range traces {
		prevNode := "source"
		
		for _, hop := range trace.Hops {
			nodeID := hop.IP
			
			// Update or create node
			if node, exists := nodeMap[nodeID]; exists {
				node.HitCount++
				node.AvgLatency = (node.AvgLatency*float64(node.HitCount-1) + hop.LatencyMs) / float64(node.HitCount)
			} else {
				nodeType := "hop"
				if hop.IP == trace.Target {
					nodeType = "target"
				}
				nodeMap[nodeID] = &TopologyNode{
					ID:       nodeID,
					Label:    nodeID,
					Type:     nodeType,
					AvgLatency: hop.LatencyMs,
					HitCount: 1,
				}
			}
			
			// Update or create edge
			edgeID := prevNode + "->" + nodeID
			if edge, exists := edgeMap[edgeID]; exists {
				edge.Weight++
				edge.AvgLatency = (edge.AvgLatency*float64(edge.Weight-1) + hop.LatencyMs) / float64(edge.Weight)
			} else {
				edgeMap[edgeID] = &TopologyEdge{
					Source:     prevNode,
					Target:     nodeID,
					Weight:     1,
					AvgLatency: hop.LatencyMs,
				}
			}
			
			prevNode = nodeID
		}
	}
	
	// Convert to slices
	var nodes []TopologyNode
	for _, node := range nodeMap {
		nodes = append(nodes, *node)
	}
	
	var edges []TopologyEdge
	for _, edge := range edgeMap {
		edges = append(edges, *edge)
	}
	
	// Sort nodes by type (source first, then hops, then targets)
	sort.Slice(nodes, func(i, j int) bool {
		typeOrder := map[string]int{"source": 0, "hop": 1, "target": 2}
		return typeOrder[nodes[i].Type] < typeOrder[nodes[j].Type]
	})
	
	return TopologyData{Nodes: nodes, Edges: edges}
}

// GetLatencyTrends returns latency over time for charting.
func (h *AnalyticsHandlers) GetLatencyTrends(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	since := time.Now().Add(-24 * time.Hour)
	
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if d, err := time.ParseDuration(sinceStr); err == nil {
			since = time.Now().Add(-d)
		}
	}
	
	traceStorage := storage.NewTraceStorage(h.db)
	var points []LatencyPoint
	
	if target != "" {
		traces, err := traceStorage.GetHistory(target, since)
		if err != nil {
			writeJSON(w, points)
			return
		}
		for _, trace := range traces {
			for _, hop := range trace.Hops {
				if !hop.Lost {
					points = append(points, LatencyPoint{
						Timestamp: trace.Timestamp,
						Target:    trace.Target,
						HopNum:    hop.HopNum,
						IP:        hop.IP,
						LatencyMs: hop.LatencyMs,
					})
				}
			}
		}
	} else {
		traces, err := traceStorage.GetAllHistory(since)
		if err != nil {
			writeJSON(w, points)
			return
		}
		for _, trace := range traces {
			// Only include final hop (target) for overview
			for _, hop := range trace.Hops {
				if hop.IP == trace.Target && !hop.Lost {
					points = append(points, LatencyPoint{
						Timestamp: trace.Timestamp,
						Target:    trace.Target,
						HopNum:    hop.HopNum,
						IP:        hop.IP,
						LatencyMs: hop.LatencyMs,
					})
				}
			}
		}
	}
	
	writeJSON(w, points)
}

// RouteChange represents a detected route change.
type RouteChange struct {
	Target      string    `json:"target"`
	DetectedAt  time.Time `json:"detected_at"`
	OldPath     []string  `json:"old_path"`
	NewPath     []string  `json:"new_path"`
	ChangedHops []int     `json:"changed_hops"`
}

// GetAnomalies returns detected anomalies.
func (h *AnalyticsHandlers) GetAnomalies(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)
	
	traceStorage := storage.NewTraceStorage(h.db)
	traces, err := traceStorage.GetAllHistory(since)
	if err != nil {
		writeJSON(w, []RouteChange{})
		return
	}
	
	// Group traces by target
	targetTraces := make(map[string][]struct {
		Timestamp time.Time
		Hops      []string
	})
	
	for _, trace := range traces {
		var hops []string
		for _, hop := range trace.Hops {
			if !hop.Lost {
				hops = append(hops, hop.IP)
			} else {
				hops = append(hops, "*")
			}
		}
		targetTraces[trace.Target] = append(targetTraces[trace.Target], struct {
			Timestamp time.Time
			Hops      []string
		}{Timestamp: trace.Timestamp, Hops: hops})
	}
	
	// Detect route changes
	var changes []RouteChange
	for target, traceList := range targetTraces {
		if len(traceList) < 2 {
			continue
		}
		
		// Sort by timestamp descending
		sort.Slice(traceList, func(i, j int) bool {
			return traceList[i].Timestamp.After(traceList[j].Timestamp)
		})
		
		// Compare adjacent traces
		for i := 0; i < len(traceList)-1; i++ {
			current := traceList[i]
			previous := traceList[i+1]
			
			changedHops := findChangedHops(previous.Hops, current.Hops)
			if len(changedHops) > 0 {
				changes = append(changes, RouteChange{
					Target:      target,
					DetectedAt:  current.Timestamp,
					OldPath:     previous.Hops,
					NewPath:     current.Hops,
					ChangedHops: changedHops,
				})
			}
		}
	}
	
	writeJSON(w, changes)
}

func findChangedHops(old, new []string) []int {
	var changed []int
	maxLen := len(old)
	if len(new) > maxLen {
		maxLen = len(new)
	}
	
	for i := 0; i < maxLen; i++ {
		var oldHop, newHop string
		if i < len(old) {
			oldHop = old[i]
		}
		if i < len(new) {
			newHop = new[i]
		}
		
		// Ignore * (timeout) changes
		if oldHop != newHop && oldHop != "*" && newHop != "*" {
			changed = append(changed, i+1)
		}
	}
	
	return changed
}

// MermaidDiagram returns a Mermaid diagram string for topology.
func (h *AnalyticsHandlers) MermaidDiagram(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	since := time.Now().Add(-24 * time.Hour)
	
	traceStorage := storage.NewTraceStorage(h.db)
	var traces []traceWithMeta
	
	if target != "" {
		rawTraces, _ := traceStorage.GetHistory(target, since)
		for _, t := range rawTraces {
			trace := traceWithMeta{Timestamp: t.Timestamp}
			for _, hop := range t.Hops {
				if !hop.Lost {
					trace.Hops = append(trace.Hops, hopWithLatency{IP: hop.IP, LatencyMs: hop.LatencyMs})
				}
			}
			traces = append(traces, trace)
		}
	} else {
		// Fetch ALL traces when no target specified
		rawTraces, _ := traceStorage.GetAllHistory(since)
		for _, t := range rawTraces {
			trace := traceWithMeta{Timestamp: t.Timestamp}
			for _, hop := range t.Hops {
				if !hop.Lost {
					trace.Hops = append(trace.Hops, hopWithLatency{IP: hop.IP, LatencyMs: hop.LatencyMs})
				}
			}
			traces = append(traces, trace)
		}
	}
	
	// Get public IP history to match with trace times
	ipStorage := storage.NewIPStorage(h.db)
	ipHistory, _ := ipStorage.GetHistory(since)
	
	// Find public IP for a given timestamp
	findPublicIP := func(ts time.Time) string {
		var closest string
		var minDiff time.Duration = 24 * time.Hour
		for _, ip := range ipHistory {
			diff := ts.Sub(ip.Timestamp)
			if diff < 0 {
				diff = -diff
			}
			if diff < minDiff {
				minDiff = diff
				closest = ip.IP
			}
		}
		return closest
	}
	
	// Group traces by public IP
	tracesByIP := make(map[string][]traceWithMeta)
	for _, trace := range traces {
		pip := findPublicIP(trace.Timestamp)
		if pip == "" {
			pip = "unknown"
		}
		tracesByIP[pip] = append(tracesByIP[pip], trace)
	}
	
	// Build Mermaid diagram
	diagram := "graph LR\n"
	
	// Create source nodes for each public IP
	sourceNodes := make(map[string]string)
	idx := 0
	for pip := range tracesByIP {
		nodeID := fmt.Sprintf("Source%d", idx)
		sourceNodes[pip] = nodeID
		if pip == "unknown" {
			diagram += "    " + nodeID + "[\"Your Network\"]\n"
		} else {
			diagram += "    " + nodeID + "[\"📍 " + pip + "\"]\n"
		}
		idx++
	}
	
	seen := make(map[string]bool)
	edges := make(map[string]*edgeInfo)
	
	for pip, pipTraces := range tracesByIP {
		sourceNode := sourceNodes[pip]
		
		for _, trace := range pipTraces {
			prev := sourceNode
			for _, hop := range trace.Hops {
				nodeID := "N" + sanitizeForMermaid(hop.IP)
				if !seen[nodeID] {
					diagram += "    " + nodeID + "[\"" + hop.IP + "\"]\n"
					seen[nodeID] = true
				}
				edgeKey := prev + "->" + nodeID
				// Use the CUMULATIVE latency (actual latency to reach this hop)
				hopLatency := hop.LatencyMs
				if existing, ok := edges[edgeKey]; ok {
					existing.count++
					existing.totalLatency += hopLatency
				} else {
					edges[edgeKey] = &edgeInfo{from: prev, to: nodeID, totalLatency: hopLatency, count: 1}
				}
				prev = nodeID
			}
		}
	}
	
	// Add edges with average cumulative latency
	for _, edge := range edges {
		avgLatency := edge.totalLatency / float64(edge.count)
		label := fmt.Sprintf("%.0fms", avgLatency)
		diagram += "    " + edge.from + " -->|" + label + "| " + edge.to + "\n"
	}
	
	// Style target nodes
	for nodeID := range seen {
		if target != "" && nodeID == "N"+sanitizeForMermaid(target) {
			diagram += "    style " + nodeID + " fill:#00ff41,color:#000\n"
		}
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(diagram))
}

type traceWithMeta struct {
	Timestamp time.Time
	Hops      []hopWithLatency
}

type hopWithLatency struct {
	IP        string
	LatencyMs float64
}

type edgeInfo struct {
	from         string
	to           string
	totalLatency float64
	count        int
}

func sanitizeForMermaid(s string) string {
	result := ""
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result += string(c)
		}
	}
	return result
}

