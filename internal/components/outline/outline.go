package outline

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("24")).Padding(0, 1)
	h1Style        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	h2Style        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	h3Style        = lipgloss.NewStyle().Foreground(lipgloss.Color("228"))
	h4Style        = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	selectedStyle  = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
	containerStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("24")).Padding(1)
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	emptyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// Heading represents a markdown heading
type Heading struct {
	Text   string
	Level  int
	Line   int // 0-indexed line number
}

// Model is the outline view model
type Model struct {
	headings []Heading
	cursor   int
	width    int
	height   int
	active   bool
	filePath string
}

// JumpToLineMsg is sent when user selects a heading
type JumpToLineMsg struct {
	Line int
}

// OutlineClosedMsg is sent when outline is closed
type OutlineClosedMsg struct{}

func New() Model {
	return Model{}
}

// ParseHeadings extracts headings from markdown content
func ParseHeadings(content string) []Heading {
	var headings []Heading
	lines := strings.Split(content, "\n")

	inCodeBlock := false
	for i, line := range lines {
		// Check for code block
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		// Check for heading
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			level := 0
			for _, ch := range trimmed {
				if ch == '#' {
					level++
				} else {
					break
				}
			}

			if level > 0 && level <= 6 {
				text := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
				if text != "" {
					headings = append(headings, Heading{
						Text:  text,
						Level: level,
						Line:  i,
					})
				}
			}
		}
	}

	return headings
}

func (m *Model) SetContent(content string, filePath string) {
	m.headings = ParseHeadings(content)
	m.filePath = filePath
	m.cursor = 0
}

func (m *Model) Show() {
	m.active = true
	m.cursor = 0
}

func (m *Model) Hide() {
	m.active = false
}

func (m Model) Active() bool {
	return m.active
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+o":
			m.Hide()
			return m, func() tea.Msg { return OutlineClosedMsg{} }

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.headings)-1 {
				m.cursor++
			}

		case "enter":
			if m.cursor < len(m.headings) {
				line := m.headings[m.cursor].Line
				m.Hide()
				return m, func() tea.Msg { return JumpToLineMsg{Line: line} }
			}

		case "g":
			m.cursor = 0

		case "G":
			if len(m.headings) > 0 {
				m.cursor = len(m.headings) - 1
			}

		case "1":
			// Jump to next h1
			m.jumpToLevel(1)

		case "2":
			// Jump to next h2
			m.jumpToLevel(2)

		case "3":
			// Jump to next h3
			m.jumpToLevel(3)

		case "pgup", "ctrl+u":
			m.cursor -= m.height / 2
			if m.cursor < 0 {
				m.cursor = 0
			}

		case "pgdown", "ctrl+d":
			m.cursor += m.height / 2
			if m.cursor >= len(m.headings) {
				m.cursor = len(m.headings) - 1
			}
		}

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.cursor -= 3
			if m.cursor < 0 {
				m.cursor = 0
			}
		case tea.MouseButtonWheelDown:
			m.cursor += 3
			if m.cursor >= len(m.headings) {
				m.cursor = len(m.headings) - 1
			}
		}
	}

	return m, nil
}

func (m *Model) jumpToLevel(level int) {
	// Find next heading of the specified level from current position
	for i := m.cursor + 1; i < len(m.headings); i++ {
		if m.headings[i].Level == level {
			m.cursor = i
			return
		}
	}
	// Wrap around from beginning
	for i := 0; i < m.cursor; i++ {
		if m.headings[i].Level == level {
			m.cursor = i
			return
		}
	}
}

func (m Model) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	// Title
	title := titleStyle.Render("Outline")
	b.WriteString(title + "\n")

	// Stats
	stats := headerStyle.Render("Enter: jump | 1/2/3: jump to H1/H2/H3")
	b.WriteString(stats + "\n\n")

	if len(m.headings) == 0 {
		b.WriteString(emptyStyle.Render("No headings found"))
		return containerStyle.Width(m.width).Render(b.String())
	}

	maxVisible := m.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.headings) {
		end = len(m.headings)
	}

	for i := start; i < end; i++ {
		heading := m.headings[i]

		// Indent based on level
		indent := strings.Repeat("  ", heading.Level-1)

		// Style based on level
		var style lipgloss.Style
		switch heading.Level {
		case 1:
			style = h1Style
		case 2:
			style = h2Style
		case 3:
			style = h3Style
		default:
			style = h4Style
		}

		text := heading.Text
		maxLen := m.width - len(indent) - 10
		if maxLen > 0 && len(text) > maxLen {
			text = text[:maxLen-3] + "..."
		}

		line := indent
		if i == m.cursor {
			line += selectedStyle.Render("▶ " + text)
		} else {
			line += style.Render("  " + text)
		}

		b.WriteString(line + "\n")
	}

	// Scroll indicator
	if len(m.headings) > maxVisible {
		scrollInfo := emptyStyle.Render("\n[" + string(rune('0'+m.cursor+1)) + "/" + string(rune('0'+len(m.headings))) + "]")
		if m.cursor+1 >= 10 || len(m.headings) >= 10 {
			scrollInfo = emptyStyle.Render("\n[" + itoa(m.cursor+1) + "/" + itoa(len(m.headings)) + "]")
		}
		b.WriteString(scrollInfo)
	}

	return containerStyle.Width(m.width).Render(b.String())
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}
