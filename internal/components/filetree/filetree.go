package filetree

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/internal/vault"
)

type Node struct {
	Name     string
	Path     string
	IsDir    bool
	Expanded bool
	Children []*Node
	Parent   *Node
	Depth    int
}

type Model struct {
	Root         *Node
	FlatNodes    []*Node
	Cursor       int
	vault        *vault.Vault
	width        int
	height       int
	focused      bool
	selectedPath string
}

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Toggle   key.Binding
	PageUp   key.Binding
	PageDown key.Binding
}

var DefaultKeyMap = KeyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "down")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	Toggle:   key.NewBinding(key.WithKeys("tab", "l", "h"), key.WithHelp("tab", "toggle")),
	PageUp:   key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup", "page up")),
	PageDown: key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdown", "page down")),
}

type FileSelectedMsg struct {
	Path string
}

func New(v *vault.Vault) Model {
	m := Model{
		vault:   v,
		focused: true,
	}
	m.buildTree()
	return m
}

func (m *Model) buildTree() {
	m.Root = &Node{
		Name:     filepath.Base(m.vault.Path),
		Path:     "",
		IsDir:    true,
		Expanded: true,
		Depth:    0,
	}

	pathNodes := make(map[string]*Node)
	pathNodes[""] = m.Root

	var paths []string
	for path := range m.vault.Files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, relPath := range paths {
		file := m.vault.Files[relPath]
		if relPath == "." {
			continue
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		currentPath := ""

		for i, part := range parts {
			parentPath := currentPath
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = filepath.Join(currentPath, part)
			}

			if _, exists := pathNodes[currentPath]; exists {
				continue
			}

			parent := pathNodes[parentPath]
			isDir := i < len(parts)-1 || file.IsDir

			node := &Node{
				Name:     part,
				Path:     currentPath,
				IsDir:    isDir,
				Expanded: false,
				Parent:   parent,
				Depth:    parent.Depth + 1,
			}

			parent.Children = append(parent.Children, node)
			pathNodes[currentPath] = node
		}
	}

	sortNodes(m.Root)
	m.flattenTree()
}

func sortNodes(node *Node) {
	sort.Slice(node.Children, func(i, j int) bool {
		if node.Children[i].IsDir != node.Children[j].IsDir {
			return node.Children[i].IsDir
		}
		return strings.ToLower(node.Children[i].Name) < strings.ToLower(node.Children[j].Name)
	})

	for _, child := range node.Children {
		sortNodes(child)
	}
}

func (m *Model) flattenTree() {
	m.FlatNodes = nil
	m.flattenNode(m.Root)
}

func (m *Model) flattenNode(node *Node) {
	m.FlatNodes = append(m.FlatNodes, node)

	if node.Expanded {
		for _, child := range node.Children {
			m.flattenNode(child)
		}
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			if m.Cursor > 0 {
				m.Cursor--
			}
		case key.Matches(msg, DefaultKeyMap.Down):
			if m.Cursor < len(m.FlatNodes)-1 {
				m.Cursor++
			}
		case key.Matches(msg, DefaultKeyMap.Enter):
			if m.Cursor < len(m.FlatNodes) {
				node := m.FlatNodes[m.Cursor]
				if node.IsDir {
					node.Expanded = !node.Expanded
					m.flattenTree()
				} else {
					m.selectedPath = node.Path
					return m, func() tea.Msg {
						return FileSelectedMsg{Path: node.Path}
					}
				}
			}
		case key.Matches(msg, DefaultKeyMap.Toggle):
			if m.Cursor < len(m.FlatNodes) {
				node := m.FlatNodes[m.Cursor]
				if node.IsDir {
					node.Expanded = !node.Expanded
					m.flattenTree()
				}
			}
		case key.Matches(msg, DefaultKeyMap.PageUp):
			m.Cursor -= m.height / 2
			if m.Cursor < 0 {
				m.Cursor = 0
			}
		case key.Matches(msg, DefaultKeyMap.PageDown):
			m.Cursor += m.height / 2
			if m.Cursor >= len(m.FlatNodes) {
				m.Cursor = len(m.FlatNodes) - 1
			}
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if msg.Y >= 0 && msg.Y < len(m.FlatNodes) && msg.Y < m.height {
				m.Cursor = msg.Y
				node := m.FlatNodes[m.Cursor]
				if node.IsDir {
					node.Expanded = !node.Expanded
					m.flattenTree()
				} else {
					m.selectedPath = node.Path
					return m, func() tea.Msg {
						return FileSelectedMsg{Path: node.Path}
					}
				}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	start := 0
	end := len(m.FlatNodes)

	if m.height > 0 && end > m.height {
		if m.Cursor >= m.height/2 {
			start = m.Cursor - m.height/2
		}
		if start+m.height < end {
			end = start + m.height
		} else {
			start = end - m.height
			if start < 0 {
				start = 0
			}
		}
	}

	for i := start; i < end && i < len(m.FlatNodes); i++ {
		node := m.FlatNodes[i]
		line := m.renderNode(node, i == m.Cursor)
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) renderNode(node *Node, selected bool) string {
	indent := strings.Repeat("  ", node.Depth)

	var icon string
	if node.IsDir {
		if node.Expanded {
			icon = "▼ "
		} else {
			icon = "▶ "
		}
	} else {
		icon = "  "
	}

	name := node.Name
	if len(name) > m.width-len(indent)-4 && m.width > 0 {
		name = name[:m.width-len(indent)-7] + "..."
	}

	line := indent + icon + name

	style := lipgloss.NewStyle()
	if selected {
		if m.focused {
			style = style.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
		} else {
			style = style.Background(lipgloss.Color("240")).Foreground(lipgloss.Color("255"))
		}
	} else if node.IsDir {
		style = style.Foreground(lipgloss.Color("39"))
	}

	if m.width > 0 {
		line = style.Width(m.width).Render(line)
	} else {
		line = style.Render(line)
	}

	return line
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

func (m Model) Focused() bool {
	return m.focused
}

func (m Model) SelectedPath() string {
	return m.selectedPath
}

func (m *Model) Refresh() {
	m.vault.Scan()
	m.buildTree()
}
