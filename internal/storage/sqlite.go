// Package storage provides SQLite persistence for netpulse.
package storage

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection.
type DB struct {
	*sql.DB
	mu sync.RWMutex
}

var (
	instance *DB
	once     sync.Once
)

// GetDB returns the singleton database instance.
func GetDB() *DB {
	return instance
}

// Initialize creates and initializes the database.
func Initialize(dataDir string) (*DB, error) {
	var initErr error
	once.Do(func() {
		dbPath := filepath.Join(dataDir, "netpulse.db")
		db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL&_busy_timeout=5000")
		if err != nil {
			initErr = fmt.Errorf("failed to open database: %w", err)
			return
		}
		
		// Set connection pool settings
		db.SetMaxOpenConns(1) // SQLite only supports one writer
		db.SetMaxIdleConns(1)
		
		instance = &DB{DB: db}
		
		if err := instance.createTables(); err != nil {
			initErr = fmt.Errorf("failed to create tables: %w", err)
			return
		}
	})
	
	return instance, initErr
}

func (db *DB) createTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS ip_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT NOT NULL,
			asn TEXT,
			isp TEXT,
			country TEXT,
			city TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ip_history_timestamp ON ip_history(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_ip_history_ip ON ip_history(ip)`,
		
		`CREATE TABLE IF NOT EXISTS traces (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_timestamp ON traces(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_target ON traces(target)`,
		
		`CREATE TABLE IF NOT EXISTS trace_hops (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trace_id INTEGER NOT NULL,
			hop_num INTEGER NOT NULL,
			ip TEXT,
			hostname TEXT,
			latency_ms REAL,
			lost INTEGER DEFAULT 0,
			FOREIGN KEY (trace_id) REFERENCES traces(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_trace_hops_trace_id ON trace_hops(trace_id)`,
		
		`CREATE TABLE IF NOT EXISTS scan_hosts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT NOT NULL UNIQUE,
			hostname TEXT,
			alive INTEGER DEFAULT 0,
			latency_ms REAL,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			display_name TEXT,
			tags TEXT,
			icon TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_hosts_ip ON scan_hosts(ip)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_hosts_alive ON scan_hosts(alive)`,
		
		`CREATE TABLE IF NOT EXISTS scan_ports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host_id INTEGER NOT NULL,
			port INTEGER NOT NULL,
			protocol TEXT DEFAULT 'tcp',
			service TEXT,
			state TEXT DEFAULT 'open',
			banner TEXT,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (host_id) REFERENCES scan_hosts(id) ON DELETE CASCADE,
			UNIQUE(host_id, port, protocol)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_ports_host_id ON scan_ports(host_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_ports_port ON scan_ports(port)`,
		
		`CREATE TABLE IF NOT EXISTS anomalies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			description TEXT,
			severity TEXT DEFAULT 'info',
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			data TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_anomalies_timestamp ON anomalies(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_anomalies_type ON anomalies(type)`,
	}
	
	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("failed to execute: %s: %w", table, err)
		}
	}

	// Migrations: Try to add columns if they don't exist (for existing DBs)
	// We ignore errors here because if the column exists, it will fail, which is fine.
	migrations := []string{
		"ALTER TABLE scan_hosts ADD COLUMN display_name TEXT",
		"ALTER TABLE scan_hosts ADD COLUMN tags TEXT",
		"ALTER TABLE scan_hosts ADD COLUMN icon TEXT",
	}
	for _, m := range migrations {
		db.Exec(m)
	}
	
	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.DB.Close()
}

// WithLock executes a function with write lock.
func (db *DB) WithLock(fn func() error) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return fn()
}

// WithRLock executes a function with read lock.
func (db *DB) WithRLock(fn func() error) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return fn()
}
