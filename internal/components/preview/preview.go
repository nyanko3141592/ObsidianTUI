package preview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/internal/parser"
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

	return Model{
		viewport:     vp,
		selectedLink: -1,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			m.viewport.LineUp(1)
		case key.Matches(msg, DefaultKeyMap.Down):
			m.viewport.LineDown(1)
		case key.Matches(msg, DefaultKeyMap.PageUp):
			m.viewport.HalfViewUp()
		case key.Matches(msg, DefaultKeyMap.PageDown):
			m.viewport.HalfViewDown()
		case key.Matches(msg, DefaultKeyMap.Top):
			m.viewport.GotoTop()
		case key.Matches(msg, DefaultKeyMap.Bottom):
			m.viewport.GotoBottom()
		case key.Matches(msg, DefaultKeyMap.NextLink):
			if len(m.links) > 0 {
				m.selectedLink = (m.selectedLink + 1) % len(m.links)
			}
		case key.Matches(msg, DefaultKeyMap.PrevLink):
			if len(m.links) > 0 {
				m.selectedLink--
				if m.selectedLink < 0 {
					m.selectedLink = len(m.links) - 1
				}
			}
		case key.Matches(msg, DefaultKeyMap.FollowLink):
			if m.selectedLink >= 0 && m.selectedLink < len(m.links) {
				link := m.links[m.selectedLink]
				return m, func() tea.Msg {
					return LinkFollowMsg{Target: link.Target}
				}
			}
		}

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("63")).
		Width(m.width).
		Padding(0, 1)

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

	header := headerStyle.Render("PREVIEW | " + fileName + linkInfo)

	viewportStyle := lipgloss.NewStyle()
	if m.focused {
		viewportStyle = viewportStyle.BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))
	} else {
		viewportStyle = viewportStyle.BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
	}

	content := viewportStyle.Render(m.viewport.View())

	scrollInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf(" %.0f%%", m.viewport.ScrollPercent()*100))

	return lipgloss.JoinVertical(lipgloss.Left, header, content, scrollInfo)
}

func (m *Model) SetContent(content string, filePath string) {
	m.content = content
	m.filePath = filePath
	m.links = parser.ExtractWikiLinks(content)
	m.selectedLink = -1

	if m.renderer == nil {
		var err error
		m.renderer, err = parser.NewMarkdownRenderer(m.width - 4)
		if err != nil {
			m.rendered = content
		}
	}

	if m.renderer != nil {
		rendered, err := m.renderer.Render(content)
		if err != nil {
			m.rendered = content
		} else {
			m.rendered = rendered
		}
	}

	m.viewport.SetContent(m.rendered)
	m.viewport.GotoTop()
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width - 4
	m.viewport.Height = height - 6

	if m.renderer != nil && m.content != "" {
		m.renderer, _ = parser.NewMarkdownRenderer(width - 4)
		if m.renderer != nil {
			rendered, err := m.renderer.Render(m.content)
			if err == nil {
				m.rendered = rendered
				m.viewport.SetContent(m.rendered)
			}
		}
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
