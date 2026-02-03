package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/user/netpulse/internal/storage"
)

// GeoIP represents geographical IP information.
type GeoIP struct {
	IP      string  `json:"ip"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	City    string  `json:"city"`
	Country string  `json:"country"`
	ISP     string  `json:"isp"`
}

// GeoIPCache caches GeoIP lookups.
type GeoIPCache struct {
	mu    sync.RWMutex
	cache map[string]*GeoIP
	ttl   time.Duration
}

// NewGeoIPCache creates a new cache.
func NewGeoIPCache() *GeoIPCache {
	return &GeoIPCache{
		cache: make(map[string]*GeoIP),
		ttl:   24 * time.Hour,
	}
}

var geoCache = NewGeoIPCache()

// LookupIP looks up geographical information for an IP.
func LookupIP(ip string) (*GeoIP, error) {
	// Check cache
	geoCache.mu.RLock()
	if cached, ok := geoCache.cache[ip]; ok {
		geoCache.mu.RUnlock()
		return cached, nil
	}
	geoCache.mu.RUnlock()
	
	// Use ip-api.com (free, no API key, 45 req/min limit)
	resp, err := http.Get("http://ip-api.com/json/" + ip + "?fields=status,country,city,lat,lon,isp,query")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result struct {
		Status  string  `json:"status"`
		Country string  `json:"country"`
		City    string  `json:"city"`
		Lat     float64 `json:"lat"`
		Lon     float64 `json:"lon"`
		ISP     string  `json:"isp"`
		Query   string  `json:"query"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	geo := &GeoIP{
		IP:      ip,
		Lat:     result.Lat,
		Lon:     result.Lon,
		City:    result.City,
		Country: result.Country,
		ISP:     result.ISP,
	}
	
	// Cache result
	geoCache.mu.Lock()
	geoCache.cache[ip] = geo
	geoCache.mu.Unlock()
	
	return geo, nil
}

// GeoIPHandler handles single IP lookup.
func (h *Handlers) GeoIPHandler(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		writeError(w, nil, http.StatusBadRequest)
		return
	}
	
	geo, err := LookupIP(ip)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	
	writeJSON(w, geo)
}

// TraceGeoData represents a trace with geo coordinates.
type TraceGeoData struct {
	ID        int64          `json:"id"`
	Target    string         `json:"target"`
	Timestamp time.Time      `json:"timestamp"`
	Hops      []HopGeoData   `json:"hops"`
}

// HopGeoData represents a hop with geo coordinates.
type HopGeoData struct {
	HopNum    int     `json:"hop_num"`
	IP        string  `json:"ip"`
	LatencyMs float64 `json:"latency_ms"`
	Lost      bool    `json:"lost"`
	Lat       float64 `json:"lat,omitempty"`
	Lon       float64 `json:"lon,omitempty"`
	City      string  `json:"city,omitempty"`
	Country   string  `json:"country,omitempty"`
}

// TraceGeoHandler returns a trace with geo coordinates for all hops.
func (h *Handlers) TraceGeoHandler(w http.ResponseWriter, r *http.Request) {
	// Parse trace ID from URL path
	path := r.URL.Path
	// Expected: /api/traces/{id}/geo
	var traceID int64
	n, err := parseTraceIDFromPath(path)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	traceID = n
	
	// Get trace
	traceStorage := storage.NewTraceStorage(h.db)
	trace, err := traceStorage.GetByID(traceID)
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}
	
	// Build geo data
	result := TraceGeoData{
		ID:        trace.ID,
		Target:    trace.Target,
		Timestamp: trace.Timestamp,
		Hops:      make([]HopGeoData, 0, len(trace.Hops)),
	}
	
	for _, hop := range trace.Hops {
		hopGeo := HopGeoData{
			HopNum:    hop.HopNum,
			IP:        hop.IP,
			LatencyMs: hop.LatencyMs,
			Lost:      hop.Lost,
		}
		
		if !hop.Lost && hop.IP != "" {
			// Lookup geo (with rate limiting consideration)
			if geo, err := LookupIP(hop.IP); err == nil && geo.Lat != 0 {
				hopGeo.Lat = geo.Lat
				hopGeo.Lon = geo.Lon
				hopGeo.City = geo.City
				hopGeo.Country = geo.Country
			}
			// Small delay to avoid rate limiting
			time.Sleep(50 * time.Millisecond)
		}
		
		result.Hops = append(result.Hops, hopGeo)
	}
	
	writeJSON(w, result)
}

func parseTraceIDFromPath(path string) (int64, error) {
	// Path: /api/traces/123/geo
	// Split and find the ID
	parts := []string{}
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	
	// Find "traces" then get next part as ID
	for i, p := range parts {
		if p == "traces" && i+1 < len(parts) {
			return strconv.ParseInt(parts[i+1], 10, 64)
		}
	}
	
	return 0, nil
}
