package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/user/netpulse/internal/model"
)

// IPStorage handles IP history persistence.
type IPStorage struct {
	db *DB
}

// NewIPStorage creates a new IP storage handler.
func NewIPStorage(db *DB) *IPStorage {
	return &IPStorage{db: db}
}

// Save stores an IP record.
func (s *IPStorage) Save(record *model.IPRecord) error {
	query := `INSERT INTO ip_history (ip, asn, isp, country, city, timestamp) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	
	result, err := s.db.Exec(query, 
		record.IP, record.ASN, record.ISP, 
		record.Country, record.City, record.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to insert IP record: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	record.ID = id
	
	return nil
}

// GetLatest returns the most recent IP record.
func (s *IPStorage) GetLatest() (*model.IPRecord, error) {
	query := `SELECT id, ip, asn, isp, country, city, timestamp 
			  FROM ip_history ORDER BY timestamp DESC LIMIT 1`
	
	var record model.IPRecord
	err := s.db.QueryRow(query).Scan(
		&record.ID, &record.IP, &record.ASN, 
		&record.ISP, &record.Country, &record.City, &record.Timestamp)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest IP: %w", err)
	}
	
	return &record, nil
}

// GetHistory returns IP history since a given time.
func (s *IPStorage) GetHistory(since time.Time) ([]model.IPRecord, error) {
	query := `SELECT id, ip, asn, isp, country, city, timestamp 
			  FROM ip_history WHERE timestamp >= ? ORDER BY timestamp DESC`
	
	rows, err := s.db.Query(query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query IP history: %w", err)
	}
	defer rows.Close()
	
	var records []model.IPRecord
	for rows.Next() {
		var record model.IPRecord
		if err := rows.Scan(
			&record.ID, &record.IP, &record.ASN,
			&record.ISP, &record.Country, &record.City, &record.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan IP record: %w", err)
		}
		records = append(records, record)
	}
	
	return records, rows.Err()
}

// GetChanges returns IP changes (distinct IPs) since a given time.
func (s *IPStorage) GetChanges(since time.Time) ([]model.IPRecord, error) {
	query := `SELECT id, ip, asn, isp, country, city, timestamp 
			  FROM ip_history 
			  WHERE timestamp >= ? 
			  GROUP BY ip 
			  ORDER BY timestamp DESC`
	
	rows, err := s.db.Query(query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query IP changes: %w", err)
	}
	defer rows.Close()
	
	var records []model.IPRecord
	for rows.Next() {
		var record model.IPRecord
		if err := rows.Scan(
			&record.ID, &record.IP, &record.ASN,
			&record.ISP, &record.Country, &record.City, &record.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan IP record: %w", err)
		}
		records = append(records, record)
	}
	
	return records, rows.Err()
}

// HasChanged checks if the IP has changed since the last record.
func (s *IPStorage) HasChanged(currentIP string) (bool, error) {
	latest, err := s.GetLatest()
	if err != nil {
		return false, err
	}
	if latest == nil {
		return true, nil // No previous record, consider it a change
	}
	return latest.IP != currentIP, nil
}

// Count returns the total number of IP records.
func (s *IPStorage) Count() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM ip_history").Scan(&count)
	return count, err
}

// CountSince returns the number of IP records since a given time.
func (s *IPStorage) CountSince(since time.Time) (int, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM ip_history WHERE timestamp >= ?", since).Scan(&count)
	return count, err
}

// GetDistinctCount returns the number of distinct IPs since a given time.
func (s *IPStorage) GetDistinctCount(since time.Time) (int, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(DISTINCT ip) FROM ip_history WHERE timestamp >= ?", since).Scan(&count)
	return count, err
}
