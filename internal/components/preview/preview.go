package preview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/internal/parser"
	"github.com/takahashinaoki/obsidiantui/internal/vault"
)

var (
	previewHeaderBase = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("63")).
				Padding(0, 1)
	previewFocusedBorder = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))
	previewUnfocusedBorder = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))
	previewScrollStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	embedHeaderStyle   = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57")).
				Padding(0, 1)
	embedBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("57")).
				Padding(0, 1).
				MarginLeft(2)
	embedErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Italic(true)
)

type Model struct {
	viewport     viewport.Model
	content      string
	rendered     string
	filePath     string
	width        int
	height       int
	focused      bool
	renderer     *parser.MarkdownRenderer
	links        []parser.Link
	selectedLink int
	vault        *vault.Vault
	maxEmbedDepth int
}

type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Top        key.Binding
	Bottom     key.Binding
	NextLink   key.Binding
	PrevLink   key.Binding
	FollowLink key.Binding
}

var DefaultKeyMap = KeyMap{
	Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "scroll up")),
	Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "scroll down")),
	PageUp:     key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup", "page up")),
	PageDown:   key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdown", "page down")),
	Top:        key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
	Bottom:     key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
	NextLink:   key.NewBinding(key.WithKeys("tab", "n"), key.WithHelp("tab", "next link")),
	PrevLink:   key.NewBinding(key.WithKeys("shift+tab", "N"), key.WithHelp("shift+tab", "prev link")),
	FollowLink: key.NewBinding(key.WithKeys("enter", "ctrl+]"), key.WithHelp("enter", "follow link")),
}

type LinkFollowMsg struct {
	Target string
}

func New() Model {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle()
	vp.MouseWheelEnabled = true

	return Model{
		viewport:      vp,
		selectedLink:  -1,
		maxEmbedDepth: 3, // prevent infinite recursion
	}
}

func (m *Model) SetVault(v *vault.Vault) {
	m.vault = v
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			m.viewport.LineUp(1)
			return m, nil
		case key.Matches(msg, DefaultKeyMap.Down):
			m.viewport.LineDown(1)
			return m, nil
		case key.Matches(msg, DefaultKeyMap.PageUp):
			m.viewport.HalfViewUp()
			return m, nil
		case key.Matches(msg, DefaultKeyMap.PageDown):
			m.viewport.HalfViewDown()
			return m, nil
		case key.Matches(msg, DefaultKeyMap.Top):
			m.viewport.GotoTop()
			return m, nil
		case key.Matches(msg, DefaultKeyMap.Bottom):
			m.viewport.GotoBottom()
			return m, nil
		case key.Matches(msg, DefaultKeyMap.NextLink):
			if len(m.links) > 0 {
				m.selectedLink = (m.selectedLink + 1) % len(m.links)
			}
			return m, nil
		case key.Matches(msg, DefaultKeyMap.PrevLink):
			if len(m.links) > 0 {
				m.selectedLink--
				if m.selectedLink < 0 {
					m.selectedLink = len(m.links) - 1
				}
			}
			return m, nil
		case key.Matches(msg, DefaultKeyMap.FollowLink):
			if m.selectedLink >= 0 && m.selectedLink < len(m.links) {
				link := m.links[m.selectedLink]
				return m, func() tea.Msg {
					return LinkFollowMsg{Target: link.Target}
				}
			}
			return m, nil
		}

	case tea.MouseMsg:
		// Handle mouse wheel directly
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.viewport.LineUp(3)
			return m, nil
		case tea.MouseButtonWheelDown:
			m.viewport.LineDown(3)
			return m, nil
		}
		// Let viewport handle other mouse events
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	fileName := m.filePath
	if fileName == "" {
		fileName = "No file"
	}

	linkInfo := ""
	if len(m.links) > 0 {
		linkInfo = " | Links: " + string(rune('0'+len(m.links)))
		if m.selectedLink >= 0 {
			linkInfo += " [" + m.links[m.selectedLink].DisplayText + "]"
		}
	}

	header := previewHeaderBase.Width(m.width).Render("PREVIEW | " + fileName + linkInfo)

	var viewportStyle lipgloss.Style
	if m.focused {
		viewportStyle = previewFocusedBorder
	} else {
		viewportStyle = previewUnfocusedBorder
	}

	content := viewportStyle.Render(m.viewport.View())

	scrollInfo := previewScrollStyle.Render(fmt.Sprintf(" %.0f%%", m.viewport.ScrollPercent()*100))

	return lipgloss.JoinVertical(lipgloss.Left, header, content, scrollInfo)
}

func (m *Model) SetContent(content string, filePath string) {
	m.content = content
	m.filePath = filePath
	m.links = parser.ExtractWikiLinks(content)
	m.selectedLink = -1

	m.renderContent()
	m.viewport.GotoTop()
}

func (m *Model) renderContent() {
	width := m.width - 4
	if width < 40 {
		width = 40
	}

	// Expand embedded notes before rendering
	contentWithEmbeds := m.expandEmbeds(m.content, 0, make(map[string]bool))

	var err error
	m.renderer, err = parser.NewMarkdownRenderer(width)
	if err != nil {
		m.rendered = contentWithEmbeds
		m.viewport.SetContent(m.rendered)
		return
	}

	rendered, err := m.renderer.Render(contentWithEmbeds)
	if err != nil {
		m.rendered = contentWithEmbeds
	} else {
		m.rendered = rendered
	}
	m.viewport.SetContent(m.rendered)
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width - 4
	m.viewport.Height = height - 6

	if m.content != "" {
		m.renderContent()
	}
}

func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

func (m Model) Focused() bool {
	return m.focused
}

func (m Model) Content() string {
	return m.content
}

func (m Model) FilePath() string {
	return m.filePath
}

func (m *Model) GetSelectedLink() *parser.Link {
	if m.selectedLink >= 0 && m.selectedLink < len(m.links) {
		return &m.links[m.selectedLink]
	}
	return nil
}

func (m Model) Links() []parser.Link {
	return m.links
}

func (m *Model) ScrollToLine(line int) {
	lines := strings.Split(m.rendered, "\n")
	if line < len(lines) {
		m.viewport.SetYOffset(line)
	}
}

// expandEmbeds replaces ![[note]] with the embedded note content
func (m *Model) expandEmbeds(content string, depth int, seen map[string]bool) string {
	if m.vault == nil || depth >= m.maxEmbedDepth {
		return content
	}

	embeds := parser.ExtractEmbedLinks(content)
	if len(embeds) == 0 {
		return content
	}

	// Process embeds from end to start to preserve positions
	result := content
	for i := len(embeds) - 1; i >= 0; i-- {
		embed := embeds[i]

		// Try to resolve the embed target
		target := embed.Target
		resolvedPath := m.vault.FindFile(target + ".md")
		if resolvedPath == "" {
			resolvedPath = m.vault.FindFile(target)
		}

		var replacement string
		if resolvedPath == "" {
			// Broken embed
			replacement = fmt.Sprintf("\n> **⚠ Embed not found: %s**\n", target)
		} else if seen[resolvedPath] {
			// Circular reference
			replacement = fmt.Sprintf("\n> **⚠ Circular embed: %s**\n", target)
		} else {
			// Read the embedded file
			file, ok := m.vault.Files[resolvedPath]
			if !ok || file.Content == "" {
				replacement = fmt.Sprintf("\n> **⚠ Cannot read: %s**\n", target)
			} else {
				// Mark as seen to prevent cycles
				seen[resolvedPath] = true

				// Recursively expand embeds in the embedded content
				embeddedContent := m.expandEmbeds(file.Content, depth+1, seen)

				// Format with visual indicator
				displayName := target
				if embed.AltText != "" {
					displayName = embed.AltText
				}

				// Create a blockquote-style embed
				lines := strings.Split(embeddedContent, "\n")
				var quotedLines []string
				quotedLines = append(quotedLines, fmt.Sprintf("\n---\n**📎 %s**\n", displayName))
				for _, line := range lines {
					quotedLines = append(quotedLines, "> "+line)
				}
				quotedLines = append(quotedLines, "\n---\n")

				replacement = strings.Join(quotedLines, "\n")
			}
		}

		// Replace the embed with the replacement text
		result = result[:embed.StartPos] + replacement + result[embed.EndPos:]
	}

	return result
}
