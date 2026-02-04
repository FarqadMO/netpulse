package monitor

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/user/netpulse/internal/model"
)

var client = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// DoHResponse represents a DNS-over-HTTPS response
type DoHResponse struct {
	Status int `json:"Status"`
	Answer []struct {
		Type int    `json:"type"`
		Data string `json:"data"`
	} `json:"Answer"`
}

// MeasureUDP resolves google.com via UDP 53 and returns latency + IP
func MeasureUDP(resolverIP string) (int, string, error) {
	resolverAddr := resolverIP + ":53"
	start := time.Now()
	
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 2 * time.Second}
			return d.DialContext(ctx, "udp", resolverAddr)
		},
	}
	
	ips, err := r.LookupHost(context.Background(), "google.com")
	if err != nil {
		return 0, "", err
	}
	
	resolvedIP := ""
	if len(ips) > 0 {
		resolvedIP = ips[0]
	}
	
	return int(time.Since(start).Milliseconds()), resolvedIP, nil
}

// MeasureDoH performs a DNS-over-HTTPS check and returns latency + IP
func MeasureDoH(url string) (int, string, error) {
	start := time.Now()
	req, err := http.NewRequest("GET", url+"?name=google.com&type=A", nil)
    if err != nil { return 0, "", err }
    req.Header.Set("Accept", "application/dns-json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return 0, "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

    var doh DoHResponse
    if err := json.NewDecoder(resp.Body).Decode(&doh); err != nil {
        return 0, "", err
    }
    
    resolvedIP := ""
    for _, ans := range doh.Answer {
        if ans.Type == 1 { // A Record
            resolvedIP = ans.Data
            break
        }
    }
	
	return int(time.Since(start).Milliseconds()), resolvedIP, nil
}

// TargetProvider returns list of targets to monitor
type TargetProvider func() ([]model.DNSTarget, error)

// Run starts the monitoring loop
func Run(interval time.Duration, provider TargetProvider, callback func(model.DNSMetric)) {
    // Default targets
	defaults := []model.DNSTarget{
		{Name: "Google", IP: "8.8.8.8", DoHURL: "https://dns.google/resolve"},
		{Name: "Cloudflare", IP: "1.1.1.1", DoHURL: "https://cloudflare-dns.com/dns-query"},
		{Name: "Quad9", IP: "9.9.9.9", DoHURL: "https://dns.quad9.net/dns-query"},
	}

	ticker := time.NewTicker(interval)
	
	check := func() {
        targets := make([]model.DNSTarget, len(defaults))
        copy(targets, defaults)
        
        // Add custom targets
        if custom, err := provider(); err == nil {
            targets = append(targets, custom...)
        }

		for _, t := range targets {
			// UDP
            if t.IP != "" {
			    if lat, ip, err := MeasureUDP(t.IP); err == nil {
				    callback(model.DNSMetric{
                        Server: t.Name, 
                        Protocol: "udp", 
                        ResolvedIP: ip,
                        LatencyMs: lat, 
                        Timestamp: time.Now(),
                    })
			    }
            }
			// DoH
            if t.DoHURL != "" {
			    if lat, ip, err := MeasureDoH(t.DoHURL); err == nil {
				    callback(model.DNSMetric{
                        Server: t.Name, 
                        Protocol: "doh", 
                        ResolvedIP: ip,
                        LatencyMs: lat, 
                        Timestamp: time.Now(),
                    })
			    }
            }
		}
	}

    go func() {
        check() // Run immediately
        for range ticker.C {
            check()
        }
    }()
}
