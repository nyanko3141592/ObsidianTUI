package tagpane

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/internal/vault"
)

var (
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("166")).Padding(0, 1)
	tagStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("135"))
	tagSelectedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	fileStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	fileSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	countStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	containerStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("166")).Padding(1)
	headerStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

type ViewMode int

const (
	ViewTags ViewMode = iota
	ViewFiles
)

type Tag struct {
	Name  string
	Count int
}

type Model struct {
	vault       *vault.Vault
	tags        []Tag
	files       []string
	tagCursor   int
	fileCursor  int
	width       int
	height      int
	active      bool
	viewMode    ViewMode
	selectedTag string
}

type FileSelectedMsg struct {
	Path string
}

type TagPaneClosedMsg struct{}

func New(v *vault.Vault) Model {
	return Model{
		vault:    v,
		viewMode: ViewTags,
	}
}

func (m *Model) BuildTagList() {
	m.tags = nil

	tagMap := m.vault.Tags
	for tagName, files := range tagMap {
		m.tags = append(m.tags, Tag{
			Name:  tagName,
			Count: len(files),
		})
	}

	// Sort by count (descending), then by name
	sort.Slice(m.tags, func(i, j int) bool {
		if m.tags[i].Count != m.tags[j].Count {
			return m.tags[i].Count > m.tags[j].Count
		}
		return m.tags[i].Name < m.tags[j].Name
	})
}

func (m *Model) Show() {
	m.active = true
	m.viewMode = ViewTags
	m.tagCursor = 0
	m.fileCursor = 0
	m.selectedTag = ""
	m.BuildTagList()
}

func (m *Model) Hide() {
	m.active = false
}

func (m Model) Active() bool {
	return m.active
}

func (m *Model) selectTag() {
	if m.tagCursor < len(m.tags) {
		tag := m.tags[m.tagCursor]
		m.selectedTag = tag.Name
		m.files = m.vault.GetFilesWithTag(tag.Name)
		m.fileCursor = 0
		m.viewMode = ViewFiles
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			if m.viewMode == ViewFiles {
				m.viewMode = ViewTags
				m.selectedTag = ""
				return m, nil
			}
			m.Hide()
			return m, func() tea.Msg { return TagPaneClosedMsg{} }

		case "up", "k":
			if m.viewMode == ViewTags {
				if m.tagCursor > 0 {
					m.tagCursor--
				}
			} else {
				if m.fileCursor > 0 {
					m.fileCursor--
				}
			}

		case "down", "j":
			if m.viewMode == ViewTags {
				if m.tagCursor < len(m.tags)-1 {
					m.tagCursor++
				}
			} else {
				if m.fileCursor < len(m.files)-1 {
					m.fileCursor++
				}
			}

		case "enter":
			if m.viewMode == ViewTags {
				m.selectTag()
			} else {
				if m.fileCursor < len(m.files) {
					path := m.files[m.fileCursor]
					m.Hide()
					return m, func() tea.Msg { return FileSelectedMsg{Path: path} }
				}
			}

		case "tab":
			// Toggle between tags and files
			if m.viewMode == ViewFiles {
				m.viewMode = ViewTags
			} else if m.selectedTag != "" {
				m.viewMode = ViewFiles
			}

		case "backspace", "h":
			if m.viewMode == ViewFiles {
				m.viewMode = ViewTags
			}

		case "g":
			if m.viewMode == ViewTags {
				m.tagCursor = 0
			} else {
				m.fileCursor = 0
			}

		case "G":
			if m.viewMode == ViewTags {
				m.tagCursor = len(m.tags) - 1
			} else {
				m.fileCursor = len(m.files) - 1
			}

		case "pgup", "ctrl+u":
			step := m.height / 2
			if m.viewMode == ViewTags {
				m.tagCursor -= step
				if m.tagCursor < 0 {
					m.tagCursor = 0
				}
			} else {
				m.fileCursor -= step
				if m.fileCursor < 0 {
					m.fileCursor = 0
				}
			}

		case "pgdown", "ctrl+d":
			step := m.height / 2
			if m.viewMode == ViewTags {
				m.tagCursor += step
				if m.tagCursor >= len(m.tags) {
					m.tagCursor = len(m.tags) - 1
				}
			} else {
				m.fileCursor += step
				if m.fileCursor >= len(m.files) {
					m.fileCursor = len(m.files) - 1
				}
			}
		}

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.viewMode == ViewTags {
				m.tagCursor -= 3
				if m.tagCursor < 0 {
					m.tagCursor = 0
				}
			} else {
				m.fileCursor -= 3
				if m.fileCursor < 0 {
					m.fileCursor = 0
				}
			}
		case tea.MouseButtonWheelDown:
			if m.viewMode == ViewTags {
				m.tagCursor += 3
				if m.tagCursor >= len(m.tags) {
					m.tagCursor = len(m.tags) - 1
				}
			} else {
				m.fileCursor += 3
				if m.fileCursor >= len(m.files) {
					m.fileCursor = len(m.files) - 1
				}
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

	// Title
	title := titleStyle.Render("Tags")
	b.WriteString(title + "\n")

	// Stats
	totalTags := len(m.tags)
	stats := headerStyle.Render(fmt.Sprintf("Total: %d tags | Enter: select | Tab: switch", totalTags))
	b.WriteString(stats + "\n\n")

	if m.viewMode == ViewTags {
		b.WriteString(m.renderTagList())
	} else {
		b.WriteString(m.renderFileList())
	}

	return containerStyle.Width(m.width).Render(b.String())
}

func (m Model) renderTagList() string {
	var b strings.Builder

	if len(m.tags) == 0 {
		return countStyle.Render("No tags found")
	}

	maxVisible := m.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := 0
	if m.tagCursor >= maxVisible {
		start = m.tagCursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.tags) {
		end = len(m.tags)
	}

	for i := start; i < end; i++ {
		tag := m.tags[i]

		name := "#" + tag.Name
		maxNameLen := m.width - 15
		if maxNameLen > 0 && len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		countStr := countStyle.Render(fmt.Sprintf("(%d)", tag.Count))

		var style lipgloss.Style
		if i == m.tagCursor {
			style = tagSelectedStyle
			name = "▶ " + name
		} else {
			style = tagStyle
			name = "  " + name
		}

		b.WriteString(style.Render(name) + " " + countStr + "\n")
	}

	// Scroll indicator
	if len(m.tags) > maxVisible {
		scrollInfo := countStyle.Render(fmt.Sprintf("\n[%d/%d]", m.tagCursor+1, len(m.tags)))
		b.WriteString(scrollInfo)
	}

	return b.String()
}

func (m Model) renderFileList() string {
	var b strings.Builder

	// Show selected tag
	tagHeader := tagSelectedStyle.Render("#"+m.selectedTag) + " " + countStyle.Render(fmt.Sprintf("(%d files)", len(m.files)))
	b.WriteString(tagHeader + "\n")
	b.WriteString(headerStyle.Render("Backspace/h: back to tags") + "\n\n")

	if len(m.files) == 0 {
		return b.String() + countStyle.Render("No files with this tag")
	}

	maxVisible := m.height - 12
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := 0
	if m.fileCursor >= maxVisible {
		start = m.fileCursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.files) {
		end = len(m.files)
	}

	for i := start; i < end; i++ {
		file := m.files[i]

		// Remove .md extension and truncate if needed
		name := strings.TrimSuffix(file, ".md")
		maxNameLen := m.width - 10
		if maxNameLen > 0 && len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		var style lipgloss.Style
		if i == m.fileCursor {
			style = fileSelectedStyle
			name = "▶ " + name
		} else {
			style = fileStyle
			name = "  " + name
		}

		b.WriteString(style.Render(name) + "\n")
	}

	// Scroll indicator
	if len(m.files) > maxVisible {
		scrollInfo := countStyle.Render(fmt.Sprintf("\n[%d/%d]", m.fileCursor+1, len(m.files)))
		b.WriteString(scrollInfo)
	}

	return b.String()
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}
