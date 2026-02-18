package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	// Removed unused errorStyle to fix lint

	menuStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1).
			Width(40)

	headerStyle = lipgloss.NewStyle().
			MarginBottom(1)
)

type ViewMode int

const (
	DashboardMode ViewMode = iota
	TrafficMode
)

type LogMsg string
type MetricsUpdateMsg struct {
	TotalRequests int64
	AvgLatency    float64
}
type ConfigReloadMsg struct{}

type TrafficPacket struct {
	Timestamp time.Time
	Method    string
	Path      string
	Backend   string
	Status    int
	Latency   int64
	TenantID  string
}

type Model struct {
	Uptime        time.Time
	TotalRequests int64
	AvgLatency    float64
	Logs          []string
	Traffic       []TrafficPacket
	Mode          ViewMode
	ShowMenu      bool
	Viewport      viewport.Model
	TrafficTable  table.Model
	Ready         bool
	TermWidth     int
	TermHeight    int
	OnReload      func()
}

func NewModel(onReload func()) Model {
	columns := []table.Column{
		{Title: "Time", Width: 10},
		{Title: "Method", Width: 8},
		{Title: "Status", Width: 8},
		{Title: "Path", Width: 20},
		{Title: "Backend", Width: 15},
		{Title: "Lat", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return Model{
		Uptime:       time.Now(),
		Logs:         make([]string, 0),
		Traffic:      make([]TrafficPacket, 0),
		Mode:         DashboardMode,
		OnReload:     onReload,
		TrafficTable: t,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "f2":
			m.ShowMenu = !m.ShowMenu
		case "tab":
			if m.Mode == DashboardMode {
				m.Mode = TrafficMode
			} else {
				m.Mode = DashboardMode
			}
		case "r":
			if m.ShowMenu {
				if m.OnReload != nil {
					m.OnReload()
					m.Logs = append(m.Logs, infoStyle.Render("Configuration reload requested via TUI"))
					m.ShowMenu = false
				}
			}
		case "t":
			if m.ShowMenu {
				m.Mode = TrafficMode
				m.ShowMenu = false
			}
		case "d":
			if m.ShowMenu {
				m.Mode = DashboardMode
				m.ShowMenu = false
			}
		case "c":
			if m.ShowMenu {
				m.TotalRequests = 0
				m.AvgLatency = 0
				m.Traffic = nil
				m.TrafficTable.SetRows(nil)
				m.Logs = append(m.Logs, infoStyle.Render("Metrics and Traffic cleared"))
				m.ShowMenu = false
			}
		}

	case tea.WindowSizeMsg:
		m.TermWidth = msg.Width
		m.TermHeight = msg.Height
		headerHeight := 10
		if !m.Ready {
			m.Viewport = viewport.New(msg.Width, msg.Height-headerHeight)
			m.Viewport.SetContent(strings.Join(m.Logs, "\n"))
			m.Ready = true
		} else {
			m.Viewport.Width = msg.Width
			m.Viewport.Height = msg.Height - headerHeight
		}
		m.TrafficTable.SetHeight(msg.Height - headerHeight - 2)

	case LogMsg:
		m.Logs = append(m.Logs, string(msg))
		if len(m.Logs) > 500 {
			m.Logs = m.Logs[len(m.Logs)-500:]
		}
		m.Viewport.SetContent(strings.Join(m.Logs, "\n"))
		m.Viewport.GotoBottom()

	case TrafficPacket:
		m.Traffic = append(m.Traffic, msg)
		if len(m.Traffic) > 100 {
			m.Traffic = m.Traffic[len(m.Traffic)-100:]
		}
		rows := make([]table.Row, len(m.Traffic))
		for i, p := range m.Traffic {
			rows[i] = table.Row{
				p.Timestamp.Format("15:04:05"),
				p.Method,
				fmt.Sprintf("%d", p.Status),
				p.Path,
				p.Backend,
				fmt.Sprintf("%dms", p.Latency),
			}
		}
		m.TrafficTable.SetRows(rows)

	case MetricsUpdateMsg:
		m.TotalRequests = msg.TotalRequests
		m.AvgLatency = msg.AvgLatency
	}

	if m.Mode == TrafficMode {
		m.TrafficTable, cmd = m.TrafficTable.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.Viewport, cmd = m.Viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.Ready {
		return "\n  Initializing APIM Monitor..."
	}

	uptime := time.Since(m.Uptime).Round(time.Second)
	modeStr := "DASHBOARD"
	if m.Mode == TrafficMode {
		modeStr = "TRAFFIC MONITOR (WIRESHARK-STYLE)"
	}

	header := headerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("APIM CORE MONITOR")+" "+infoStyle.Render(modeStr),
			fmt.Sprintf("Uptime: %s", infoStyle.Render(uptime.String())),
			fmt.Sprintf("Requests: %s", infoStyle.Render(fmt.Sprintf("%d", m.TotalRequests))),
			fmt.Sprintf("Avg Latency: %s", infoStyle.Render(fmt.Sprintf("%.2fms", m.AvgLatency))),
			"",
			titleStyle.Render(fmt.Sprintf("VIEW: [%s]", modeStr)),
		),
	)

	var body string
	if m.Mode == DashboardMode {
		body = m.Viewport.View()
	} else {
		body = m.TrafficTable.View()
	}

	view := header + "\n" + body

	if m.ShowMenu {
		menuContent := lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("F2 MENU"),
			"",
			"[D] Switch to Dashboard",
			"[T] Switch to Traffic Monitor",
			"[R] Reload Configuration",
			"[C] Clear Metrics & Traffic",
			"[Q] Quit Monitor",
			"",
			"Press F2 to close",
		)
		menu := menuStyle.Render(menuContent)

		return m.overlay(view, menu)
	}

	return view
}

func (m Model) overlay(base string, overlay string) string {
	return base + "\n\n" + overlay
}
