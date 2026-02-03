package probes

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/user/netpulse/internal/model"
)

// PingProbe handles ping sweep operations.
type PingProbe struct {
	concurrency int
	timeout     time.Duration
}

// NewPingProbe creates a new ping probe.
func NewPingProbe(concurrency int, timeout time.Duration) *PingProbe {
	if concurrency <= 0 {
		concurrency = 50
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &PingProbe{
		concurrency: concurrency,
		timeout:     timeout,
	}
}

// SweepSubnet performs a ping sweep on the given CIDR subnet.
func (p *PingProbe) SweepSubnet(ctx context.Context, cidr string) ([]model.ScanHost, error) {
	ips, err := expandCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}
	
	// Worker pool
	jobs := make(chan string, len(ips))
	results := make(chan model.ScanHost, len(ips))
	
	var wg sync.WaitGroup
	for i := 0; i < p.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					host := p.pingHost(ctx, ip)
					results <- host
				}
			}
		}()
	}
	
	// Send jobs
	go func() {
		for _, ip := range ips {
			select {
			case jobs <- ip:
			case <-ctx.Done():
				break
			}
		}
		close(jobs)
	}()
	
	// Wait and close results
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// Collect results
	var hosts []model.ScanHost
	for host := range results {
		hosts = append(hosts, host)
	}
	
	return hosts, nil
}

// pingHost pings a single host using TCP SYN (connect) method.
func (p *PingProbe) pingHost(ctx context.Context, ip string) model.ScanHost {
	host := model.ScanHost{
		IP:       ip,
		LastSeen: time.Now(),
		Alive:    false,
	}
	
	// Try common ports for TCP ping
	ports := []int{80, 443, 22, 21, 445, 139}
	
	for _, port := range ports {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), p.timeout)
		latency := float64(time.Since(start).Microseconds()) / 1000.0
		
		if err == nil {
			conn.Close()
			host.Alive = true
			host.LatencyMs = latency
			break
		}
		
		// Connection refused means host is alive but port closed
		if isConnectionRefused(err) {
			host.Alive = true
			host.LatencyMs = latency
			break
		}
	}
	
	// Attempt reverse DNS lookup for alive hosts
	if host.Alive {
		if names, err := net.LookupAddr(ip); err == nil && len(names) > 0 {
			host.Hostname = names[0]
		}
	}
	
	return host
}

// PingHosts pings a list of specific hosts.
func (p *PingProbe) PingHosts(ctx context.Context, ips []string) []model.ScanHost {
	results := make([]model.ScanHost, len(ips))
	
	var wg sync.WaitGroup
	sem := make(chan struct{}, p.concurrency)
	
	for i, ip := range ips {
		wg.Add(1)
		go func(idx int, addr string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			results[idx] = p.pingHost(ctx, addr)
		}(i, ip)
	}
	
	wg.Wait()
	return results
}

// expandCIDR expands a CIDR to a list of IP addresses.
func expandCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	
	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
		ips = append(ips, ip.String())
	}
	
	// Remove network and broadcast addresses for /24 and smaller
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}
	
	return ips, nil
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func isConnectionRefused(err error) bool {
	if opErr, ok := err.(*net.OpError); ok {
		// Check for specific syscall errors
		if syscallErr, ok := opErr.Err.(*net.OpError); ok {
			opErr = syscallErr
		}
		// Check the error message for "connection refused"
		errStr := opErr.Error()
		if contains(errStr, "connection refused") || 
		   contains(errStr, "refused") ||
		   contains(errStr, "reset") {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
