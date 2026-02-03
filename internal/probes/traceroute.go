package probes

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/user/netpulse/internal/model"
)

// TracerouteProbe handles traceroute operations.
type TracerouteProbe struct {
	maxHops int
	timeout time.Duration
}

// NewTracerouteProbe creates a new traceroute probe.
func NewTracerouteProbe() *TracerouteProbe {
	return &TracerouteProbe{
		maxHops: 30,
		timeout: 3 * time.Second,
	}
}

// Trace performs a traceroute to the target using system command.
func (p *TracerouteProbe) Trace(ctx context.Context, target string) (*model.TraceResult, error) {
	result := &model.TraceResult{
		Target:    target,
		Timestamp: time.Now(),
		Hops:      make([]model.TraceHop, 0, p.maxHops),
	}

	// Use system traceroute command
	hops, err := p.runSystemTraceroute(ctx, target)
	if err != nil {
		return result, err
	}
	result.Hops = hops

	return result, nil
}

func (p *TracerouteProbe) runSystemTraceroute(ctx context.Context, target string) ([]model.TraceHop, error) {
	// Build command based on OS
	var cmd *exec.Cmd
	
	// Use traceroute on Unix (macOS/Linux)
	// -n = numeric output (no DNS), -q 1 = 1 probe per hop, -w = wait time
	cmd = exec.CommandContext(ctx, "traceroute", "-n", "-q", "1", "-w", "2", "-m", 
		strconv.Itoa(p.maxHops), target)

	output, err := cmd.Output()
	if err != nil {
		// Try fallback to ICMP traceroute
		cmd = exec.CommandContext(ctx, "traceroute", "-n", "-I", "-q", "1", "-w", "2", "-m",
			strconv.Itoa(p.maxHops), target)
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("traceroute failed: %w", err)
		}
	}

	return p.parseTracerouteOutput(string(output))
}

// parseTracerouteOutput parses the output of the traceroute command.
func (p *TracerouteProbe) parseTracerouteOutput(output string) ([]model.TraceHop, error) {
	var hops []model.TraceHop

	// Regex to match hop lines
	// Format: " 1  192.168.0.1  1.234 ms" or " 1  * * *"
	hopRegex := regexp.MustCompile(`^\s*(\d+)\s+(?:(\d+\.\d+\.\d+\.\d+)\s+(\d+\.?\d*)\s*ms|\*\s+\*\s+\*)`)
	
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		
		matches := hopRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			hopNum, _ := strconv.Atoi(matches[1])
			hop := model.TraceHop{
				HopNum: hopNum,
				Lost:   true,
			}

			if len(matches) >= 4 && matches[2] != "" {
				hop.IP = matches[2]
				hop.Lost = false
				if matches[3] != "" {
					hop.LatencyMs, _ = strconv.ParseFloat(matches[3], 64)
				}
			}

			hops = append(hops, hop)
		}
	}

	return hops, nil
}

// TraceMultiple traces multiple targets concurrently.
func (p *TracerouteProbe) TraceMultiple(ctx context.Context, targets []string) ([]*model.TraceResult, error) {
	results := make([]*model.TraceResult, len(targets))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, target := range targets {
		wg.Add(1)
		go func(idx int, t string) {
			defer wg.Done()

			result, err := p.Trace(ctx, t)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			results[idx] = result
		}(i, target)
	}

	wg.Wait()
	return results, firstErr
}

// SetMaxHops sets the maximum number of hops.
func (p *TracerouteProbe) SetMaxHops(max int) {
	if max > 0 && max <= 64 {
		p.maxHops = max
	}
}

// SetTimeout sets the per-hop timeout.
func (p *TracerouteProbe) SetTimeout(timeout time.Duration) {
	if timeout > 0 {
		p.timeout = timeout
	}
}
