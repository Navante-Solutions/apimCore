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

	headerLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#353535")).
			Padding(0, 1)

	footerKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#AAAAAA")).
			Padding(0, 1)

	footerActionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#7D56F4")).
				Padding(0, 1)

	menuStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1).
			Width(40)

	headerStyle = lipgloss.NewStyle().
			MarginBottom(1).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("240"))
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
	Uptime         time.Time
	TotalRequests  int64
	AvgLatency     float64
	Logs           []string
	Traffic        []TrafficPacket
	Mode           ViewMode
	ShowMenu       bool
	Viewport       viewport.Model
	TrafficTable   table.Model
	Ready          bool
	TermWidth      int
	TermHeight     int
	OnReload       func()
	SelectedPacket *TrafficPacket
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
		case "f10", "ctrl+c":
			return m, tea.Quit
		case "f2":
			m.ShowMenu = !m.ShowMenu
		case "f3":
			m.Mode = DashboardMode
			m.TrafficTable.Blur()
			m.ShowMenu = false
		case "f4":
			m.Mode = TrafficMode
			m.TrafficTable.Focus()
			m.ShowMenu = false
		case "tab":
			if m.Mode == DashboardMode {
				m.Mode = TrafficMode
				m.TrafficTable.Focus()
			} else {
				m.Mode = DashboardMode
				m.TrafficTable.Blur()
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
				m.TrafficTable.Focus()
				m.ShowMenu = false
			}
		case "d":
			if m.ShowMenu {
				m.Mode = DashboardMode
				m.TrafficTable.Blur()
				m.ShowMenu = false
			}
		case "enter":
			if m.Mode == TrafficMode && !m.ShowMenu {
				curr := m.TrafficTable.SelectedRow()
				if len(curr) > 0 {
					idx := m.TrafficTable.Cursor()
					if idx >= 0 && idx < len(m.Traffic) {
						m.SelectedPacket = &m.Traffic[idx]
					}
				}
			}
		case "esc":
			if m.SelectedPacket != nil {
				m.SelectedPacket = nil
			} else if m.ShowMenu {
				m.ShowMenu = false
			}
		case "c":
			if m.ShowMenu {
				m.TotalRequests = 0
				m.AvgLatency = 0
				m.Traffic = nil
				m.TrafficTable.SetRows(nil)
				m.SelectedPacket = nil
				m.Logs = append(m.Logs, infoStyle.Render("Metrics and Traffic cleared"))
				m.ShowMenu = false
			}
		}

	case tea.WindowSizeMsg:
		m.TermWidth = msg.Width
		m.TermHeight = msg.Height
		headerHeight := 8
		footerHeight := 1

		if !m.Ready {
			m.Viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.Viewport.SetContent(strings.Join(m.Logs, "\n"))
			m.Ready = true
		} else {
			m.Viewport.Width = msg.Width
			m.Viewport.Height = msg.Height - headerHeight - footerHeight
		}
		m.TrafficTable.SetWidth(msg.Width)
		m.TrafficTable.SetHeight(msg.Height - headerHeight - footerHeight - 2)

	case LogMsg:
		m.Logs = append(m.Logs, string(msg))
		if len(m.Logs) > 1000 {
			m.Logs = m.Logs[len(m.Logs)-1000:]
		}
		m.Viewport.SetContent(strings.Join(m.Logs, "\n"))
		if m.Viewport.AtBottom() {
			m.Viewport.GotoBottom()
		}

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
		modeStr = "TRAFFIC MONITOR"
	}

	// HTOP-like Header
	header := headerStyle.Width(m.TermWidth).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(m.TermWidth/2).Render(
				lipgloss.JoinVertical(lipgloss.Left,
					titleStyle.Render("APIM CORE"),
					headerLabelStyle.Render("Uptime:  ")+infoStyle.Render(uptime.String()),
					headerLabelStyle.Render("Requests: ")+infoStyle.Render(fmt.Sprintf("%d", m.TotalRequests)),
					headerLabelStyle.Render("Latency:  ")+infoStyle.Render(fmt.Sprintf("%.2fms", m.AvgLatency)),
				),
			),
			lipgloss.NewStyle().Width(m.TermWidth/2).Render(
				lipgloss.JoinVertical(lipgloss.Left,
					headerLabelStyle.Render("Mode:     ")+statusStyle.Render(modeStr),
					headerLabelStyle.Render("Status:   ")+infoStyle.Render("RUNNING"),
					"",
				),
			),
		),
	)

	var body string
	if m.Mode == DashboardMode {
		body = m.Viewport.View()
	} else {
		body = m.TrafficTable.View()
		if m.SelectedPacket != nil {
			details := lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(1).
				Width(m.TermWidth - 4).
				Render(fmt.Sprintf(
					"PACKET DETAILS\n\nMethod: %s\nPath: %s\nBackend: %s\nStatus: %d\nLatency: %dms\nTenant: %s\nTime: %s",
					m.SelectedPacket.Method,
					m.SelectedPacket.Path,
					m.SelectedPacket.Backend,
					m.SelectedPacket.Status,
					m.SelectedPacket.Latency,
					m.SelectedPacket.TenantID,
					m.SelectedPacket.Timestamp.Format(time.RFC3339),
				))
			body = lipgloss.JoinVertical(lipgloss.Left, body, details)
		}
	}

	// HTOP-like Footer
	footerParts := []string{
		footerKeyStyle.Render("F2") + footerActionStyle.Render("Setup"),
		footerKeyStyle.Render("F3") + footerActionStyle.Render("Dash"),
		footerKeyStyle.Render("F4") + footerActionStyle.Render("Traffic"),
		footerKeyStyle.Render("Tab") + footerActionStyle.Render("NextView"),
		footerKeyStyle.Render("F10") + footerActionStyle.Render("Quit"),
	}
	footer := lipgloss.JoinHorizontal(lipgloss.Left, footerParts...)
	footer = lipgloss.NewStyle().Background(lipgloss.Color("#7D56F4")).Width(m.TermWidth).Render(footer)

	view := lipgloss.JoinVertical(lipgloss.Left, header, body)

	// Add padding to ensure body fills space
	bodyHeight := lipgloss.Height(body)
	headerHeight := lipgloss.Height(header)
	neededPadding := m.TermHeight - bodyHeight - headerHeight - 1
	if neededPadding > 0 {
		view += strings.Repeat("\n", neededPadding)
	}

	view += footer

	if m.ShowMenu {
		menuContent := lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("F2 SETTINGS"),
			"",
			"[D] Switch to Dashboard",
			"[T] Switch to Traffic Monitor",
			"[R] Reload Configuration",
			"[C] Clear Metrics & Traffic",
			"[Esc] Close Menu",
			"[F10] Quit",
			"",
		)
		menu := menuStyle.Render(menuContent)

		return m.overlay(view, menu)
	}

	return view
}

func (m Model) overlay(base string, overlay string) string {
	// Crude overlay for now
	return base + "\n\n" + overlay
}
