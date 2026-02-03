// Package daemon provides background service functionality.
package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/user/netpulse/internal/storage"
	"github.com/user/netpulse/internal/util"
)

// Daemon manages the background service.
type Daemon struct {
	config     *util.Config
	scheduler  *Scheduler
	db         *storage.DB
	pidFile    string
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	running    bool
	startTime  time.Time
	mu         sync.RWMutex
}

// New creates a new daemon instance.
func New(cfg *util.Config) (*Daemon, error) {
	db, err := storage.Initialize(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	d := &Daemon{
		config:    cfg,
		db:        db,
		pidFile:   filepath.Join(cfg.DataDir, "netpulse.pid"),
		ctx:       ctx,
		cancel:    cancel,
	}
	
	d.scheduler = NewScheduler(ctx, d)
	
	return d, nil
}

// Start starts the daemon.
func (d *Daemon) Start() error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("daemon already running")
	}
	d.running = true
	d.startTime = time.Now()
	d.mu.Unlock()
	
	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	
	util.Info("Daemon starting...")
	
	// Register jobs
	d.registerJobs()
	
	// Start scheduler
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.scheduler.Run()
	}()
	
	// Handle signals
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.handleSignals()
	}()
	
	util.Info("Daemon started with PID %d", os.Getpid())
	
	return nil
}

// Wait waits for the daemon to finish.
func (d *Daemon) Wait() {
	d.wg.Wait()
}

// Stop stops the daemon gracefully.
func (d *Daemon) Stop() error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = false
	d.mu.Unlock()
	
	util.Info("Daemon stopping...")
	
	d.cancel() // Signal all goroutines to stop
	
	// Wait for graceful shutdown with timeout
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		util.Info("Daemon stopped gracefully")
	case <-time.After(30 * time.Second):
		util.Warn("Daemon stop timed out")
	}
	
	// Clean up
	d.removePIDFile()
	if d.db != nil {
		d.db.Close()
	}
	
	return nil
}

func (d *Daemon) handleSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	select {
	case sig := <-sigCh:
		util.Info("Received signal: %v", sig)
		d.Stop()
	case <-d.ctx.Done():
		return
	}
}

func (d *Daemon) writePIDFile() error {
	pid := os.Getpid()
	return os.WriteFile(d.pidFile, []byte(strconv.Itoa(pid)), 0644)
}

func (d *Daemon) removePIDFile() {
	os.Remove(d.pidFile)
}

// IsRunning returns whether the daemon is running.
func (d *Daemon) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// GetStatus returns the daemon status.
func (d *Daemon) GetStatus() *DaemonStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	return &DaemonStatus{
		Running:   d.running,
		PID:       os.Getpid(),
		StartTime: d.startTime,
		Uptime:    time.Since(d.startTime),
		Jobs:      d.scheduler.GetJobStatuses(),
	}
}

// DaemonStatus holds the current daemon status.
type DaemonStatus struct {
	Running   bool
	PID       int
	StartTime time.Time
	Uptime    time.Duration
	Jobs      []JobStatus
}

// GetDB returns the database instance.
func (d *Daemon) GetDB() *storage.DB {
	return d.db
}

// GetConfig returns the configuration.
func (d *Daemon) GetConfig() *util.Config {
	return d.config
}

// GetContext returns the daemon context.
func (d *Daemon) GetContext() context.Context {
	return d.ctx
}
