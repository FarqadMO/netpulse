// Package tui provides a terminal user interface.
package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/user/netpulse/internal/storage"
	"github.com/user/netpulse/internal/util"
)

// App is the main TUI application.
type App struct {
	db     *storage.DB
	config *util.Config
}

// NewApp creates a new TUI application.
func NewApp(db *storage.DB, cfg *util.Config) *App {
	return &App{
		db:     db,
		config: cfg,
	}
}

// Run starts the TUI application.
func (a *App) Run() error {
	p := tea.NewProgram(newModel(a.db, a.config), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// model is the main bubbletea model.
type model struct {
	db        *storage.DB
	config    *util.Config
	dashboard *Dashboard
	spinner   spinner.Model
	ready     bool
	width     int
	height    int
	err       error
}

func newModel(db *storage.DB, cfg *util.Config) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	
	return model{
		db:      db,
		config:  cfg,
		spinner: s,
	}
}

// Init initializes the model.
func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadData(m.db),
	)
}

// Update handles messages.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			return m, loadData(m.db)
		}
	
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.dashboard != nil {
			m.dashboard.SetSize(msg.Width, msg.Height)
		}
	
	case dataMsg:
		m.ready = true
		m.dashboard = NewDashboard(msg, m.width, m.height)
	
	case errMsg:
		m.err = msg.err
	
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	
	return m, nil
}

// View renders the UI.
func (m model) View() string {
	if m.err != nil {
		return ErrorStyle.Render("Error: " + m.err.Error())
	}
	
	if !m.ready {
		return LoadingStyle.Render(m.spinner.View() + " Loading...")
	}
	
	return m.dashboard.View()
}

// Messages
type dataMsg struct {
	Data *DashboardData
}

type errMsg struct {
	err error
}

func loadData(db *storage.DB) tea.Cmd {
	return func() tea.Msg {
		data, err := fetchDashboardData(db)
		if err != nil {
			return errMsg{err}
		}
		return dataMsg{Data: data}
	}
}

func fetchDashboardData(db *storage.DB) (*DashboardData, error) {
	data := &DashboardData{}
	
	// Get latest IP
	ipStorage := storage.NewIPStorage(db)
	if latest, err := ipStorage.GetLatest(); err == nil && latest != nil {
		data.CurrentIP = latest.IP
		data.ISP = latest.ISP
		data.ASN = latest.ASN
		data.LastIPCheck = latest.Timestamp.Format("15:04:05")
	}
	
	// Get stats
	if count, err := ipStorage.Count(); err == nil {
		data.IPRecordCount = count
	}
	
	scanStorage := storage.NewScanStorage(db)
	if count, err := scanStorage.CountAliveHosts(); err == nil {
		data.AliveHostCount = count
	}
	if count, err := scanStorage.CountOpenPorts(); err == nil {
		data.OpenPortCount = count
	}
	
	// Get alive hosts
	if hosts, err := scanStorage.GetAliveHosts(); err == nil {
		for _, h := range hosts {
			data.Hosts = append(data.Hosts, HostInfo{
				IP:       h.IP,
				Hostname: h.Hostname,
				Latency:  h.LatencyMs,
			})
		}
	}
	
	return data, nil
}
