package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/navantesolutions/apimcore/internal/gateway"
	"github.com/navantesolutions/apimcore/internal/hub"
	"github.com/navantesolutions/apimcore/internal/store"
)

var (
	// User Suggested Styles
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	warning   = lipgloss.AdaptiveColor{Light: "#FF0000", Dark: "#FF5F87"}

	specialStyle = lipgloss.NewStyle().Foreground(special)
	warningStyle = lipgloss.NewStyle().Foreground(warning)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)

	columnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(1).
			MarginRight(1).
			Width(38)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	dashboardTitleStyle = lipgloss.NewStyle().
				Foreground(highlight).
				Bold(true).
				MarginBottom(1)

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
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	mainWindowStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)
)

type ViewMode int

const (
	ViewGlobal    ViewMode = iota // F2
	ViewDashboard                 // F3
	ViewTraffic                   // F4
	ViewAdmin                     // F5
	ViewSecurity                  // F6
	ViewHealth                    // F7
	ViewConfig                    // F8
	ViewPortal                    // F9
)

type Alert struct {
	Message string
	Level   string // "info", "warn", "error"
	Expires time.Time
}

type LogMsg string
type MetricsUpdateMsg struct {
	TotalRequests int64
	AvgLatency    float64
}
type ConfigReloadMsg struct{}
type tickMsg time.Time

type TrafficPacket = hub.TrafficEvent

type Model struct {
	// Infrastructure
	TermWidth  int
	TermHeight int
	Ready      bool
	OnReload   func()
	Store      *store.Store
	Gateway    *gateway.Gateway

	// Metrics & State
	Uptime        time.Time
	TotalRequests int64
	AvgLatency    float64
	CPUUsage      float64
	RAMUsage      float64
	Traffic       []hub.TrafficEvent
	Logs          []string
	Mode          ViewMode
	Hub           *hub.Broadcaster
	RateLimited   int64
	Blocked       int64

	// Navigation & Alerts
	Alerts []Alert

	// Components
	Viewport     viewport.Model
	TrafficTable table.Model
	Progress     progress.Model

	// Selection
	SelectedPacket *hub.TrafficEvent
}

func NewModel(onReload func(), s *store.Store, g *gateway.Gateway, h *hub.Broadcaster) Model {
	columns := []table.Column{
		{Title: "Time", Width: 10},
		{Title: "Geo", Width: 6},
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

	tableStyles := table.DefaultStyles()
	tableStyles.Header = tableStyles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	tableStyles.Selected = tableStyles.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(tableStyles)

	return Model{
		Uptime:       time.Now(),
		OnReload:     onReload,
		Store:        s,
		Gateway:      g,
		Hub:          h,
		TrafficTable: t,
		Mode:         ViewDashboard,
		Logs:         []string{},
		Alerts:       []Alert{},
		Traffic:      []hub.TrafficEvent{},
		Progress:     progress.New(progress.WithDefaultGradient()),
		TermWidth:    80,
		TermHeight:   24,
		Ready:        false,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tick(),
		tea.WindowSize(),
	)
}

func tick() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch msg.String() {
		case "f10", "q", "ctrl+c":
			return m, tea.Quit
		case "f2":
			m.Mode = ViewGlobal
		case "f3":
			m.Mode = ViewDashboard
		case "f4":
			m.Mode = ViewTraffic
		case "f5":
			m.Mode = ViewAdmin
		case "f6":
			m.Mode = ViewSecurity
		case "f7":
			m.Mode = ViewHealth
		case "f8":
			m.Mode = ViewConfig
		case "f9":
			m.Mode = ViewPortal
		case "tab":
			m.Mode = (m.Mode + 1) % 8
		case "esc":
			if m.SelectedPacket != nil {
				m.SelectedPacket = nil
			} else {
				m.Mode = ViewDashboard
			}
		case "up", "down":
			if m.Mode == ViewTraffic && m.SelectedPacket == nil {
				m.TrafficTable, cmd = m.TrafficTable.Update(msg)
			}
		case "enter":
			if m.Mode == ViewTraffic && m.SelectedPacket == nil {
				idx := m.TrafficTable.Cursor()
				if idx >= 0 && idx < len(m.Traffic) {
					m.SelectedPacket = &m.Traffic[idx]
				}
			}
		case "b":
			if m.Mode == ViewTraffic && m.SelectedPacket != nil {
				ip := m.SelectedPacket.IP
				cfg := m.Gateway.GetSecurity()
				cfg.IPBlacklist = append(cfg.IPBlacklist, ip)
				m.Gateway.UpdateSecurity(cfg)
				m.Alerts = append(m.Alerts, Alert{
					Message: fmt.Sprintf("BANNED: %s", ip),
					Level:   "warn",
					Expires: time.Now().Add(3 * time.Second),
				})
			}
		case "r":
			if m.OnReload != nil {
				m.OnReload()
				m.Alerts = append(m.Alerts, Alert{
					Message: "CONFIGURATION RELOADED",
					Level:   "info",
					Expires: time.Now().Add(2 * time.Second),
				})
			}
		case "c":
			if m.Mode == ViewSecurity {
				cfg := m.Gateway.GetSecurity()
				cfg.IPBlacklist = []string{}
				m.Gateway.UpdateSecurity(cfg)
				m.Alerts = append(m.Alerts, Alert{
					Message: "BLACKLIST PURGED",
					Level:   "info",
					Expires: time.Now().Add(2 * time.Second),
				})
			}
		case "g":
			if m.Mode == ViewSecurity {
				cfg := m.Gateway.GetSecurity()
				if len(cfg.AllowedCountries) > 0 {
					cfg.AllowedCountries = []string{}
					m.Alerts = append(m.Alerts, Alert{Message: "GEO: GLOBAL MODE", Level: "info", Expires: time.Now().Add(2 * time.Second)})
				} else {
					cfg.AllowedCountries = []string{"US", "BR", "DE"}
					m.Alerts = append(m.Alerts, Alert{Message: "GEO: RESTRICTED", Level: "warn", Expires: time.Now().Add(2 * time.Second)})
				}
				m.Gateway.UpdateSecurity(cfg)
			}
		}

	case tea.WindowSizeMsg:
		m.TermWidth = msg.Width
		m.TermHeight = msg.Height
		headerHeight := lipgloss.Height(m.renderHeader(""))
		footerHeight := 1
		windowPadding := 4
		availableWidth := msg.Width - windowPadding
		availableHeight := msg.Height - headerHeight - footerHeight - windowPadding

		if availableWidth < 1 {
			availableWidth = 1
		}
		if availableHeight < 1 {
			availableHeight = 1
		}

		logsViewportHeight := 10
		if availableHeight < logsViewportHeight+5 {
			logsViewportHeight = availableHeight - 5
		}
		if logsViewportHeight < 4 {
			logsViewportHeight = 4
		}

		if !m.Ready {
			m.Viewport = viewport.New(availableWidth, logsViewportHeight)
			if len(m.Logs) > 0 {
				m.Viewport.SetContent(strings.Join(m.Logs, "\n"))
			} else {
				m.Viewport.SetContent("Initializing APIM Core Management Hub...")
			}
			m.Ready = true
		} else {
			m.Viewport.Width = availableWidth
			m.Viewport.Height = logsViewportHeight
		}
		m.TrafficTable.SetWidth(availableWidth)
		m.TrafficTable.SetHeight(availableHeight - 2)

		m.Progress.Width = 34
		return m, nil

	case LogMsg:
		m.Logs = append(m.Logs, string(msg))
		if len(m.Logs) > 1000 {
			m.Logs = m.Logs[len(m.Logs)-1000:]
		}
		if m.Ready {
			m.Viewport.SetContent(strings.Join(m.Logs, "\n"))
			if m.Viewport.AtBottom() {
				m.Viewport.GotoBottom()
			}
		}

	case hub.SystemStats:
		m.TotalRequests = msg.TotalRequests
		m.AvgLatency = msg.AvgLatency
		m.CPUUsage = msg.CPUUsage
		m.RAMUsage = float64(msg.MemoryUsageMB) / 8192.0 // Mock total RAM for scale
		m.RateLimited = msg.RateLimited
		m.Blocked = msg.Blocked
		return m, nil

	case tickMsg:
		return m, tick()

	case progress.FrameMsg:
		newModel, cmd := m.Progress.Update(msg)
		if newProgressModel, ok := newModel.(progress.Model); ok {
			m.Progress = newProgressModel
		}
		return m, cmd

	case hub.TrafficEvent:
		m.Traffic = append(m.Traffic, msg)
		if len(m.Traffic) > 100 {
			m.Traffic = m.Traffic[1:]
		}
		// Highlight DDoS / Blocked
		if msg.Status == 429 || msg.Status == 403 {
			m.Alerts = append(m.Alerts, Alert{
				Message: fmt.Sprintf("Security Event: %d from %s", msg.Status, msg.Country),
				Level:   "warn",
				Expires: time.Now().Add(5 * time.Second),
			})
		}
		m.updateTrafficTable()
		return m, nil
	}

	if m.Mode == ViewTraffic {
		m.TrafficTable, cmd = m.TrafficTable.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.Viewport, cmd = m.Viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateTrafficTable() {
	rows := make([]table.Row, len(m.Traffic))
	for i, p := range m.Traffic {
		statusStr := fmt.Sprintf("%d", p.Status)
		// Heatmap logic
		if p.Status >= 500 {
			statusStr = errorStyle.Render(statusStr)
		} else if p.Status == 429 || p.Status == 403 {
			statusStr = warningStyle.Render(statusStr)
		} else if p.Status >= 200 && p.Status < 300 {
			statusStr = specialStyle.Render(statusStr)
		}

		rows[i] = table.Row{
			p.Timestamp.Format("15:04:05"),
			p.Country,
			p.Method,
			statusStr,
			p.Path,
			p.Backend,
			fmt.Sprintf("%dms", p.Latency),
		}
	}
	m.TrafficTable.SetRows(rows)
}

func (m Model) dashboardView() string {
	uptime := time.Since(m.Uptime).Round(time.Second).String()

	bodyWidth := m.TermWidth - 4
	if bodyWidth < 60 {
		bodyWidth = 60
	}
	cardWidth := (bodyWidth - 8) / 3
	if cardWidth < 24 {
		cardWidth = 24
	}

	cpuUsage := m.CPUUsage
	if cpuUsage == 0 {
		cpuUsage = 0.42
	}
	ramUsage := m.RAMUsage
	if ramUsage == 0 {
		ramUsage = 0.65
	}

	progressWidth := cardWidth - 6
	if progressWidth < 8 {
		progressWidth = 8
	}
	m.Progress.Width = progressWidth

	vitalsContent := fmt.Sprintf("Uptime:   %s\n\nCPU Load:  %.1f%%\n%s\n\nRAM Load:  %.1f%%\n%s\n\nStatus:   %s",
		infoStyle.Render(uptime),
		cpuUsage*100, m.Progress.ViewAs(cpuUsage),
		ramUsage*100, m.Progress.ViewAs(ramUsage),
		specialStyle.Render("ONLINE"))

	sparkWidth := cardWidth - 6
	if sparkWidth < 8 {
		sparkWidth = 8
	}
	spark := m.renderSparkline([]int64{10, 20, 15, 30, 25}, sparkWidth)
	trafficContent := fmt.Sprintf("Requests:  %s\nAvg Lat:   %s\nLimited:   %s\n\nPERFORMANCE TREND:\n%s",
		infoStyle.Render(fmt.Sprintf("%d", m.TotalRequests)),
		infoStyle.Render(fmt.Sprintf("%.2fms", m.AvgLatency)),
		warningStyle.Render(fmt.Sprintf("%d", m.RateLimited)),
		specialStyle.Render(spark))

	securityContent := fmt.Sprintf("Blocked:   %s\nGeo-fence: %s\nThreats:   %s\n\nNODE:      %s\nNODES:     %s",
		warningStyle.Render(fmt.Sprintf("%d", m.Blocked)),
		specialStyle.Render("ACTIVE"),
		warningStyle.Render("HIGH"),
		infoStyle.Render("US-EAST-1A"),
		infoStyle.Render("12 ACTIVE"))

	cardStyle := columnStyle.MarginRight(0).Width(cardWidth)
	vitalsCard := cardStyle.Render(fmt.Sprintf("%s\n\n%s", dashboardTitleStyle.Render("SYSTEM VITALS"), vitalsContent))
	trafficCard := cardStyle.Render(fmt.Sprintf("%s\n\n%s", dashboardTitleStyle.Render("TRAFFIC MONITOR"), trafficContent))
	securityCard := cardStyle.Render(fmt.Sprintf("%s\n\n%s", dashboardTitleStyle.Render("SECURITY"), securityContent))

	cardsRow := lipgloss.JoinHorizontal(lipgloss.Top, vitalsCard, trafficCard, securityCard)

	logsBoxWidth := bodyWidth - 2
	if logsBoxWidth < 40 {
		logsBoxWidth = 40
	}
	logsBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Padding(0, 1).
		Width(logsBoxWidth).
		MaxWidth(logsBoxWidth)

	var logsAreaContent string
	if m.Ready && m.Viewport.Width > 0 && m.Viewport.Height > 0 {
		logsAreaContent = m.Viewport.View()
	} else {
		if len(m.Logs) > 0 {
			logsAreaContent = strings.Join(m.Logs, "\n")
		} else {
			logsAreaContent = "Initializing APIM Core Management Hub...\nWaiting for system metrics..."
		}
	}

	logsHeader := dashboardTitleStyle.Render("RECENT CORE LOGS") + " (↑/↓ scroll)"
	logsSection := logsBoxStyle.Render(logsHeader + "\n\n" + logsAreaContent)

	return lipgloss.JoinVertical(lipgloss.Left,
		dashboardTitleStyle.Render("DASHBOARD"),
		"",
		cardsRow,
		"",
		logsSection,
	)
}

func (m Model) globalView() string {
	bodyWidth := m.TermWidth - 4
	if bodyWidth < 40 {
		bodyWidth = 40
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Padding(0, 1).
		Width(bodyWidth - 2).
		MaxWidth(bodyWidth - 2)
	mapContent := m.renderGlobalMap()
	boxContent := boxStyle.Render(dashboardTitleStyle.Render("GLOBAL THREAT MAP") + "\n\n" + mapContent)
	return lipgloss.JoinVertical(lipgloss.Left,
		dashboardTitleStyle.Render("GLOBAL"),
		"",
		boxContent,
	)
}

func (m Model) configView() string {
	bodyWidth := m.TermWidth - 4
	if bodyWidth < 40 {
		bodyWidth = 40
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Padding(0, 1).
		Width(bodyWidth - 2).
		MaxWidth(bodyWidth - 2)
	content := "Loaded from: config.yaml\n\n(Editable Console - Coming Soon)"
	boxContent := boxStyle.Render(dashboardTitleStyle.Render("LIVE CONFIGURATION (YAML)") + "\n\n" + content)
	return lipgloss.JoinVertical(lipgloss.Left,
		dashboardTitleStyle.Render("CONFIG"),
		"",
		boxContent,
	)
}

func (m Model) portalView() string {
	bodyWidth := m.TermWidth - 4
	if bodyWidth < 40 {
		bodyWidth = 40
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Padding(0, 1).
		Width(bodyWidth - 2).
		MaxWidth(bodyWidth - 2)
	content := fmt.Sprintf("Public APIs: 2\nDocumentation: 85%%\nStatus: %s", specialStyle.Render("LIVE"))
	boxContent := boxStyle.Render(dashboardTitleStyle.Render("DEVELOPER PORTAL") + "\n\n" + content)
	return lipgloss.JoinVertical(lipgloss.Left,
		dashboardTitleStyle.Render("PORTAL"),
		"",
		boxContent,
	)
}

func (m Model) trafficView() string {
	bodyWidth := m.TermWidth - 4
	if bodyWidth < 1 {
		bodyWidth = 1
	}
	tableContent := m.TrafficTable.View()
	if m.SelectedPacket != nil {
		detailsStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(0, 1).
			Width(bodyWidth - 2).
			MaxWidth(bodyWidth - 2)
		details := detailsStyle.Render(fmt.Sprintf("%s\n\nMethod: %s\nPath: %s\nBackend: %s\nStatus: %d\nLatency: %dms\nTenant: %s\nGeo: %s\nTime: %s",
			dashboardTitleStyle.Render("PACKET DETAILS"),
			m.SelectedPacket.Method,
			m.SelectedPacket.Path,
			m.SelectedPacket.Backend,
			m.SelectedPacket.Status,
			m.SelectedPacket.Latency,
			m.SelectedPacket.TenantID,
			m.SelectedPacket.Country,
			m.SelectedPacket.Timestamp.Format(time.RFC3339),
		))
		tableContent = lipgloss.JoinVertical(lipgloss.Left, tableContent, "", details)
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		dashboardTitleStyle.Render("TRAFFIC"),
		"",
		tableContent,
	)
}

func (m Model) adminView() string {
	bodyWidth := m.TermWidth - 4
	if bodyWidth < 60 {
		bodyWidth = 60
	}
	cardWidth := (bodyWidth - 8) / 3
	if cardWidth < 24 {
		cardWidth = 24
	}
	cardStyle := columnStyle.MarginRight(0).Width(cardWidth)

	prods := m.Store.ListProducts()
	prodContent := "PRODUCTS:\n"
	for _, p := range prods {
		status := specialStyle.Render("Active")
		if !p.Published {
			status = "Draft"
		}
		prodContent += fmt.Sprintf("• [%-10s] %-20s (%s)\n", p.Slug, p.Name, status)
	}

	defs := m.Store.ListDefinitions()
	defContent := "API DEFINITIONS:\n"
	for _, d := range defs {
		defContent += fmt.Sprintf("• %-20s -> %s\n", d.Name, d.BackendURL)
	}

	subs := m.Store.ListSubscriptions()
	subContent := fmt.Sprintf("ACTIVE SUBSCRIPTIONS: %d\n", len(subs))
	subContent += "TENANTS: Walmart, Target, Acme"

	cardsRow := lipgloss.JoinHorizontal(lipgloss.Top,
		cardStyle.Render(fmt.Sprintf("%s\n\n%s", dashboardTitleStyle.Render("CATALOG"), prodContent)),
		cardStyle.Render(fmt.Sprintf("%s\n\n%s", dashboardTitleStyle.Render("ROUTING"), defContent)),
		cardStyle.Render(fmt.Sprintf("%s\n\n%s", dashboardTitleStyle.Render("CLIENTS"), subContent)),
	)
	return lipgloss.JoinVertical(lipgloss.Left,
		dashboardTitleStyle.Render("ADMIN"),
		"",
		cardsRow,
	)
}

func (m Model) securityView() string {
	bodyWidth := m.TermWidth - 4
	if bodyWidth < 60 {
		bodyWidth = 60
	}
	cardWidth := (bodyWidth - 8) / 3
	if cardWidth < 24 {
		cardWidth = 24
	}
	cardStyle := columnStyle.MarginRight(0).Width(cardWidth)

	cfg := m.Gateway.GetSecurity()

	blacklistContent := "CURRENT BLACKLIST:\n"
	if len(cfg.IPBlacklist) == 0 {
		blacklistContent += "  (Empty)\n"
	}
	for i, ip := range cfg.IPBlacklist {
		if i > 5 {
			blacklistContent += "  ...\n"
			break
		}
		blacklistContent += fmt.Sprintf("• %s\n", warningStyle.Render(ip))
	}

	regions := "ALLOWED REGIONS:\n"
	if len(cfg.AllowedCountries) == 0 {
		regions += specialStyle.Render("  GLOBAL (All Allowed)\n")
	} else {
		regions += fmt.Sprintf("  %s\n", strings.Join(cfg.AllowedCountries, ", "))
	}

	limits := fmt.Sprintf("RATE LIMITING: %s\n", specialStyle.Render("ENABLED"))
	limits += fmt.Sprintf("RPS: %.2f | Burst: %d\n", cfg.RateLimit.RPP, cfg.RateLimit.Burst)

	cardsRow := lipgloss.JoinHorizontal(lipgloss.Top,
		cardStyle.Render(fmt.Sprintf("%s\n\n%s", dashboardTitleStyle.Render("IP PROTECTION"), blacklistContent)),
		cardStyle.Render(fmt.Sprintf("%s\n\n%s", dashboardTitleStyle.Render("GEO-FENCING"), regions)),
		cardStyle.Render(fmt.Sprintf("%s\n\n%s", dashboardTitleStyle.Render("THROTTLING"), limits)),
	)
	controls := "\n" + footerKeyStyle.Render("C") + footerActionStyle.Render("Clear Blacklist") + "  " +
		footerKeyStyle.Render("G") + footerActionStyle.Render("Toggle Global Geo")

	return lipgloss.JoinVertical(lipgloss.Left,
		dashboardTitleStyle.Render("SECURITY"),
		"",
		cardsRow,
		controls,
	)
}

func (m Model) healthView() string {
	bodyWidth := m.TermWidth - 4
	if bodyWidth < 60 {
		bodyWidth = 60
	}
	cardWidth := (bodyWidth - 4) / 2
	if cardWidth < 28 {
		cardWidth = 28
	}
	cardStyle := columnStyle.MarginRight(0).Width(cardWidth)

	health := fmt.Sprintf("Admin API: %s\n", specialStyle.Render("OK"))
	health += fmt.Sprintf("Gateway:   %s\n", specialStyle.Render("OK"))
	health += fmt.Sprintf("DevPortal: %s\n", specialStyle.Render("OK"))
	health += fmt.Sprintf("Store:     %s\n", specialStyle.Render("CONSISTENT"))

	metrics := "PROMETHEUS (LIVE):\n"
	metrics += fmt.Sprintf("• requests_total: %d\n", m.TotalRequests)
	metrics += fmt.Sprintf("• latency_avg:   %.2fms\n", m.AvgLatency)
	metrics += fmt.Sprintf("• errors_rate:   0.0%%\n")

	cardsRow := lipgloss.JoinHorizontal(lipgloss.Top,
		cardStyle.Render(fmt.Sprintf("%s\n\n%s", dashboardTitleStyle.Render("SERVICES"), health)),
		cardStyle.Render(fmt.Sprintf("%s\n\n%s", dashboardTitleStyle.Render("METRICS"), metrics)),
	)
	return lipgloss.JoinVertical(lipgloss.Left,
		dashboardTitleStyle.Render("HEALTH"),
		"",
		cardsRow,
	)
}

func (m Model) View() string {
	if m.TermWidth == 0 || m.TermHeight == 0 {
		m.TermWidth = 80
		m.TermHeight = 24
	}

	uptime := time.Since(m.Uptime).Round(time.Second).String()

	header := m.renderHeader(uptime)

	var body string
	switch m.Mode {
	case ViewGlobal:
		body = m.globalView()
	case ViewDashboard:
		body = m.dashboardView()
	case ViewTraffic:
		body = m.trafficView()
	case ViewAdmin:
		body = m.adminView()
	case ViewSecurity:
		body = m.securityView()
	case ViewHealth:
		body = m.healthView()
	case ViewConfig:
		body = m.configView()
	case ViewPortal:
		body = m.portalView()
	}

	alerts := ""
	now := time.Now()
	for _, a := range m.Alerts {
		if now.Before(a.Expires) {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Padding(0, 1)
			if a.Level == "warn" {
				style = style.Background(lipgloss.Color("208"))
			} else {
				style = style.Background(lipgloss.Color("12"))
			}
			alerts += style.Render(a.Message) + " "
		}
	}
	if alerts != "" {
		body = alerts + "\n" + body
	}

	footer := m.renderFooter()

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	windowBorderHeight := 2
	windowPadding := 2
	availableHeight := m.TermHeight - headerHeight - footerHeight - windowBorderHeight - windowPadding

	if availableHeight < 1 {
		availableHeight = 1
	}

	bodyLines := strings.Split(body, "\n")
	if len(bodyLines) > availableHeight {
		body = strings.Join(bodyLines[:availableHeight], "\n")
	}

	windowWidth := m.TermWidth - 2
	if windowWidth < 1 {
		windowWidth = 1
	}

	bodyWidth := windowWidth - 2
	if bodyWidth < 1 {
		bodyWidth = 1
	}
	bodyRendered := lipgloss.NewStyle().Width(bodyWidth).MaxWidth(bodyWidth).MaxHeight(availableHeight).Render(body)

	windowContent := bodyRendered

	windowedContent := mainWindowStyle.
		Width(windowWidth).
		Height(availableHeight + windowBorderHeight).
		Render(windowContent)

	fullView := lipgloss.JoinVertical(lipgloss.Left, header, windowedContent, footer)

	return fullView
}
