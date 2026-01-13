package filetree

import (
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/internal/vault"
)

var (
	selectedFocusedStyle   = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
	selectedUnfocusedStyle = lipgloss.NewStyle().Background(lipgloss.Color("240")).Foreground(lipgloss.Color("255"))
	dirStyle               = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	defaultStyle           = lipgloss.NewStyle()
	indentStrings          = []string{"", "  ", "    ", "      ", "        ", "          ", "            ", "              ", "                ", "                  "}
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
	m.FlatNodes = m.FlatNodes[:0]
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
		switch msg.String() {
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.FlatNodes)-1 {
				m.Cursor++
			}
		case "enter":
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
		case "tab", "l", "h":
			if m.Cursor < len(m.FlatNodes) {
				node := m.FlatNodes[m.Cursor]
				if node.IsDir {
					node.Expanded = !node.Expanded
					m.flattenTree()
				}
			}
		case "pgup", "ctrl+u":
			m.Cursor -= m.height / 2
			if m.Cursor < 0 {
				m.Cursor = 0
			}
		case "pgdown", "ctrl+d":
			m.Cursor += m.height / 2
			if m.Cursor >= len(m.FlatNodes) {
				m.Cursor = len(m.FlatNodes) - 1
			}
		}

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.Cursor -= 3
			if m.Cursor < 0 {
				m.Cursor = 0
			}
		case tea.MouseButtonWheelDown:
			m.Cursor += 3
			if m.Cursor >= len(m.FlatNodes) {
				m.Cursor = len(m.FlatNodes) - 1
			}
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress {
				// Calculate scroll offset same as View()
				start := 0
				if m.height > 0 && len(m.FlatNodes) > m.height {
					if m.Cursor >= m.height/2 {
						start = m.Cursor - m.height/2
					}
					if start+m.height > len(m.FlatNodes) {
						start = len(m.FlatNodes) - m.height
						if start < 0 {
							start = 0
						}
					}
				}

				clickedIndex := msg.Y + start
				if clickedIndex >= 0 && clickedIndex < len(m.FlatNodes) {
					m.Cursor = clickedIndex
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
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	b.Grow(m.width * m.height)

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
		m.renderNode(&b, node, i == m.Cursor)
		b.WriteByte('\n')
	}

	return b.String()
}

func (m Model) renderNode(b *strings.Builder, node *Node, selected bool) {
	// Cached indent
	if node.Depth < len(indentStrings) {
		b.WriteString(indentStrings[node.Depth])
	} else {
		for i := 0; i < node.Depth; i++ {
			b.WriteString("  ")
		}
	}

	// Icon
	if node.IsDir {
		if node.Expanded {
			b.WriteString("▼ ")
		} else {
			b.WriteString("▶ ")
		}
	} else {
		b.WriteString("  ")
	}

	// Name (with truncation if needed)
	name := node.Name
	maxLen := m.width - node.Depth*2 - 4
	if maxLen > 0 && len(name) > maxLen {
		name = name[:maxLen-3] + "..."
	}

	// Apply style using cached styles
	var style lipgloss.Style
	if selected {
		if m.focused {
			style = selectedFocusedStyle
		} else {
			style = selectedUnfocusedStyle
		}
	} else if node.IsDir {
		style = dirStyle
	} else {
		style = defaultStyle
	}

	b.WriteString(style.Render(name))
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
