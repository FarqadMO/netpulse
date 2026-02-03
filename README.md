# ▶ NETPULSE_

<div align="center">

```
 _   _ _____ _____ ____  _   _ _     ____  _____ 
| \ | | ____|_   _|  _ \| | | | |   / ___|| ____|
|  \| |  _|   | | | |_) | | | | |   \___ \|  _|  
| |\  | |___  | | |  __/| |_| | |___ ___) | |___ 
|_| \_|_____| |_| |_|    \___/|_____|____/|_____|
```

**[ Network Reconnaissance & Monitoring System ]**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-00ff41?style=flat-square)](.)
[![License](https://img.shields.io/badge/License-MIT-ff00ff?style=flat-square)](LICENSE)

</div>

---

## `> OVERVIEW`

NetPulse is a **production-grade network monitoring daemon** that silently watches your network infrastructure. Deploy it once, and get continuous intelligence about:

| Module | Description |
|--------|-------------|
| 🌐 **IP Monitor** | Tracks public IP changes with ASN/ISP resolution |
| 🔀 **Traceroute** | Maps network paths to multiple targets |
| 📡 **Ping Sweep** | Discovers alive hosts on local subnets |
| 🔓 **Port Scanner** | Identifies open services on discovered hosts |
| 📊 **Analytics** | Route change detection & latency trends |

---

## `> FEATURES`

```
[✓] Cross-platform daemon (Linux/macOS/Windows)
[✓] Adaptive monitoring intervals
[✓] Real-time TUI dashboard with Lipgloss
[✓] Web dashboard with 5 theme options
[✓] Network topology visualization (Mermaid)
[✓] Latency trend graphs (Chart.js)  
[✓] Route anomaly detection
[✓] Markdown reports with network diagrams
[✓] SQLite persistence (zero-config)
[✓] RESTful API for integration
```

---

## `> INSTALLATION`

### Build from Source

```bash
# Clone
git clone https://github.com/user/netpulse.git
cd netpulse

# Build
go build -o netpulse ./cmd/netpulse

# (Optional) Install globally
sudo mv netpulse /usr/local/bin/
```

### Cross-Compile

```bash
GOOS=linux   GOARCH=amd64 go build -o netpulse-linux   ./cmd/netpulse
GOOS=darwin  GOARCH=arm64 go build -o netpulse-macos   ./cmd/netpulse
GOOS=windows GOARCH=amd64 go build -o netpulse.exe     ./cmd/netpulse
```

---

## `> QUICK START`

```bash
# Start monitoring daemon
./netpulse start

# Start with web dashboard
./netpulse start --with-web

# Check status
./netpulse status

# Launch TUI
./netpulse ui

# Standalone web server
./netpulse web --port 8080

# Generate report
./netpulse report --last 24h

# Stop daemon
./netpulse stop
```

---

## `> COMMANDS`

| Command | Description |
|---------|-------------|
| `start` | Start daemon (`-f` foreground, `--with-web` include dashboard) |
| `stop` | Graceful shutdown |
| `status` | Show daemon status & network statistics |
| `ui` | Interactive TUI dashboard |
| `web` | Launch web dashboard (`--port N`) |
| `report` | Generate Markdown report (`--last 24h/7d/30d`) |

---

## `> WEB DASHBOARD`

Access at `http://localhost:8080` after starting with `--with-web`

### Tabs
- **Overview** - Current IP, daemon status, discovered hosts
- **Topology** - Mermaid network path visualization
- **Traces** - Filterable traceroute history
- **Latency** - Time-series latency graphs
- **Anomalies** - Route change detection

### Themes
Switch between 5 color schemes:

| Theme | Colors |
|-------|--------|
| 🟢 Hacker | Matrix green |
| 💜 Cyberpunk | Pink & cyan |
| 🔴 Blood | Deep red |
| 🟠 Amber | Retro orange |
| 🔵 Ocean | Cool blue |

---

## `> API ENDPOINTS`

| Endpoint | Description |
|----------|-------------|
| `GET /api/ip` | Current public IP |
| `GET /api/ip/history` | IP change history |
| `GET /api/traces` | Traceroute results |
| `GET /api/hosts` | Discovered hosts |
| `GET /api/status` | Daemon status |
| `GET /api/analytics/topology` | Network graph data |
| `GET /api/analytics/latency` | Latency time series |
| `GET /api/analytics/anomalies` | Route changes |
| `GET /report` | Download Markdown report |

---

## `> CONFIGURATION`

Create `~/.netpulse/config.yaml`:

```yaml
# Logging
log_level: info

# Probe intervals
ip_check_interval: 5m
trace_interval: 15m
ping_sweep_interval: 30m
port_scan_interval: 1h

# Traceroute targets
trace_targets:
  - 8.8.8.8          # Google DNS
  - 1.1.1.1          # Cloudflare
  - 208.67.222.222   # OpenDNS

# Network scanning
sweep_subnet: 192.168.1.0/24
sweep_concurrency: 50

# Port scanning
scan_ports: [22, 80, 443, 3389, 8080]
scan_concurrency: 20
```

---

## `> DATA STORAGE`

All data persists in `~/.netpulse/`:

```
~/.netpulse/
├── config.yaml      # Configuration
├── netpulse.db      # SQLite database
├── netpulse.log     # Daemon logs
├── netpulse.pid     # Process ID
└── reports/         # Generated reports
```

### Database Schema

| Table | Contents |
|-------|----------|
| `ip_history` | Public IP records with ASN/ISP |
| `traces` | Traceroute sessions |
| `trace_hops` | Individual hops |
| `scan_hosts` | Discovered hosts |
| `scan_ports` | Open ports |

---

## `> ARCHITECTURE`

```
┌─────────────────────────────────────────────────────┐
│                    netpulse CLI                      │
├──────────┬──────────┬──────────┬──────────┬─────────┤
│  start   │  status  │   ui     │   web    │ report  │
└────┬─────┴────┬─────┴────┬─────┴────┬─────┴────┬────┘
     │          │          │          │          │
     ▼          ▼          ▼          ▼          ▼
┌─────────┐ ┌────────┐ ┌───────┐ ┌────────┐ ┌────────┐
│ daemon  │ │storage │ │  tui  │ │  web   │ │ report │
│scheduler│ │ SQLite │ │bubble │ │ html   │ │mermaid │
└────┬────┘ └───┬────┘ └───────┘ └────────┘ └────────┘
     │          │
     ▼          ▼
┌────────────────────────────────────────────────────┐
│                     probes                          │
├────────────┬────────────┬────────────┬─────────────┤
│  ip.go     │traceroute  │  ping.go   │ portscan.go │
│ monitoring │   .go      │ sweep      │             │
└────────────┴────────────┴────────────┴─────────────┘
```

---

## `> LICENSE`

MIT License - See [LICENSE](LICENSE) for details

---

<div align="center">

**[ SYSTEM ONLINE ]**

`Built with 💚 in Go`

</div>
