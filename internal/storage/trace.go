package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/user/netpulse/internal/model"
)

// TraceStorage handles traceroute persistence.
type TraceStorage struct {
	db *DB
}

// NewTraceStorage creates a new trace storage handler.
func NewTraceStorage(db *DB) *TraceStorage {
	return &TraceStorage{db: db}
}

// Save stores a traceroute result with its hops.
func (s *TraceStorage) Save(trace *model.TraceResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert trace header
	result, err := tx.Exec(
		"INSERT INTO traces (target, timestamp) VALUES (?, ?)",
		trace.Target, trace.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to insert trace: %w", err)
	}

	traceID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get trace ID: %w", err)
	}
	trace.ID = traceID

	// Insert hops
	stmt, err := tx.Prepare(
		`INSERT INTO trace_hops (trace_id, hop_num, ip, hostname, latency_ms, lost) 
		 VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare hop statement: %w", err)
	}
	defer stmt.Close()

	for i := range trace.Hops {
		hop := &trace.Hops[i]
		lost := 0
		if hop.Lost {
			lost = 1
		}
		result, err := stmt.Exec(traceID, hop.HopNum, hop.IP, hop.Hostname, hop.LatencyMs, lost)
		if err != nil {
			return fmt.Errorf("failed to insert hop %d: %w", hop.HopNum, err)
		}
		hopID, _ := result.LastInsertId()
		hop.ID = hopID
		hop.TraceID = traceID
	}

	return tx.Commit()
}

// GetLatest returns the most recent trace for a target.
func (s *TraceStorage) GetLatest(target string) (*model.TraceResult, error) {
	query := `SELECT id, target, timestamp FROM traces 
			  WHERE target = ? ORDER BY timestamp DESC LIMIT 1`

	var trace model.TraceResult
	err := s.db.QueryRow(query, target).Scan(&trace.ID, &trace.Target, &trace.Timestamp)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest trace: %w", err)
	}

	hops, err := s.getHops(trace.ID)
	if err != nil {
		return nil, err
	}
	trace.Hops = hops

	return &trace, nil
}

// GetByID returns a trace by its ID.
func (s *TraceStorage) GetByID(id int64) (*model.TraceResult, error) {
	query := `SELECT id, target, timestamp FROM traces WHERE id = ?`

	var trace model.TraceResult
	err := s.db.QueryRow(query, id).Scan(&trace.ID, &trace.Target, &trace.Timestamp)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("trace not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get trace: %w", err)
	}

	hops, err := s.getHops(trace.ID)
	if err != nil {
		return nil, err
	}
	trace.Hops = hops

	return &trace, nil
}

func (s *TraceStorage) getHops(traceID int64) ([]model.TraceHop, error) {
	query := `SELECT id, trace_id, hop_num, ip, hostname, latency_ms, lost 
			  FROM trace_hops WHERE trace_id = ? ORDER BY hop_num`

	rows, err := s.db.Query(query, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query hops: %w", err)
	}
	defer rows.Close()

	var hops []model.TraceHop
	for rows.Next() {
		var hop model.TraceHop
		var lost int
		if err := rows.Scan(
			&hop.ID, &hop.TraceID, &hop.HopNum,
			&hop.IP, &hop.Hostname, &hop.LatencyMs, &lost); err != nil {
			return nil, fmt.Errorf("failed to scan hop: %w", err)
		}
		hop.Lost = lost == 1
		hops = append(hops, hop)
	}

	return hops, rows.Err()
}

// GetHistory returns traces for a target since a given time.
func (s *TraceStorage) GetHistory(target string, since time.Time) ([]model.TraceResult, error) {
	query := `SELECT id, target, timestamp FROM traces 
			  WHERE target = ? AND timestamp >= ? ORDER BY timestamp DESC LIMIT 20`

	rows, err := s.db.Query(query, target, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query traces: %w", err)
	}

	// Collect traces first without fetching hops
	var traces []model.TraceResult
	for rows.Next() {
		var trace model.TraceResult
		if err := rows.Scan(&trace.ID, &trace.Target, &trace.Timestamp); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan trace: %w", err)
		}
		traces = append(traces, trace)
	}
	rows.Close() // Close rows before making more queries

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Now fetch hops for each trace (after rows is closed)
	for i := range traces {
		hops, err := s.getHops(traces[i].ID)
		if err != nil {
			return nil, err
		}
		traces[i].Hops = hops
	}

	return traces, nil
}

// GetAllHistory returns all traces since a given time.
func (s *TraceStorage) GetAllHistory(since time.Time) ([]model.TraceResult, error) {
	query := `SELECT id, target, timestamp FROM traces 
			  WHERE timestamp >= ? ORDER BY timestamp DESC LIMIT 20`

	rows, err := s.db.Query(query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query all traces: %w", err)
	}

	// Collect traces first without fetching hops
	var traces []model.TraceResult
	for rows.Next() {
		var trace model.TraceResult
		if err := rows.Scan(&trace.ID, &trace.Target, &trace.Timestamp); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan trace: %w", err)
		}
		traces = append(traces, trace)
	}
	rows.Close() // Close rows before making more queries

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Now fetch hops for each trace (after rows is closed)
	for i := range traces {
		hops, err := s.getHops(traces[i].ID)
		if err != nil {
			return nil, err
		}
		traces[i].Hops = hops
	}

	return traces, nil
}

// GetTargets returns distinct trace targets.
func (s *TraceStorage) GetTargets() ([]string, error) {
	rows, err := s.db.Query("SELECT DISTINCT target FROM traces")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []string
	for rows.Next() {
		var target string
		if err := rows.Scan(&target); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	return targets, rows.Err()
}

// GetTracesForTarget returns all traces for a given target with basic info (ID, timestamp)
func (s *TraceStorage) GetTracesForTarget(target string, limit int) ([]model.TraceResult, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, target, timestamp FROM traces 
			  WHERE target = ? ORDER BY timestamp DESC LIMIT ?`

	rows, err := s.db.Query(query, target, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query traces for target: %w", err)
	}
	defer rows.Close()

	var traces []model.TraceResult
	for rows.Next() {
		var trace model.TraceResult
		if err := rows.Scan(&trace.ID, &trace.Target, &trace.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan trace: %w", err)
		}
		traces = append(traces, trace)
	}

	return traces, rows.Err()
}
