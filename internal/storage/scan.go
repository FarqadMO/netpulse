package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/user/netpulse/internal/model"
)

// ScanStorage handles host and port scan persistence.
type ScanStorage struct {
	db *DB
}

// NewScanStorage creates a new scan storage handler.
func NewScanStorage(db *DB) *ScanStorage {
	return &ScanStorage{db: db}
}

// SaveHost stores or updates a discovered host.
func (s *ScanStorage) SaveHost(host *model.ScanHost) error {
	query := `INSERT INTO scan_hosts (ip, hostname, alive, latency_ms, last_seen) 
			  VALUES (?, ?, ?, ?, ?)
			  ON CONFLICT(ip) DO UPDATE SET 
			  hostname = excluded.hostname,
			  alive = excluded.alive,
			  latency_ms = excluded.latency_ms,
			  last_seen = excluded.last_seen`
	
	result, err := s.db.Exec(query, 
		host.IP, host.Hostname, host.Alive, host.LatencyMs, host.LastSeen)
	if err != nil {
		return fmt.Errorf("failed to save host: %w", err)
	}
	
	if host.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil && id > 0 {
			host.ID = id
		} else {
			// Get existing ID
			s.db.QueryRow("SELECT id FROM scan_hosts WHERE ip = ?", host.IP).Scan(&host.ID)
		}
	}
	
	return nil
}

// SavePort stores or updates a port scan result.
func (s *ScanStorage) SavePort(port *model.ScanPort) error {
	query := `INSERT INTO scan_ports (host_id, port, protocol, service, state, banner, last_seen) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)
			  ON CONFLICT(host_id, port, protocol) DO UPDATE SET 
			  service = excluded.service,
			  state = excluded.state,
			  banner = excluded.banner,
			  last_seen = excluded.last_seen`
	
	result, err := s.db.Exec(query,
		port.HostID, port.Port, port.Protocol, 
		port.Service, port.State, port.Banner, port.LastSeen)
	if err != nil {
		return fmt.Errorf("failed to save port: %w", err)
	}
	
	if port.ID == 0 {
		id, _ := result.LastInsertId()
		port.ID = id
	}
	
	return nil
}

// GetHost returns a host by IP.
func (s *ScanStorage) GetHost(ip string) (*model.ScanHost, error) {
	query := `SELECT id, ip, hostname, alive, latency_ms, last_seen, display_name, tags, icon 
			  FROM scan_hosts WHERE ip = ?`
	
	var host model.ScanHost
	var displayName, tags, icon sql.NullString

	err := s.db.QueryRow(query, ip).Scan(
		&host.ID, &host.IP, &host.Hostname, 
		&host.Alive, &host.LatencyMs, &host.LastSeen,
		&displayName, &tags, &icon)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}
	
	if displayName.Valid {
		host.DisplayName = displayName.String
	}
	if icon.Valid {
		host.Icon = icon.String
	}
	if tags.Valid && tags.String != "" {
		host.Tags = strings.Split(tags.String, ",")
	}
	
	return &host, nil
}

// GetAliveHosts returns all alive hosts.
func (s *ScanStorage) GetAliveHosts() ([]model.ScanHost, error) {
	query := `SELECT id, ip, hostname, alive, latency_ms, last_seen, display_name, tags, icon 
			  FROM scan_hosts WHERE alive = 1 ORDER BY ip`
	
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query hosts: %w", err)
	}
	defer rows.Close()
	
	var hosts []model.ScanHost
	for rows.Next() {
		var h model.ScanHost
		var displayName, tags, icon sql.NullString
		
		if err := rows.Scan(&h.ID, &h.IP, &h.Hostname, &h.Alive, &h.LatencyMs, &h.LastSeen, &displayName, &tags, &icon); err != nil {
			continue
		}
		
		if displayName.Valid {
			h.DisplayName = displayName.String
		}
		if icon.Valid {
			h.Icon = icon.String
		}
		if tags.Valid && tags.String != "" {
			h.Tags = strings.Split(tags.String, ",")
		}

		hosts = append(hosts, h)
	}
	
	return hosts, rows.Err()
}

// GetHostPorts returns open ports for a host.
func (s *ScanStorage) GetHostPorts(hostID int64) ([]model.ScanPort, error) {
	query := `SELECT id, host_id, port, protocol, service, state, banner, last_seen 
			  FROM scan_ports WHERE host_id = ? AND state = 'open' ORDER BY port`
	
	rows, err := s.db.Query(query, hostID)
	if err != nil {
		return nil, fmt.Errorf("failed to query ports: %w", err)
	}
	defer rows.Close()
	
	var ports []model.ScanPort
	for rows.Next() {
		var port model.ScanPort
		if err := rows.Scan(
			&port.ID, &port.HostID, &port.Port, &port.Protocol,
			&port.Service, &port.State, &port.Banner, &port.LastSeen); err != nil {
			return nil, fmt.Errorf("failed to scan port: %w", err)
		}
		ports = append(ports, port)
	}
	
	return ports, rows.Err()
}

// GetRecentlyDiscovered returns hosts discovered since a given time.
func (s *ScanStorage) GetRecentlyDiscovered(since time.Time) ([]model.ScanHost, error) {
	query := `SELECT id, ip, hostname, alive, latency_ms, last_seen 
			  FROM scan_hosts WHERE last_seen >= ? ORDER BY last_seen DESC`
	
	rows, err := s.db.Query(query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent hosts: %w", err)
	}
	defer rows.Close()
	
	var hosts []model.ScanHost
	for rows.Next() {
		var host model.ScanHost
		if err := rows.Scan(
			&host.ID, &host.IP, &host.Hostname,
			&host.Alive, &host.LatencyMs, &host.LastSeen); err != nil {
			return nil, fmt.Errorf("failed to scan host: %w", err)
		}
		hosts = append(hosts, host)
	}
	
	return hosts, rows.Err()
}

// GetNewPorts returns ports discovered since a given time.
func (s *ScanStorage) GetNewPorts(since time.Time) ([]model.ScanPort, error) {
	query := `SELECT id, host_id, port, protocol, service, state, banner, last_seen 
			  FROM scan_ports WHERE last_seen >= ? ORDER BY last_seen DESC`
	
	rows, err := s.db.Query(query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query new ports: %w", err)
	}
	defer rows.Close()
	
	var ports []model.ScanPort
	for rows.Next() {
		var port model.ScanPort
		if err := rows.Scan(
			&port.ID, &port.HostID, &port.Port, &port.Protocol,
			&port.Service, &port.State, &port.Banner, &port.LastSeen); err != nil {
			return nil, fmt.Errorf("failed to scan port: %w", err)
		}
		ports = append(ports, port)
	}
	
	return ports, rows.Err()
}

// CountAliveHosts returns the number of alive hosts.
func (s *ScanStorage) CountAliveHosts() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM scan_hosts WHERE alive = 1").Scan(&count)
	return count, err
}

// CountOpenPorts returns the number of open ports.
func (s *ScanStorage) CountOpenPorts() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM scan_ports WHERE state = 'open'").Scan(&count)
	return count, err
}

// UpdateHostMetadata updates the user-defined metadata for a host.
func (s *ScanStorage) UpdateHostMetadata(id int64, displayName string, tags []string, icon string) error {
	query := `UPDATE scan_hosts SET display_name = ?, tags = ?, icon = ? WHERE id = ?`
	tagStr := strings.Join(tags, ",")
	
	_, err := s.db.Exec(query, displayName, tagStr, icon, id)
	if err != nil {
		return fmt.Errorf("failed to update host metadata: %w", err)
	}
	return nil
}
