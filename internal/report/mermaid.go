package report

import (
	"fmt"
	"strings"

	"github.com/user/netpulse/internal/model"
)

// GenerateMermaidDiagram creates a Mermaid flowchart for a traceroute.
func GenerateMermaidDiagram(trace model.TraceResult) string {
	var sb strings.Builder
	
	sb.WriteString("```mermaid\n")
	sb.WriteString("flowchart LR\n")
	sb.WriteString("    style Source fill:#90EE90\n")
	sb.WriteString("    style Target fill:#87CEEB\n")
	sb.WriteString("\n")
	
	// Source node
	sb.WriteString("    Source[Your Network]\n")
	
	prevNode := "Source"
	for i, hop := range trace.Hops {
		nodeID := fmt.Sprintf("H%d", i+1)
		var label string
		
		if hop.Lost {
			label = fmt.Sprintf("Hop %d\\n* * *", i+1)
			sb.WriteString(fmt.Sprintf("    %s[%s]:::lost\n", nodeID, label))
		} else {
			ip := hop.IP
			if hop.Hostname != "" && hop.Hostname != hop.IP {
				ip = fmt.Sprintf("%s\\n%s", shortenHostname(hop.Hostname), hop.IP)
			}
			label = fmt.Sprintf("Hop %d\\n%s\\n%.1fms", i+1, ip, hop.LatencyMs)
			sb.WriteString(fmt.Sprintf("    %s[%s]\n", nodeID, label))
		}
		
		sb.WriteString(fmt.Sprintf("    %s --> %s\n", prevNode, nodeID))
		prevNode = nodeID
	}
	
	// Target node
	sb.WriteString(fmt.Sprintf("    Target[%s]\n", trace.Target))
	sb.WriteString(fmt.Sprintf("    %s --> Target\n", prevNode))
	
	sb.WriteString("\n")
	sb.WriteString("    classDef lost fill:#FFB6C1,stroke:#FF0000\n")
	sb.WriteString("```\n")
	
	return sb.String()
}

// GenerateTraceComparison creates a Mermaid diagram comparing two traces.
func GenerateTraceComparison(oldTrace, newTrace model.TraceResult) string {
	var sb strings.Builder
	
	sb.WriteString("```mermaid\n")
	sb.WriteString("flowchart TB\n")
	sb.WriteString("    subgraph Before\n")
	sb.WriteString("    direction LR\n")
	
	// Old trace
	prevNode := "OldSrc"
	sb.WriteString("    OldSrc((Start))\n")
	for i, hop := range oldTrace.Hops {
		nodeID := fmt.Sprintf("O%d", i+1)
		if hop.Lost {
			sb.WriteString(fmt.Sprintf("    %s[*]\n", nodeID))
		} else {
			sb.WriteString(fmt.Sprintf("    %s[%s]\n", nodeID, shortenIP(hop.IP)))
		}
		sb.WriteString(fmt.Sprintf("    %s --> %s\n", prevNode, nodeID))
		prevNode = nodeID
	}
	sb.WriteString("    end\n\n")
	
	sb.WriteString("    subgraph After\n")
	sb.WriteString("    direction LR\n")
	
	// New trace
	prevNode = "NewSrc"
	sb.WriteString("    NewSrc((Start))\n")
	for i, hop := range newTrace.Hops {
		nodeID := fmt.Sprintf("N%d", i+1)
		
		// Check if this hop is new
		isNew := !containsHop(oldTrace.Hops, hop.IP)
		
		if hop.Lost {
			sb.WriteString(fmt.Sprintf("    %s[*]\n", nodeID))
		} else if isNew {
			sb.WriteString(fmt.Sprintf("    %s[%s]:::new\n", nodeID, shortenIP(hop.IP)))
		} else {
			sb.WriteString(fmt.Sprintf("    %s[%s]\n", nodeID, shortenIP(hop.IP)))
		}
		sb.WriteString(fmt.Sprintf("    %s --> %s\n", prevNode, nodeID))
		prevNode = nodeID
	}
	sb.WriteString("    end\n\n")
	
	sb.WriteString("    classDef new fill:#90EE90,stroke:#228B22\n")
	sb.WriteString("```\n")
	
	return sb.String()
}

// GenerateNetworkTopology creates a Mermaid diagram showing network topology.
func GenerateNetworkTopology(traces []model.TraceResult) string {
	if len(traces) == 0 {
		return ""
	}
	
	var sb strings.Builder
	
	sb.WriteString("```mermaid\n")
	sb.WriteString("flowchart TD\n")
	sb.WriteString("    You[Your Network]:::source\n\n")
	
	// Collect unique hops
	hopSet := make(map[string]bool)
	edges := make(map[string]bool)
	
	for _, trace := range traces {
		prevNode := "You"
		for _, hop := range trace.Hops {
			if hop.Lost || hop.IP == "" {
				continue
			}
			
			nodeID := ipToNodeID(hop.IP)
			hopSet[hop.IP] = true
			
			edge := fmt.Sprintf("%s->%s", prevNode, nodeID)
			edges[edge] = true
			
			prevNode = nodeID
		}
		
		// Connect to target
		targetID := ipToNodeID(trace.Target)
		edge := fmt.Sprintf("%s->%s", prevNode, targetID)
		edges[edge] = true
	}
	
	// Write nodes
	for ip := range hopSet {
		nodeID := ipToNodeID(ip)
		sb.WriteString(fmt.Sprintf("    %s[%s]\n", nodeID, shortenIP(ip)))
	}
	
	// Write target nodes
	for _, trace := range traces {
		targetID := ipToNodeID(trace.Target)
		sb.WriteString(fmt.Sprintf("    %s[%s]:::target\n", targetID, trace.Target))
	}
	
	sb.WriteString("\n")
	
	// Write edges
	for edge := range edges {
		parts := strings.Split(edge, "->")
		sb.WriteString(fmt.Sprintf("    %s --> %s\n", parts[0], parts[1]))
	}
	
	sb.WriteString("\n")
	sb.WriteString("    classDef source fill:#90EE90\n")
	sb.WriteString("    classDef target fill:#87CEEB\n")
	sb.WriteString("```\n")
	
	return sb.String()
}

func shortenHostname(hostname string) string {
	if len(hostname) > 20 {
		parts := strings.Split(hostname, ".")
		if len(parts) > 2 {
			return parts[0] + "..."
		}
		return hostname[:17] + "..."
	}
	return hostname
}

func shortenIP(ip string) string {
	return ip
}

func ipToNodeID(ip string) string {
	// Convert IP to valid Mermaid node ID
	return "N" + strings.ReplaceAll(ip, ".", "_")
}

func containsHop(hops []model.TraceHop, ip string) bool {
	for _, hop := range hops {
		if hop.IP == ip {
			return true
		}
	}
	return false
}
