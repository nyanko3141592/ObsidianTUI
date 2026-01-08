package editor

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/internal/parser"
)

type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
)

type Model struct {
	textarea     textarea.Model
	content      string
	filePath     string
	mode         Mode
	modified     bool
	width        int
	height       int
	focused      bool
	cursorLine   int
	cursorCol    int
	links        []parser.Link
	scrollOffset int
}

type KeyMap struct {
	Save       key.Binding
	Quit       key.Binding
	InsertMode key.Binding
	NormalMode key.Binding
	FollowLink key.Binding
}

var DefaultKeyMap = KeyMap{
	Save:       key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
	Quit:       key.NewBinding(key.WithKeys("ctrl+q"), key.WithHelp("ctrl+q", "quit")),
	InsertMode: key.NewBinding(key.WithKeys("i", "a"), key.WithHelp("i/a", "insert mode")),
	NormalMode: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "normal mode")),
	FollowLink: key.NewBinding(key.WithKeys("ctrl+]", "gd"), key.WithHelp("ctrl+]", "follow link")),
}

type SaveRequestMsg struct {
	Path    string
	Content string
}

type LinkFollowMsg struct {
	Target string
}

func New() Model {
	ta := textarea.New()
	ta.Placeholder = "Start typing..."
	ta.CharLimit = 0
	ta.ShowLineNumbers = true

	return Model{
		textarea: ta,
		mode:     ModeNormal,
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
		case key.Matches(msg, DefaultKeyMap.Save):
			return m, func() tea.Msg {
				return SaveRequestMsg{
					Path:    m.filePath,
					Content: m.textarea.Value(),
				}
			}

		case key.Matches(msg, DefaultKeyMap.NormalMode):
			m.mode = ModeNormal
			m.textarea.Blur()
			return m, nil

		case key.Matches(msg, DefaultKeyMap.InsertMode) && m.mode == ModeNormal:
			m.mode = ModeInsert
			m.textarea.Focus()
			return m, nil

		case key.Matches(msg, DefaultKeyMap.FollowLink) && m.mode == ModeNormal:
			link := m.getLinkAtCursor()
			if link != nil {
				return m, func() tea.Msg {
					return LinkFollowMsg{Target: link.Target}
				}
			}
		}

		if m.mode == ModeInsert {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)

			if m.textarea.Value() != m.content {
				m.modified = true
			}
		} else {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Width(m.width).
		Padding(0, 1)

	modeStr := "NORMAL"
	if m.mode == ModeInsert {
		modeStr = "INSERT"
	}

	modifiedStr := ""
	if m.modified {
		modifiedStr = " [+]"
	}

	fileName := m.filePath
	if fileName == "" {
		fileName = "No file"
	}

	header := headerStyle.Render(modeStr + " | " + fileName + modifiedStr)

	editorStyle := lipgloss.NewStyle()
	if m.focused {
		editorStyle = editorStyle.BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))
	} else {
		editorStyle = editorStyle.BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
	}

	editor := editorStyle.Render(m.textarea.View())

	return lipgloss.JoinVertical(lipgloss.Left, header, editor)
}

func (m *Model) SetContent(content string, filePath string) {
	m.content = content
	m.filePath = filePath
	m.textarea.SetValue(content)
	m.modified = false
	m.links = parser.ExtractAllLinks(content)
	m.textarea.SetCursor(0)
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textarea.SetWidth(width - 4)
	m.textarea.SetHeight(height - 4)
}

func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if focused && m.mode == ModeInsert {
		m.textarea.Focus()
	} else {
		m.textarea.Blur()
	}
}

func (m Model) Focused() bool {
	return m.focused
}

func (m Model) Content() string {
	return m.textarea.Value()
}

func (m Model) FilePath() string {
	return m.filePath
}

func (m Model) Modified() bool {
	return m.modified
}

func (m *Model) SetModified(modified bool) {
	m.modified = modified
}

func (m Model) Mode() Mode {
	return m.mode
}

func (m *Model) getLinkAtCursor() *parser.Link {
	if len(m.links) > 0 {
		return &m.links[0]
	}
	return nil
}
