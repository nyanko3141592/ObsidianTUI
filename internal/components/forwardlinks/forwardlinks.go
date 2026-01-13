package forwardlinks

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/internal/vault"
)

var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("32")).Padding(0, 1)
	subtitleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	selectedStyle  = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
	linkStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	brokenStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	moreStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	noLinksStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	containerStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("32")).Padding(1)
)

// Link represents a forward link with its resolved status
type Link struct {
	Target       string // original target text
	ResolvedPath string // resolved file path (empty if broken)
	IsBroken     bool
}

type Model struct {
	vault    *vault.Vault
	links    []Link
	cursor   int
	width    int
	height   int
	active   bool
	filePath string
}

type FileSelectedMsg struct {
	Path string
}

type ForwardLinksClosedMsg struct{}

func New(v *vault.Vault) Model {
	return Model{
		vault: v,
	}
}

func (m *Model) Show(filePath string) {
	m.active = true
	m.filePath = filePath
	m.cursor = 0
	m.buildLinks()
}

func (m *Model) buildLinks() {
	m.links = nil

	file, ok := m.vault.Files[m.filePath]
	if !ok {
		return
	}

	seen := make(map[string]bool)
	for _, link := range file.Links {
		if !link.IsWikiLink {
			continue
		}

		target := link.Target
		if seen[target] {
			continue
		}
		seen[target] = true

		// Try to resolve the link
		resolvedPath := m.vault.FindFile(target + ".md")
		if resolvedPath == "" {
			resolvedPath = m.vault.FindFile(target)
		}

		l := Link{
			Target:       target,
			ResolvedPath: resolvedPath,
			IsBroken:     resolvedPath == "",
		}
		m.links = append(m.links, l)
	}

	// Sort: resolved links first, then alphabetically
	sort.Slice(m.links, func(i, j int) bool {
		if m.links[i].IsBroken != m.links[j].IsBroken {
			return !m.links[i].IsBroken
		}
		return m.links[i].Target < m.links[j].Target
	})
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
		case "esc", "q":
			m.Hide()
			return m, func() tea.Msg { return ForwardLinksClosedMsg{} }

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.links)-1 {
				m.cursor++
			}

		case "enter":
			if m.cursor < len(m.links) {
				link := m.links[m.cursor]
				if !link.IsBroken {
					m.Hide()
					return m, func() tea.Msg { return FileSelectedMsg{Path: link.ResolvedPath} }
				}
			}

		case "g":
			m.cursor = 0

		case "G":
			if len(m.links) > 0 {
				m.cursor = len(m.links) - 1
			}

		case "pgup", "ctrl+u":
			m.cursor -= m.height / 2
			if m.cursor < 0 {
				m.cursor = 0
			}

		case "pgdown", "ctrl+d":
			m.cursor += m.height / 2
			if m.cursor >= len(m.links) {
				m.cursor = len(m.links) - 1
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
			if m.cursor >= len(m.links) {
				m.cursor = len(m.links) - 1
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Forward Links") + "\n")
	b.WriteString(subtitleStyle.Render("Links from: "+m.filePath) + "\n\n")

	if len(m.links) > 0 {
		maxItems := m.height - 8
		if maxItems < 3 {
			maxItems = 3
		}

		start := 0
		if m.cursor >= maxItems {
			start = m.cursor - maxItems + 1
		}

		end := start + maxItems
		if end > len(m.links) {
			end = len(m.links)
		}

		brokenCount := 0
		for _, l := range m.links {
			if l.IsBroken {
				brokenCount++
			}
		}

		if brokenCount > 0 {
			b.WriteString(brokenStyle.Render("⚠ " + itoa(brokenCount) + " broken link(s)") + "\n\n")
		}

		for i := start; i < end; i++ {
			link := m.links[i]

			displayText := link.Target
			maxLen := m.width - 10
			if maxLen > 0 && len(displayText) > maxLen {
				displayText = displayText[:maxLen-3] + "..."
			}

			var style lipgloss.Style
			prefix := "  "

			if i == m.cursor {
				style = selectedStyle
				prefix = "▶ "
			} else if link.IsBroken {
				style = brokenStyle
				prefix = "✗ "
			} else {
				style = linkStyle
				prefix = "→ "
			}

			b.WriteString(style.Render(prefix+displayText) + "\n")
		}

		if len(m.links) > maxItems {
			b.WriteString(moreStyle.Render("\n  ... and more"))
		}
	} else {
		b.WriteString(noLinksStyle.Render("  No outgoing links"))
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
