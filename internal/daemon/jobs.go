package daemon

import (
	"context"
	"time"

	"github.com/user/netpulse/internal/model"
	"github.com/user/netpulse/internal/probes"
	"github.com/user/netpulse/internal/storage"
	"github.com/user/netpulse/internal/util"
)

// registerJobs registers all probe jobs with the scheduler.
func (d *Daemon) registerJobs() {
	// IP Check Job
	d.scheduler.AddJob(&Job{
		Name:     "ip_check",
		Interval: d.config.IPCheckInterval,
		Run:      d.runIPCheck,
	})
	
	// Traceroute Job
	d.scheduler.AddJob(&Job{
		Name:     "traceroute",
		Interval: d.config.TraceInterval,
		Run:      d.runTraceroute,
	})
	
	// Ping Sweep Job
	d.scheduler.AddJob(&Job{
		Name:     "ping_sweep",
		Interval: d.config.PingSweepInterval,
		Run:      d.runPingSweep,
	})
	
	// Port Scan Job
	d.scheduler.AddJob(&Job{
		Name:     "port_scan",
		Interval: d.config.PortScanInterval,
		Run:      d.runPortScan,
	})
}

func (d *Daemon) runIPCheck(ctx context.Context) error {
	probe := probes.NewIPProbe()
	
	// Get public IP
	ip, err := probe.GetPublicIP(ctx)
	if err != nil {
		return err
	}
	
	util.Info("Detected public IP: %s", ip)
	
	// Get ASN info
	asnInfo, err := probes.GetASNInfo(ctx, ip)
	if err != nil {
		util.Warn("Failed to get ASN info: %v", err)
	}
	
	// Create record
	record := &model.IPRecord{
		IP:        ip,
		Timestamp: time.Now(),
	}
	if asnInfo != nil {
		record.ASN = asnInfo.ASN
		record.ISP = asnInfo.ISP
		record.Country = asnInfo.Country
		record.City = asnInfo.City
	}
	
	// Check if IP changed
	ipStorage := storage.NewIPStorage(d.db)
	changed, err := ipStorage.HasChanged(ip)
	if err != nil {
		return err
	}
	
	// Save record
	if err := ipStorage.Save(record); err != nil {
		return err
	}
	
	if changed {
		util.Info("IP changed to: %s (%s)", ip, record.ISP)
	}
	
	return nil
}

func (d *Daemon) runTraceroute(ctx context.Context) error {
	probe := probes.NewTracerouteProbe()
	traceStorage := storage.NewTraceStorage(d.db)
	
	for _, target := range d.config.TraceTargets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		util.Debug("Tracing route to %s", target)
		
		result, err := probe.Trace(ctx, target)
		if err != nil {
			util.Warn("Traceroute to %s failed: %v", target, err)
			continue
		}
		
		if err := traceStorage.Save(result); err != nil {
			util.Warn("Failed to save trace for %s: %v", target, err)
			continue
		}
		
		util.Info("Traceroute to %s: %d hops", target, len(result.Hops))
	}
	
	return nil
}

func (d *Daemon) runPingSweep(ctx context.Context) error {
	if d.config.SweepSubnet == "" {
		util.Debug("Ping sweep disabled (no subnet configured)")
		return nil
	}
	
	probe := probes.NewPingProbe(d.config.SweepConcurrency, d.config.SweepTimeout)
	scanStorage := storage.NewScanStorage(d.db)
	
	util.Debug("Starting ping sweep of %s", d.config.SweepSubnet)
	
	hosts, err := probe.SweepSubnet(ctx, d.config.SweepSubnet)
	if err != nil {
		return err
	}
	
	aliveCount := 0
	for i := range hosts {
		host := &hosts[i]
		if err := scanStorage.SaveHost(host); err != nil {
			util.Warn("Failed to save host %s: %v", host.IP, err)
		}
		if host.Alive {
			aliveCount++
		}
	}
	
	util.Info("Ping sweep complete: %d/%d hosts alive", aliveCount, len(hosts))
	
	return nil
}

func (d *Daemon) runPortScan(ctx context.Context) error {
	scanStorage := storage.NewScanStorage(d.db)
	
	// Get alive hosts
	hosts, err := scanStorage.GetAliveHosts()
	if err != nil {
		return err
	}
	
	if len(hosts) == 0 {
		util.Debug("No alive hosts to scan")
		return nil
	}
	
	scanner := probes.NewPortScanner(
		d.config.ScanConcurrency,
		d.config.ScanTimeout,
		d.config.ScanPorts,
	)
	
	util.Debug("Starting port scan on %d hosts", len(hosts))
	
	totalPorts := 0
	for _, host := range hosts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		ports, err := scanner.ScanHost(ctx, host.IP)
		if err != nil {
			util.Warn("Port scan on %s failed: %v", host.IP, err)
			continue
		}
		
		for i := range ports {
			port := &ports[i]
			port.HostID = host.ID
			if err := scanStorage.SavePort(port); err != nil {
				util.Warn("Failed to save port %d on %s: %v", port.Port, host.IP, err)
			}
		}
		
		totalPorts += len(ports)
	}
	
	util.Info("Port scan complete: %d open ports found", totalPorts)
	
	return nil
}
