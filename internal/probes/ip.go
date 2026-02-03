// Package probes provides network probing functionality.
package probes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// IPProvider represents a public IP provider.
type IPProvider struct {
	Name string
	URL  string
}

// DefaultIPProviders returns the default IP providers.
func DefaultIPProviders() []IPProvider {
	return []IPProvider{
		{Name: "ipify", URL: "https://api.ipify.org"},
		{Name: "ifconfig.me", URL: "https://ifconfig.me/ip"},
		{Name: "icanhazip", URL: "https://icanhazip.com"},
	}
}

// IPProbe handles public IP detection.
type IPProbe struct {
	providers []IPProvider
	client    *http.Client
	timeout   time.Duration
}

// NewIPProbe creates a new IP probe.
func NewIPProbe() *IPProbe {
	return &IPProbe{
		providers: DefaultIPProviders(),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		timeout: 10 * time.Second,
	}
}

// GetPublicIP returns the public IP using consensus from multiple providers.
func (p *IPProbe) GetPublicIP(ctx context.Context) (string, error) {
	results := make(chan string, len(p.providers))
	errors := make(chan error, len(p.providers))
	
	for _, provider := range p.providers {
		go func(prov IPProvider) {
			ip, err := p.fetchIP(ctx, prov.URL)
			if err != nil {
				errors <- fmt.Errorf("%s: %w", prov.Name, err)
				return
			}
			results <- ip
		}(provider)
	}
	
	// Collect results
	var ips []string
	var errs []error
	
	timeout := time.After(p.timeout)
	for i := 0; i < len(p.providers); i++ {
		select {
		case ip := <-results:
			ips = append(ips, ip)
		case err := <-errors:
			errs = append(errs, err)
		case <-timeout:
			break
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	
	if len(ips) == 0 {
		if len(errs) > 0 {
			return "", fmt.Errorf("all providers failed: %v", errs)
		}
		return "", fmt.Errorf("no IP providers responded")
	}
	
	// Return consensus IP (most common)
	return p.consensus(ips), nil
}

func (p *IPProbe) fetchIP(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "netpulse/1.0")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}
	
	ip := strings.TrimSpace(string(body))
	if !isValidIP(ip) {
		return "", fmt.Errorf("invalid IP: %s", ip)
	}
	
	return ip, nil
}

func (p *IPProbe) consensus(ips []string) string {
	counts := make(map[string]int)
	for _, ip := range ips {
		counts[ip]++
	}
	
	var maxIP string
	var maxCount int
	for ip, count := range counts {
		if count > maxCount {
			maxIP = ip
			maxCount = count
		}
	}
	
	return maxIP
}

func isValidIP(ip string) bool {
	// Basic validation: contains dots (IPv4) or colons (IPv6)
	if len(ip) < 7 || len(ip) > 45 {
		return false
	}
	return strings.Contains(ip, ".") || strings.Contains(ip, ":")
}

// ASNInfo holds ASN and ISP information.
type ASNInfo struct {
	IP      string `json:"query"`
	ASN     string `json:"as"`
	ISP     string `json:"isp"`
	Org     string `json:"org"`
	Country string `json:"country"`
	City    string `json:"city"`
}

// GetASNInfo fetches ASN and ISP information for an IP.
func GetASNInfo(ctx context.Context, ip string) (*ASNInfo, error) {
	// Using ip-api.com (free, no auth required)
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=query,as,isp,org,country,city", ip)
	
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ASN info: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ASN API returned status %d", resp.StatusCode)
	}
	
	var info ASNInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode ASN response: %w", err)
	}
	
	return &info, nil
}
