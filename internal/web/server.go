// Package web provides a lightweight web dashboard.
package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/user/netpulse/internal/model"
	"github.com/user/netpulse/internal/monitor"
	"github.com/user/netpulse/internal/storage"
	"github.com/user/netpulse/internal/util"
)

// Server is the web server.
type Server struct {
	db     *storage.DB
	config *util.Config
	port   int
	srv    *http.Server
}

// NewServer creates a new web server.
func NewServer(db *storage.DB, cfg *util.Config, port int) *Server {
	return &Server{
		db:     db,
		config: cfg,
		port:   port,
	}
}

// Start starts the web server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register routes
	h := NewHandlers(s.db, s.config)
	a := NewAnalyticsHandlers(s.db, s.config)

	mux.HandleFunc("/", h.Dashboard)
	mux.HandleFunc("/api/ip", h.APIGetIP)
	mux.HandleFunc("/api/ip/history", h.APIGetIPHistory)
	mux.HandleFunc("/api/traces", h.APIGetTraces)
	mux.HandleFunc("/api/traces/", h.TraceGeoHandler) // Handles /api/traces/{id}/geo
	mux.HandleFunc("/api/traces/by-target", h.APIGetTracesByTarget)
	mux.HandleFunc("/api/public-ip-at-time", h.APIGetPublicIPAtTime)
	mux.HandleFunc("/api/hosts", h.APIGetHosts)
	mux.HandleFunc("/api/hosts/", h.APIUpdateHostMetadata) // Handles /api/hosts/{id}/metadata
	mux.HandleFunc("/api/status", h.APIGetStatus)
	mux.HandleFunc("/api/dns/history", h.APIGetDNSHistory)
	mux.HandleFunc("/api/dns/targets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.APIAddDNSTarget(w, r)
		} else {
			h.APIGetDNSTargets(w, r)
		}
	})
	mux.HandleFunc("/api/dns/targets/", h.APIDeleteDNSTarget)
	mux.HandleFunc("/api/geoip", h.GeoIPHandler)
	mux.HandleFunc("/report", h.DownloadReport)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(GetStaticFS())))

	// Analytics routes
	mux.HandleFunc("/api/analytics/topology", a.GetTopology)
	mux.HandleFunc("/api/analytics/latency", a.GetLatencyTrends)
	mux.HandleFunc("/api/analytics/anomalies", a.GetAnomalies)
	mux.HandleFunc("/api/analytics/mermaid", a.MermaidDiagram)

	// Start DNS Monitor (1 minute interval)
	go monitor.Run(1*time.Minute, func() ([]model.DNSTarget, error) {
		return s.db.GetDNSTargets()
	}, func(m model.DNSMetric) {
		if err := s.db.SaveDNSMetric(m); err != nil {
			util.Error("DNS Monitor Save: %v", err)
		}
	})

	s.srv = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		s.srv.Shutdown(ctx)
	}()

	util.Info("Web server starting on port %d", s.port)

	if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Stop stops the web server.
func (s *Server) Stop() error {
	if s.srv == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.srv.Shutdown(ctx)
}
