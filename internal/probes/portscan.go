package probes

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/user/netpulse/internal/model"
)

// PortScanner handles port scanning operations.
type PortScanner struct {
	concurrency int
	timeout     time.Duration
	ports       []int
}

// NewPortScanner creates a new port scanner.
func NewPortScanner(concurrency int, timeout time.Duration, ports []int) *PortScanner {
	if concurrency <= 0 {
		concurrency = 20
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if len(ports) == 0 {
		ports = DefaultPorts()
	}
	return &PortScanner{
		concurrency: concurrency,
		timeout:     timeout,
		ports:       ports,
	}
}

// DefaultPorts returns the top 50 most common ports.
func DefaultPorts() []int {
	return []int{
		21, 22, 23, 25, 53, 80, 110, 111, 135, 139,
		143, 443, 445, 993, 995, 1723, 3306, 3389, 5432, 5900,
		8080, 8443, 8888, 27017, 6379, 11211, 1433, 1521, 5984, 9200,
		2181, 9092, 6443, 10250, 2379, 4443, 7443, 8000, 8001, 8002,
		9000, 9001, 9090, 9091, 9443, 10000, 10443, 15672, 27018, 27019,
	}
}

// Port service names mapping.
var serviceNames = map[int]string{
	21:    "ftp", 22: "ssh", 23: "telnet", 25: "smtp", 53: "dns",
	80:    "http", 110: "pop3", 111: "rpc", 135: "msrpc", 139: "netbios",
	143:   "imap", 443: "https", 445: "smb", 993: "imaps", 995: "pop3s",
	1433:  "mssql", 1521: "oracle", 1723: "pptp", 3306: "mysql", 3389: "rdp",
	5432:  "postgresql", 5900: "vnc", 5984: "couchdb", 6379: "redis",
	8080:  "http-alt", 8443: "https-alt", 8888: "http-alt", 9092: "kafka",
	9200:  "elasticsearch", 11211: "memcached", 27017: "mongodb",
}

// ScanResult represents the result of scanning a single port.
type ScanResult struct {
	Port    int
	State   string
	Service string
	Banner  string
}

// ScanHost scans a single host for open ports.
func (s *PortScanner) ScanHost(ctx context.Context, host string) ([]model.ScanPort, error) {
	jobs := make(chan int, len(s.ports))
	results := make(chan *ScanResult, len(s.ports))
	
	var wg sync.WaitGroup
	for i := 0; i < s.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					result := s.scanPort(ctx, host, port)
					if result != nil {
						results <- result
					}
				}
			}
		}()
	}
	
	// Send jobs
	go func() {
		for _, port := range s.ports {
			select {
			case jobs <- port:
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
	var ports []model.ScanPort
	for result := range results {
		ports = append(ports, model.ScanPort{
			Port:     result.Port,
			Protocol: "tcp",
			Service:  result.Service,
			State:    result.State,
			Banner:   result.Banner,
			LastSeen: time.Now(),
		})
	}
	
	return ports, nil
}

func (s *PortScanner) scanPort(ctx context.Context, host string, port int) *ScanResult {
	addr := fmt.Sprintf("%s:%d", host, port)
	
	conn, err := net.DialTimeout("tcp", addr, s.timeout)
	if err != nil {
		return nil // Port closed or filtered
	}
	defer conn.Close()
	
	result := &ScanResult{
		Port:    port,
		State:   "open",
		Service: getServiceName(port),
	}
	
	// Try to grab banner
	result.Banner = grabBanner(conn, s.timeout/2)
	
	return result
}

func getServiceName(port int) string {
	if name, ok := serviceNames[port]; ok {
		return name
	}
	return "unknown"
}

func grabBanner(conn net.Conn, timeout time.Duration) string {
	conn.SetReadDeadline(time.Now().Add(timeout))
	
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return ""
	}
	
	// Clean up banner
	banner := string(buf[:n])
	if len(banner) > 200 {
		banner = banner[:200]
	}
	
	return banner
}

// ScanMultipleHosts scans multiple hosts for open ports.
func (s *PortScanner) ScanMultipleHosts(ctx context.Context, hosts []string) (map[string][]model.ScanPort, error) {
	results := make(map[string][]model.ScanPort)
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	sem := make(chan struct{}, 5) // Limit concurrent host scans
	
	for _, host := range hosts {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			ports, err := s.ScanHost(ctx, h)
			if err == nil && len(ports) > 0 {
				mu.Lock()
				results[h] = ports
				mu.Unlock()
			}
		}(host)
	}
	
	wg.Wait()
	return results, nil
}

// SetPorts sets the ports to scan.
func (s *PortScanner) SetPorts(ports []int) {
	if len(ports) > 0 {
		s.ports = ports
	}
}
