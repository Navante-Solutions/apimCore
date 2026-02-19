package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const brailleBlank = '\u2800'

var (
	brailleMapStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#252525")).
			Bold(true)
	brailleMapPointsStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				Bold(true)

	threatStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00F2FF")).
			Bold(true)
)

func (m Model) renderGlobalMap() string {
	rows := []string{
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣀⣤⣤⣤⠤⣀⣤⣴⣿⣶⣆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠤⣤⠠⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠄⠀⠻⢂⣴⡿⠋⠁⠚⠿⣿⣿⣿⣿⣿⣷⠀⠀⠀⠀⠀⠀⠀⠀⠀⠁⠀⠀⠀⠀⠀⠀⠀⠀⡠⠄⠀⠀⠀⠀⠀⢠⣤⣤⣴⣷⡶⠂⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣤⣀⡒⠒⠀⢘⠀⡐⡂⣒⠀⠀⠀⠀⠀⢻⣿⣿⣿⣿⣿⠟⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⣤⣄⣀⠀⠀⠀⠘⠂⢀⡀⣾⠰⣶⣽⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣤⣤⣶⣶⣶⣤⣤⣀⣀⢀⣀⡀⠀⠀",
		"⠀⣤⣶⣦⣤⣤⣄⣠⣤⣤⣥⣈⡿⠟⠳⠈⡔⣆⡙⡟⠛⠷⣦⡀⠀⢰⣿⣿⣿⠟⠁⠠⠤⠄⠀⠀⠀⠀⢀⣴⣿⢟⣿⣿⣏⣡⣼⣾⣿⣿⣿⣿⣿⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠇⠀⠀",
		"⠐⣺⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠋⠄⠀⠐⠿⠍⠀⠀⠙⠿⠟⠀⠀⠀⠀⠀⠀⢀⠀⠐⠟⢿⠇⠘⣯⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠛⠛⠛⢉⡕⠋⠉⠁⠀⠀⠀",
		"⠀⠙⠻⠋⠁⠉⠉⠛⢿⣿⣿⣿⣿⣿⣿⣿⣿⣄⡀⠀⠐⣿⣧⣴⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠰⠨⠆⣠⣼⣤⣴⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣧⡤⠀⠀⠀⠟⠁⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣧⣼⣿⣿⠿⠿⣂⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣺⣿⠿⡿⢿⣿⣿⠟⠟⠿⣿⡟⢹⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⠁⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⠓⠀⠈⠀⠀⠀⠀⠀⠀⠀⠀⠸⡿⢋⣀⣀⠈⠂⠛⠙⠶⢶⣶⣿⣗⣀⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣏⠉⢫⠀⢀⡐⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠹⢿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⣿⣿⣿⣿⣶⣤⣶⣤⣤⢼⣿⣿⡟⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠀⠀⠈⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⢿⣿⣿⠟⠛⠉⠻⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠠⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣧⠹⣿⣿⣶⡶⠀⠈⠛⢿⣿⣿⠟⠻⣿⣿⡿⠛⠛⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠹⢿⣄⣠⠄⠀⠀⠀⠄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠐⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣜⣋⠁⠀⠀⠀⠀⠘⡟⠀⠀⠀⠈⠛⢿⠆⠀⠀⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠙⠆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠙⠛⠋⠉⠛⣿⣿⣿⣿⣿⣿⣿⡿⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠰⠀⠀⠤⠂⠀⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⣿⣿⣾⣦⣄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⢻⣿⣿⣿⣿⣿⡏⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠰⣄⠀⢀⣰⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⣿⣿⣿⣿⣿⣷⣦⣤⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣸⣿⣿⣿⣿⣿⠇⢀⣠⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠁⠀⠁⠈⠀⠀⠈⠙⠷⠢⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⣿⣿⣿⣿⣿⣿⣿⣿⡟⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⣿⣿⣿⣿⠇⠀⠸⠇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⣴⣿⣤⣰⡄⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢹⣿⣿⣿⣿⣿⠿⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠹⡿⠿⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣶⣿⣿⣿⣿⣿⣿⣿⣦⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⣿⣿⣿⡿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⠿⠟⠋⠙⠻⣿⣿⡿⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⣿⣿⠿⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠩⠀⠀⠀⠀⠀⡠⠂",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⣿⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⢏⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
	}

	res := ""
	for _, row := range rows {
		var line strings.Builder
		for _, r := range row {
			if r == brailleBlank {
				line.WriteString(brailleMapStyle.Render(string(r)))
			} else {
				line.WriteString(brailleMapPointsStyle.Render(string(r)))
			}
		}
		res += line.String() + "\n"
	}

	threat := threatStyle.Render("★")
	res += "\n" + fmt.Sprintf("%s [LIVE THREATS]: ", threat)
	if len(m.GeoThreats) == 0 {
		res += specialStyle.Render("None")
	} else {
		keys := make([]string, 0, len(m.GeoThreats))
		for k := range m.GeoThreats {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, country := range keys {
			parts = append(parts, warningStyle.Render(fmt.Sprintf("%s: %d", country, m.GeoThreats[country])))
		}
		res += strings.Join(parts, " | ")
	}
	return res
}

func (m Model) renderHeader(uptime string) string {
	width := m.TermWidth
	if width < 1 {
		width = 80
	}

	leftWidth := width / 3
	middleWidth := width / 3
	rightWidth := width - leftWidth - middleWidth

	if leftWidth < 20 {
		leftWidth = 20
		middleWidth = (width - leftWidth) / 2
		rightWidth = width - leftWidth - middleWidth
	}
	if middleWidth < 20 {
		middleWidth = 20
		rightWidth = width - leftWidth - middleWidth
	}
	if rightWidth < 20 {
		rightWidth = 20
		leftWidth = (width - rightWidth - middleWidth)
	}

	leftContent := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("ApimCore - MANAGEMENT HUB"),
		headerLabelStyle.Render("Uptime:  ")+infoStyle.Render(uptime),
		headerLabelStyle.Render("Requests: ")+infoStyle.Render(fmt.Sprintf("%d", m.TotalRequests)),
	)

	middleContent := lipgloss.JoinVertical(lipgloss.Left,
		headerLabelStyle.Render("Latency:  ")+infoStyle.Render(fmt.Sprintf("%.2fms", m.AvgLatency)),
		headerLabelStyle.Render("Status:   ")+specialStyle.Render("RUNNING"),
	)

	quickCommands := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("QUICK COMMANDS: [R] Reload Config  [C] Clear Blacklist")

	rightContent := lipgloss.JoinVertical(lipgloss.Left,
		quickCommands,
		"",
	)

	leftRendered := lipgloss.NewStyle().Width(leftWidth).MaxWidth(leftWidth).Render(leftContent)
	middleRendered := lipgloss.NewStyle().Width(middleWidth).MaxWidth(middleWidth).Render(middleContent)
	rightRendered := lipgloss.NewStyle().Width(rightWidth).MaxWidth(rightWidth).Render(rightContent)

	headerContent := lipgloss.JoinHorizontal(lipgloss.Top, leftRendered, middleRendered, rightRendered)

	return headerStyle.Width(width).MaxWidth(width).Render(headerContent)
}

var (
	activeFooterKeyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#7D56F4")).
				Padding(0, 1)
	activeFooterActionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#7D56F4")).
				Bold(true).
				Padding(0, 1)
)

func (m Model) renderFooter() string {
	width := m.TermWidth
	if width < 1 {
		width = 80
	}

	items := []struct {
		key   string
		label string
	}{
		{"F2", "Global"},
		{"F3", "Dash"},
		{"F4", "Traffic"},
		{"F5", "Admin"},
		{"F6", "Sec"},
		{"F7", "Health"},
		{"F8", "Cfg"},
		{"F9", "Dev"},
		{"F10", "Quit"},
	}

	var footerParts []string
	for i, it := range items {
		if i < 8 && int(m.Mode) == i {
			footerParts = append(footerParts, activeFooterKeyStyle.Render(it.key)+activeFooterActionStyle.Render(it.label))
		} else {
			footerParts = append(footerParts, footerKeyStyle.Render(it.key)+footerActionStyle.Render(it.label))
		}
	}

	footer := lipgloss.JoinHorizontal(lipgloss.Left, footerParts...)
	return lipgloss.NewStyle().Background(lipgloss.Color("#353535")).Width(width).MaxWidth(width).Render(footer)
}

func (m Model) renderSparkline(data []int64, width int) string {
	if len(data) == 0 {
		return strings.Repeat("─", width)
	}

	max := int64(0)
	for _, v := range data {
		if v > max {
			max = v
		}
	}
	if max == 0 {
		max = 1
	}

	sparks := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	result := ""
	step := float64(len(data)) / float64(width)

	for i := 0; i < width; i++ {
		idx := int(float64(i) * step)
		if idx >= len(data) {
			idx = len(data) - 1
		}
		value := data[idx]
		height := int((float64(value) / float64(max)) * 7)
		if height < 0 {
			height = 0
		}
		if height >= len(sparks) {
			height = len(sparks) - 1
		}
		result += specialStyle.Render(sparks[height])
	}
	return result
}
