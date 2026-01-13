package cmdpalette

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 1)
	inputStyle       = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1)
	selectedStyle    = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
	normalStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	descStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	keyStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	containerStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("57")).Padding(1)
)

// Command represents a palette command
type Command struct {
	ID          string
	Name        string
	Description string
	Key         string
}

// CommandMsg is sent when a command is selected
type CommandMsg struct {
	ID string
}

// PaletteClosedMsg is sent when palette is closed
type PaletteClosedMsg struct{}

// Model is the command palette model
type Model struct {
	textinput textinput.Model
	commands  []Command
	filtered  []Command
	cursor    int
	width     int
	height    int
	active    bool
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "Type to search commands..."
	ti.CharLimit = 100
	ti.Width = 40

	return Model{
		textinput: ti,
		commands:  defaultCommands(),
	}
}

func defaultCommands() []Command {
	return []Command{
		{ID: "search", Name: "Search Files", Description: "Search for files in vault", Key: "/"},
		{ID: "graph", Name: "Graph View", Description: "View note connections", Key: "C-g"},
		{ID: "tags", Name: "Tags", Description: "Browse all tags", Key: "C-t"},
		{ID: "outline", Name: "Outline", Description: "View document outline", Key: "C-l"},
		{ID: "backlinks", Name: "Backlinks", Description: "Show files linking to current", Key: "C-b"},
		{ID: "forwardlinks", Name: "Forward Links", Description: "Show files linked from current", Key: "M-f"},
		{ID: "daily", Name: "Daily Note", Description: "Open today's daily note", Key: "M-d"},
		{ID: "save", Name: "Save File", Description: "Save current file", Key: "C-s"},
		{ID: "refresh", Name: "Refresh Vault", Description: "Rescan vault files", Key: "C-r"},
		{ID: "newfile", Name: "New File", Description: "Create a new file", Key: "C-n"},
		{ID: "help", Name: "Toggle Help", Description: "Show/hide keybindings help", Key: "?"},
		{ID: "edit", Name: "Edit Mode", Description: "Switch to edit view", Key: "M-e"},
		{ID: "preview", Name: "Preview Mode", Description: "Switch to preview view", Key: "M-p"},
		{ID: "split", Name: "Split Mode", Description: "Switch to split view", Key: "M-s"},
		{ID: "toggle", Name: "Cycle View", Description: "Cycle through view modes", Key: "C-e"},
	}
}

func (m *Model) Show() tea.Cmd {
	m.active = true
	m.cursor = 0
	m.textinput.SetValue("")
	m.filtered = m.commands
	return m.textinput.Focus()
}

func (m *Model) Hide() {
	m.active = false
	m.textinput.Blur()
}

func (m Model) Active() bool {
	return m.active
}

func (m *Model) filter() {
	query := strings.ToLower(m.textinput.Value())
	if query == "" {
		m.filtered = m.commands
		return
	}

	m.filtered = nil
	for _, cmd := range m.commands {
		if strings.Contains(strings.ToLower(cmd.Name), query) ||
			strings.Contains(strings.ToLower(cmd.Description), query) ||
			strings.Contains(strings.ToLower(cmd.ID), query) {
			m.filtered = append(m.filtered, cmd)
		}
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.Hide()
			return m, func() tea.Msg { return PaletteClosedMsg{} }

		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil

		case "enter":
			if m.cursor < len(m.filtered) {
				cmd := m.filtered[m.cursor]
				m.Hide()
				return m, func() tea.Msg { return CommandMsg{ID: cmd.ID} }
			}
			return m, nil

		case "pgup":
			m.cursor -= 5
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, nil

		case "pgdown":
			m.cursor += 5
			if m.cursor >= len(m.filtered) {
				m.cursor = len(m.filtered) - 1
			}
			return m, nil
		}
	}

	// Update textinput
	var cmd tea.Cmd
	m.textinput, cmd = m.textinput.Update(msg)
	m.filter()

	return m, cmd
}

func (m Model) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	// Title
	title := titleStyle.Render("Command Palette")
	b.WriteString(title + "\n")

	// Input
	b.WriteString(inputStyle.Render(m.textinput.View()) + "\n\n")

	// Commands list
	maxVisible := m.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := start; i < end; i++ {
		cmd := m.filtered[i]

		name := cmd.Name
		maxNameLen := m.width - 20
		if maxNameLen > 0 && len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		key := keyStyle.Render("[" + cmd.Key + "]")
		desc := descStyle.Render(cmd.Description)

		var style lipgloss.Style
		if i == m.cursor {
			style = selectedStyle
			name = "▶ " + name
		} else {
			style = normalStyle
			name = "  " + name
		}

		line := style.Render(name) + " " + key
		b.WriteString(line + "\n")
		b.WriteString("    " + desc + "\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(descStyle.Render("  No commands found"))
	}

	return containerStyle.Width(m.width).Render(b.String())
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textinput.Width = width - 10
}
