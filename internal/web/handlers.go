package web

import (
	"encoding/json"
	"net/http"
	"strconv"
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
	since := time.Now().Add(-24 * time.Hour)
	
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
		"traces":     traces,
		"page":       page,
		"limit":      limit,
		"total":      total,
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
