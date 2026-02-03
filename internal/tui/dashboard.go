package tui

import (
	"fmt"
	"strings"
)

// DashboardData holds data for the dashboard view.
type DashboardData struct {
	CurrentIP      string
	ISP            string
	ASN            string
	LastIPCheck    string
	IPRecordCount  int
	AliveHostCount int
	OpenPortCount  int
	Hosts          []HostInfo
	Anomalies      []AnomalyInfo
}

// HostInfo represents host information for display.
type HostInfo struct {
	IP       string
	Hostname string
	Latency  float64
}

// AnomalyInfo represents anomaly information for display.
type AnomalyInfo struct {
	Type        string
	Description string
	Time        string
}

// Dashboard is the main dashboard view.
type Dashboard struct {
	data   *DashboardData
	width  int
	height int
}

// NewDashboard creates a new dashboard.
func NewDashboard(msg dataMsg, width, height int) *Dashboard {
	return &Dashboard{
		data:   msg.Data,
		width:  width,
		height: height,
	}
}

// SetSize updates the dashboard size.
func (d *Dashboard) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// View renders the dashboard.
func (d *Dashboard) View() string {
	var sb strings.Builder
	
	// Header
	header := HeaderStyle.Width(d.width).Render("🌐 NetPulse Dashboard")
	sb.WriteString(header)
	sb.WriteString("\n\n")
	
	// IP Status Section
	ipSection := d.renderIPSection()
	sb.WriteString(ipSection)
	sb.WriteString("\n")
	
	// Stats Section
	statsSection := d.renderStatsSection()
	sb.WriteString(statsSection)
	sb.WriteString("\n")
	
	// Hosts Section
	hostsSection := d.renderHostsSection()
	sb.WriteString(hostsSection)
	sb.WriteString("\n")
	
	// Help
	help := HelpStyle.Render("Press 'r' to refresh • 'q' to quit")
	sb.WriteString(help)
	
	return sb.String()
}

func (d *Dashboard) renderIPSection() string {
	sectionWidth := d.width - 4
	if sectionWidth < 40 {
		sectionWidth = 40
	}
	
	content := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s",
		LabelStyle.Render("IP:"),
		ValueStyle.Render(d.data.CurrentIP),
		LabelStyle.Render("ISP:"),
		ValueStyle.Render(d.data.ISP),
		LabelStyle.Render("ASN:"),
		ValueStyle.Render(d.data.ASN),
		LabelStyle.Render("Last Check:"),
		ValueStyle.Render(d.data.LastIPCheck),
	)
	
	return SectionStyle.Width(sectionWidth).Render(
		SectionTitleStyle.Render("📡 IP Status") + "\n" + content)
}

func (d *Dashboard) renderStatsSection() string {
	sectionWidth := d.width - 4
	if sectionWidth < 40 {
		sectionWidth = 40
	}
	
	content := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s",
		LabelStyle.Render("IP Records:"),
		ValueStyle.Render(fmt.Sprintf("%d", d.data.IPRecordCount)),
		LabelStyle.Render("Alive Hosts:"),
		ValueStyle.Render(fmt.Sprintf("%d", d.data.AliveHostCount)),
		LabelStyle.Render("Open Ports:"),
		ValueStyle.Render(fmt.Sprintf("%d", d.data.OpenPortCount)),
	)
	
	return SectionStyle.Width(sectionWidth).Render(
		SectionTitleStyle.Render("📊 Statistics") + "\n" + content)
}

func (d *Dashboard) renderHostsSection() string {
	sectionWidth := d.width - 4
	if sectionWidth < 40 {
		sectionWidth = 40
	}
	
	if len(d.data.Hosts) == 0 {
		content := DimStyle.Render("No hosts discovered yet")
		return SectionStyle.Width(sectionWidth).Render(
			SectionTitleStyle.Render("🖥️ Discovered Hosts") + "\n" + content)
	}
	
	var rows []string
	rows = append(rows, fmt.Sprintf("%-16s %-20s %s", "IP", "Hostname", "Latency"))
	rows = append(rows, strings.Repeat("─", 50))
	
	maxHosts := 10
	if len(d.data.Hosts) < maxHosts {
		maxHosts = len(d.data.Hosts)
	}
	
	for i := 0; i < maxHosts; i++ {
		h := d.data.Hosts[i]
		hostname := h.Hostname
		if hostname == "" {
			hostname = "-"
		}
		if len(hostname) > 18 {
			hostname = hostname[:15] + "..."
		}
		rows = append(rows, fmt.Sprintf("%-16s %-20s %.1f ms", h.IP, hostname, h.Latency))
	}
	
	if len(d.data.Hosts) > maxHosts {
		rows = append(rows, DimStyle.Render(fmt.Sprintf("... and %d more", len(d.data.Hosts)-maxHosts)))
	}
	
	content := strings.Join(rows, "\n")
	return SectionStyle.Width(sectionWidth).Render(
		SectionTitleStyle.Render("🖥️ Discovered Hosts") + "\n" + content)
}
