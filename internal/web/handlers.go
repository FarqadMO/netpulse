package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/user/netpulse/internal/daemon"
	"github.com/user/netpulse/internal/model"
	"github.com/user/netpulse/internal/report"
	"github.com/user/netpulse/internal/storage"
	"github.com/user/netpulse/internal/util"
)

// Handlers contains HTTP handlers.
type Handlers struct {
	db     *storage.DB
	config *util.Config
}

// NewHandlers creates new handlers.
func NewHandlers(db *storage.DB, cfg *util.Config) *Handlers {
	return &Handlers{
		db:     db,
		config: cfg,
	}
}

// Dashboard serves the main dashboard page.
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := h.getDashboardData()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := GetTemplates()
	if err := tmpl.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// APIGetIP returns the current IP.
func (h *Handlers) APIGetIP(w http.ResponseWriter, r *http.Request) {
	ipStorage := storage.NewIPStorage(h.db)
	latest, err := ipStorage.GetLatest()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	writeJSON(w, latest)
}

// APIGetIPHistory returns IP history.
func (h *Handlers) APIGetIPHistory(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)

	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if d, err := time.ParseDuration(sinceStr); err == nil {
			since = time.Now().Add(-d)
		}
	}

	ipStorage := storage.NewIPStorage(h.db)
	records, err := ipStorage.GetHistory(since)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	writeJSON(w, records)
}

// APIGetTraces returns trace history with pagination.
func (h *Handlers) APIGetTraces(w http.ResponseWriter, r *http.Request) {
	// Parse time range params
	since := time.Now().Add(-24 * time.Hour)
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr != "" && endStr != "" {
		start, err1 := time.Parse(time.RFC3339, startStr)
		end, err2 := time.Parse(time.RFC3339, endStr)
		if err1 == nil && err2 == nil {
			since = start
			// Note: we're using "since" for backward compatibility,
			// but in future we might want to add an explicit end time filter
			_ = end
		}
	}

	// Parse pagination params
	page := 1
	limit := 10
	if p := r.URL.Query().Get("page"); p != "" {
		if pn, err := strconv.Atoi(p); err == nil && pn > 0 {
			page = pn
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if ln, err := strconv.Atoi(l); err == nil && ln > 0 && ln <= 50 {
			limit = ln
		}
	}

	// Get target filter
	target := r.URL.Query().Get("target")

	traceStorage := storage.NewTraceStorage(h.db)
	var traces []model.TraceResult
	var err error

	if target != "" {
		traces, err = traceStorage.GetHistory(target, since)
	} else {
		traces, err = traceStorage.GetAllHistory(since)
	}

	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	// Apply pagination
	total := len(traces)
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		traces = []model.TraceResult{}
	} else {
		if end > total {
			end = total
		}
		traces = traces[start:end]
	}

	// Return with pagination metadata
	result := map[string]interface{}{
		"traces":      traces,
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": (total + limit - 1) / limit,
	}

	writeJSON(w, result)
}

// APIGetHosts returns discovered hosts.
func (h *Handlers) APIGetHosts(w http.ResponseWriter, r *http.Request) {
	scanStorage := storage.NewScanStorage(h.db)
	hosts, err := scanStorage.GetAliveHosts()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	// Populate ports
	for i := range hosts {
		ports, _ := scanStorage.GetHostPorts(hosts[i].ID)
		hosts[i].Ports = ports
	}

	writeJSON(w, hosts)
}

// APIGetStatus returns daemon status.
func (h *Handlers) APIGetStatus(w http.ResponseWriter, r *http.Request) {
	running, pid := daemon.CheckRunning(h.config.DataDir)

	status := map[string]interface{}{
		"running": running,
		"pid":     pid,
	}

	// Add database stats
	ipStorage := storage.NewIPStorage(h.db)
	if count, err := ipStorage.Count(); err == nil {
		status["ip_records"] = count
	}
	if latest, err := ipStorage.GetLatest(); err == nil && latest != nil {
		status["current_ip"] = latest.IP
		status["isp"] = latest.ISP
		status["asn"] = latest.ASN
		status["last_check"] = latest.Timestamp.Format("2006-01-02 15:04:05")
	}

	scanStorage := storage.NewScanStorage(h.db)
	if count, err := scanStorage.CountAliveHosts(); err == nil {
		status["alive_hosts"] = count
	}
	if count, err := scanStorage.CountOpenPorts(); err == nil {
		status["open_ports"] = count
	}

	writeJSON(w, status)
}

// DownloadReport generates and downloads a report.
func (h *Handlers) DownloadReport(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)

	gen := report.NewGenerator(h.db, h.config)
	opts := model.ReportOptions{
		Since:        since,
		Until:        time.Now(),
		IncludeIP:    true,
		IncludeTrace: true,
		IncludeScan:  true,
	}

	data, err := gen.Generate(opts)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	content := report.FormatMarkdown(data)

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=netpulse_report.md")
	w.Write([]byte(content))
}

// ServeStatic serves static files.
func (h *Handlers) ServeStatic(w http.ResponseWriter, r *http.Request) {
	// For now, return 404 for static files
	// In a production version, this would serve CSS/JS files
	http.NotFound(w, r)
}

func (h *Handlers) getDashboardData() map[string]interface{} {
	data := make(map[string]interface{})

	ipStorage := storage.NewIPStorage(h.db)
	if latest, err := ipStorage.GetLatest(); err == nil && latest != nil {
		data["current_ip"] = latest.IP
		data["isp"] = latest.ISP
		data["asn"] = latest.ASN
		data["last_check"] = latest.Timestamp.Format("2006-01-02 15:04:05")
	}

	if count, err := ipStorage.Count(); err == nil {
		data["ip_count"] = count
	}

	// Get IP history
	since := time.Now().Add(-24 * time.Hour)
	if history, err := ipStorage.GetHistory(since); err == nil {
		data["ip_history"] = history
	}

	scanStorage := storage.NewScanStorage(h.db)
	if hosts, err := scanStorage.GetAliveHosts(); err == nil {
		// Populate ports
		for i := range hosts {
			ports, _ := scanStorage.GetHostPorts(hosts[i].ID)
			hosts[i].Ports = ports
		}
		data["hosts"] = hosts
		data["host_count"] = len(hosts)
	}

	if count, err := scanStorage.CountOpenPorts(); err == nil {
		data["port_count"] = count
	}

	// Get traces
	traceStorage := storage.NewTraceStorage(h.db)
	if traces, err := traceStorage.GetAllHistory(since); err == nil {
		data["traces"] = traces
	}

	// Get trace targets from config
	data["trace_targets"] = h.config.TraceTargets

	running, _ := daemon.CheckRunning(h.config.DataDir)
	data["daemon_running"] = running

	return data
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

type UpdateHostMetadataRequest struct {
	DisplayName string   `json:"display_name"`
	Tags        []string `json:"tags"`
	Icon        string   `json:"icon"`
}

func (h *Handlers) APIUpdateHostMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, fmt.Errorf("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 { // /api/hosts/{id}/metadata or similar
		writeError(w, fmt.Errorf("invalid path"), http.StatusBadRequest)
		return
	}
	// Assuming path /api/hosts/{id}/metadata. parts: ["", "api", "hosts", "123", "metadata"]
	// Index 3 is ID.
	idStr := parts[3]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, fmt.Errorf("invalid id"), http.StatusBadRequest)
		return
	}

	var req UpdateHostMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	scanStorage := storage.NewScanStorage(h.db)
	if err := scanStorage.UpdateHostMetadata(id, req.DisplayName, req.Tags, req.Icon); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, map[string]string{"status": "ok"})
}

// APIGetDNSHistory returns DNS latency history
func (h *Handlers) APIGetDNSHistory(w http.ResponseWriter, r *http.Request) {
	// Time Range Filter
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr != "" && endStr != "" {
		start, err1 := time.Parse(time.RFC3339, startStr)
		end, err2 := time.Parse(time.RFC3339, endStr)
		if err1 == nil && err2 == nil {
			metrics, err := h.db.GetDNSHistoryTimeRange(start, end)
			if err != nil {
				writeError(w, err, http.StatusInternalServerError)
				return
			}
			writeJSON(w, metrics)
			return
		}
	}

	// Default Limit
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	metrics, err := h.db.GetDNSHistory(limit)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	writeJSON(w, metrics)
}

// APIGetDNSTargets returns all monitored targets
func (h *Handlers) APIGetDNSTargets(w http.ResponseWriter, r *http.Request) {
	targets, err := h.db.GetDNSTargets()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, targets)
}

// APIAddDNSTarget adds a new target
func (h *Handlers) APIAddDNSTarget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, fmt.Errorf("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	var req model.DNSTarget
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	if err := h.db.AddDNSTarget(req); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

// APIDeleteDNSTarget removes a target
func (h *Handlers) APIDeleteDNSTarget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, fmt.Errorf("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 { // /api/dns/targets/{id}
		writeError(w, fmt.Errorf("invalid path"), http.StatusBadRequest)
		return
	}
	idStr := parts[len(parts)-1]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, fmt.Errorf("invalid id"), http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteDNSTarget(id); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

// APIGetTracesByTarget returns all traces for a specific target
func (h *Handlers) APIGetTracesByTarget(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		writeError(w, fmt.Errorf("target parameter required"), http.StatusBadRequest)
		return
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	traceStorage := storage.NewTraceStorage(h.db)
	traces, err := traceStorage.GetTracesForTarget(target, limit)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	writeJSON(w, traces)
}

// APIGetPublicIPAtTime returns the public IP at a given timestamp
func (h *Handlers) APIGetPublicIPAtTime(w http.ResponseWriter, r *http.Request) {
	timestampStr := r.URL.Query().Get("timestamp")
	if timestampStr == "" {
		writeError(w, fmt.Errorf("timestamp parameter required"), http.StatusBadRequest)
		return
	}

	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		writeError(w, fmt.Errorf("invalid timestamp format"), http.StatusBadRequest)
		return
	}

	ipStorage := storage.NewIPStorage(h.db)
	// Get IP records around that time (within 30 minutes before/after)
	since := timestamp.Add(-30 * time.Minute)
	records, err := ipStorage.GetHistory(since)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	// Find the closest record to the requested timestamp
	var closest *model.IPRecord
	minDiff := time.Hour * 24 * 365 // 1 year as max

	for i := range records {
		diff := records[i].Timestamp.Sub(timestamp)
		if diff < 0 {
			diff = -diff
		}
		if diff < minDiff {
			minDiff = diff
			closest = &records[i]
		}
	}

	if closest == nil {
		writeJSON(w, map[string]interface{}{
			"ip":  "Unknown",
			"isp": "",
			"asn": "",
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"ip":        closest.IP,
		"isp":       closest.ISP,
		"asn":       closest.ASN,
		"timestamp": closest.Timestamp,
	})
}
