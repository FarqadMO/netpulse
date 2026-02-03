package daemon

import (
	"context"
	"sync"
	"time"

	"github.com/user/netpulse/internal/util"
)

// Job represents a scheduled job.
type Job struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context) error
	
	// State
	lastRun    time.Time
	nextRun    time.Time
	lastError  error
	errorCount int
	running    bool
	mu         sync.RWMutex
}

// JobStatus represents the status of a job.
type JobStatus struct {
	Name       string        `json:"name"`
	Interval   time.Duration `json:"interval"`
	LastRun    time.Time     `json:"last_run"`
	NextRun    time.Time     `json:"next_run"`
	LastError  string        `json:"last_error,omitempty"`
	ErrorCount int           `json:"error_count"`
	Running    bool          `json:"running"`
}

// Scheduler manages scheduled jobs.
type Scheduler struct {
	ctx    context.Context
	daemon *Daemon
	jobs   []*Job
	mu     sync.RWMutex
}

// NewScheduler creates a new scheduler.
func NewScheduler(ctx context.Context, daemon *Daemon) *Scheduler {
	return &Scheduler{
		ctx:    ctx,
		daemon: daemon,
		jobs:   make([]*Job, 0),
	}
}

// AddJob adds a job to the scheduler.
func (s *Scheduler) AddJob(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job.nextRun = time.Now().Add(time.Second * 5) // Initial delay
	s.jobs = append(s.jobs, job)
}

// Run starts the scheduler.
func (s *Scheduler) Run() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	util.Info("Scheduler started with %d jobs", len(s.jobs))
	
	for {
		select {
		case <-s.ctx.Done():
			util.Info("Scheduler stopping")
			return
		case now := <-ticker.C:
			s.checkJobs(now)
		}
	}
}

func (s *Scheduler) checkJobs(now time.Time) {
	s.mu.RLock()
	jobs := s.jobs
	s.mu.RUnlock()
	
	for _, job := range jobs {
		job.mu.RLock()
		shouldRun := !job.running && now.After(job.nextRun)
		job.mu.RUnlock()
		
		if shouldRun {
			go s.runJob(job)
		}
	}
}

func (s *Scheduler) runJob(job *Job) {
	job.mu.Lock()
	if job.running {
		job.mu.Unlock()
		return
	}
	job.running = true
	job.lastRun = time.Now()
	job.mu.Unlock()
	
	util.Debug("Running job: %s", job.Name)
	
	// Create job context with timeout
	ctx, cancel := context.WithTimeout(s.ctx, job.Interval)
	defer cancel()
	
	err := job.Run(ctx)
	
	job.mu.Lock()
	job.running = false
	if err != nil {
		job.lastError = err
		job.errorCount++
		util.Warn("Job %s failed: %v", job.Name, err)
		// Shorter retry on error
		job.nextRun = time.Now().Add(job.Interval / 2)
	} else {
		job.lastError = nil
		util.Debug("Job %s completed successfully", job.Name)
		// Adaptive interval: longer if stable
		job.nextRun = time.Now().Add(job.Interval)
	}
	job.mu.Unlock()
}

// GetJobStatuses returns the status of all jobs.
func (s *Scheduler) GetJobStatuses() []JobStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	statuses := make([]JobStatus, len(s.jobs))
	for i, job := range s.jobs {
		job.mu.RLock()
		status := JobStatus{
			Name:       job.Name,
			Interval:   job.Interval,
			LastRun:    job.lastRun,
			NextRun:    job.nextRun,
			ErrorCount: job.errorCount,
			Running:    job.running,
		}
		if job.lastError != nil {
			status.LastError = job.lastError.Error()
		}
		job.mu.RUnlock()
		statuses[i] = status
	}
	
	return statuses
}

// GetJob returns a job by name.
func (s *Scheduler) GetJob(name string) *Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, job := range s.jobs {
		if job.Name == name {
			return job
		}
	}
	return nil
}

// TriggerJob manually triggers a job.
func (s *Scheduler) TriggerJob(name string) bool {
	job := s.GetJob(name)
	if job == nil {
		return false
	}
	
	job.mu.Lock()
	job.nextRun = time.Now()
	job.mu.Unlock()
	
	return true
}
