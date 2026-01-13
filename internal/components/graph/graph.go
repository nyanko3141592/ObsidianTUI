package graph

import (
	"fmt"
	"math"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/internal/vault"
)

var (
	nodeFocusedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	nodeSelectedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	nodeNormalStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	nodeOrphanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	edgeStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	titleStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("93")).Padding(0, 1)
	containerStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("93")).Padding(1)
	statsStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	linkCountStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
)

// Node represents a file in the graph
type Node struct {
	ID         string  // relative path
	Name       string  // display name
	X, Y       float64 // position for force-directed layout
	VX, VY     float64 // velocity
	Links      int     // outgoing links count
	Backlinks  int     // incoming links count
	IsOrphan   bool    // no links at all
	IsFocused  bool    // currently focused node
}

// Edge represents a link between two files
type Edge struct {
	Source string
	Target string
}

// Model is the graph view model
type Model struct {
	vault       *vault.Vault
	nodes       []*Node
	nodeMap     map[string]*Node
	edges       []Edge
	cursor      int
	width       int
	height      int
	active      bool
	focusedFile string
	viewMode    ViewMode
	offsetX     int
	offsetY     int
}

type ViewMode int

const (
	ViewList ViewMode = iota
	ViewLocal
	ViewCanvas
)

type FileSelectedMsg struct {
	Path string
}

type GraphClosedMsg struct{}

func New(v *vault.Vault) Model {
	return Model{
		vault:    v,
		nodeMap:  make(map[string]*Node),
		viewMode: ViewList,
	}
}

func (m *Model) BuildGraph() {
	m.nodes = nil
	m.nodeMap = make(map[string]*Node)
	m.edges = nil

	files := m.vault.Files

	// Create nodes for all markdown files
	for relPath, file := range files {
		if file.IsDir || !strings.HasSuffix(strings.ToLower(file.Name), ".md") {
			continue
		}

		node := &Node{
			ID:        relPath,
			Name:      strings.TrimSuffix(file.Name, ".md"),
			Links:     len(file.Links),
			Backlinks: len(m.vault.GetBacklinks(relPath)),
		}
		node.IsOrphan = node.Links == 0 && node.Backlinks == 0

		m.nodes = append(m.nodes, node)
		m.nodeMap[relPath] = node
	}

	// Sort nodes by connection count (most connected first)
	sort.Slice(m.nodes, func(i, j int) bool {
		totalI := m.nodes[i].Links + m.nodes[i].Backlinks
		totalJ := m.nodes[j].Links + m.nodes[j].Backlinks
		if totalI != totalJ {
			return totalI > totalJ
		}
		return m.nodes[i].Name < m.nodes[j].Name
	})

	// Build edges
	for relPath, file := range files {
		if file.IsDir {
			continue
		}
		for _, link := range file.Links {
			if link.IsWikiLink {
				targetPath := m.vault.FindFile(link.Target + ".md")
				if targetPath == "" {
					targetPath = m.vault.FindFile(link.Target)
				}
				if targetPath != "" && targetPath != relPath {
					m.edges = append(m.edges, Edge{Source: relPath, Target: targetPath})
				}
			}
		}
	}

	// Initialize positions for canvas view
	m.initializePositions()
}

func (m *Model) initializePositions() {
	n := len(m.nodes)
	if n == 0 {
		return
	}

	// Circular layout
	centerX := float64(m.width) / 2
	centerY := float64(m.height) / 2
	radius := math.Min(centerX, centerY) * 0.8

	for i, node := range m.nodes {
		angle := 2 * math.Pi * float64(i) / float64(n)
		node.X = centerX + radius*math.Cos(angle)
		node.Y = centerY + radius*math.Sin(angle)
	}
}

func (m *Model) Show(focusedFile string) {
	m.active = true
	m.focusedFile = focusedFile
	m.cursor = 0
	m.BuildGraph()

	// Find focused file in node list
	if focusedFile != "" {
		for i, node := range m.nodes {
			if node.ID == focusedFile {
				m.cursor = i
				node.IsFocused = true
				break
			}
		}
	}
}

func (m *Model) Hide() {
	m.active = false
	for _, node := range m.nodes {
		node.IsFocused = false
	}
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
		case "esc", "q", "ctrl+g":
			m.Hide()
			return m, func() tea.Msg { return GraphClosedMsg{} }

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.nodes)-1 {
				m.cursor++
			}

		case "enter":
			if m.cursor < len(m.nodes) {
				path := m.nodes[m.cursor].ID
				m.Hide()
				return m, func() tea.Msg { return FileSelectedMsg{Path: path} }
			}

		case "tab":
			// Cycle view mode
			m.viewMode = (m.viewMode + 1) % 3

		case "l":
			// Switch to local graph of current selection
			if m.cursor < len(m.nodes) {
				m.focusedFile = m.nodes[m.cursor].ID
				m.viewMode = ViewLocal
			}

		case "g":
			// Go to top
			m.cursor = 0

		case "G":
			// Go to bottom
			m.cursor = len(m.nodes) - 1

		case "pgup", "ctrl+u":
			m.cursor -= m.height / 2
			if m.cursor < 0 {
				m.cursor = 0
			}

		case "pgdown", "ctrl+d":
			m.cursor += m.height / 2
			if m.cursor >= len(m.nodes) {
				m.cursor = len(m.nodes) - 1
			}
		}

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.cursor > 0 {
				m.cursor -= 3
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		case tea.MouseButtonWheelDown:
			if m.cursor < len(m.nodes)-1 {
				m.cursor += 3
				if m.cursor >= len(m.nodes) {
					m.cursor = len(m.nodes) - 1
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
	modeStr := "List"
	switch m.viewMode {
	case ViewLocal:
		modeStr = "Local"
	case ViewCanvas:
		modeStr = "Canvas"
	}
	title := titleStyle.Render(fmt.Sprintf("Graph View [%s]", modeStr))
	b.WriteString(title + "\n")

	// Stats
	totalNodes := len(m.nodes)
	totalEdges := len(m.edges)
	orphans := 0
	for _, n := range m.nodes {
		if n.IsOrphan {
			orphans++
		}
	}
	stats := statsStyle.Render(fmt.Sprintf("Nodes: %d | Edges: %d | Orphans: %d | Tab: change view", totalNodes, totalEdges, orphans))
	b.WriteString(stats + "\n\n")

	switch m.viewMode {
	case ViewList:
		b.WriteString(m.renderListView())
	case ViewLocal:
		b.WriteString(m.renderLocalView())
	case ViewCanvas:
		b.WriteString(m.renderCanvasView())
	}

	return containerStyle.Width(m.width).Render(b.String())
}

func (m Model) renderListView() string {
	var b strings.Builder

	if len(m.nodes) == 0 {
		return nodeOrphanStyle.Render("No files found")
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
	if end > len(m.nodes) {
		end = len(m.nodes)
	}

	for i := start; i < end; i++ {
		node := m.nodes[i]

		// Connection indicator
		connStr := linkCountStyle.Render(fmt.Sprintf("[%d↗ %d↙]", node.Links, node.Backlinks))

		// Node name
		name := node.Name
		maxNameLen := m.width - 25
		if maxNameLen > 0 && len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		var style lipgloss.Style
		if i == m.cursor {
			style = nodeFocusedStyle
			name = "▶ " + name
		} else if node.IsFocused {
			style = nodeSelectedStyle
			name = "● " + name
		} else if node.IsOrphan {
			style = nodeOrphanStyle
			name = "○ " + name
		} else {
			style = nodeNormalStyle
			name = "  " + name
		}

		b.WriteString(style.Render(name) + " " + connStr + "\n")
	}

	// Scroll indicator
	if len(m.nodes) > maxVisible {
		scrollInfo := statsStyle.Render(fmt.Sprintf("\n[%d/%d]", m.cursor+1, len(m.nodes)))
		b.WriteString(scrollInfo)
	}

	return b.String()
}

func (m Model) renderLocalView() string {
	var b strings.Builder

	if m.focusedFile == "" || len(m.nodes) == 0 {
		return nodeOrphanStyle.Render("No file selected")
	}

	centerNode := m.nodeMap[m.focusedFile]
	if centerNode == nil {
		return nodeOrphanStyle.Render("File not found")
	}

	// Get connected nodes
	outgoing := make(map[string]bool)
	incoming := make(map[string]bool)

	for _, edge := range m.edges {
		if edge.Source == m.focusedFile {
			outgoing[edge.Target] = true
		}
		if edge.Target == m.focusedFile {
			incoming[edge.Source] = true
		}
	}

	// Render center node
	b.WriteString(nodeFocusedStyle.Render("◉ " + centerNode.Name) + "\n")
	b.WriteString(edgeStyle.Render("│") + "\n")

	// Render outgoing links
	if len(outgoing) > 0 {
		b.WriteString(edgeStyle.Render("├─→ ") + nodeSelectedStyle.Render("Links to:") + "\n")
		i := 0
		for targetID := range outgoing {
			node := m.nodeMap[targetID]
			if node != nil {
				prefix := "│   ├─"
				if i == len(outgoing)-1 {
					prefix = "│   └─"
				}
				name := node.Name
				if len(name) > m.width-15 {
					name = name[:m.width-18] + "..."
				}
				b.WriteString(edgeStyle.Render(prefix) + " " + nodeNormalStyle.Render(name) + "\n")
				i++
			}
		}
	}

	// Render incoming links (backlinks)
	if len(incoming) > 0 {
		b.WriteString(edgeStyle.Render("│") + "\n")
		b.WriteString(edgeStyle.Render("└─← ") + nodeSelectedStyle.Render("Linked from:") + "\n")
		i := 0
		for sourceID := range incoming {
			node := m.nodeMap[sourceID]
			if node != nil {
				prefix := "    ├─"
				if i == len(incoming)-1 {
					prefix = "    └─"
				}
				name := node.Name
				if len(name) > m.width-15 {
					name = name[:m.width-18] + "..."
				}
				b.WriteString(edgeStyle.Render(prefix) + " " + nodeNormalStyle.Render(name) + "\n")
				i++
			}
		}
	}

	if len(outgoing) == 0 && len(incoming) == 0 {
		b.WriteString(nodeOrphanStyle.Render("  (orphan - no connections)"))
	}

	b.WriteString("\n" + statsStyle.Render("Press 'l' on list view to focus a node"))

	return b.String()
}

func (m Model) renderCanvasView() string {
	if len(m.nodes) == 0 {
		return nodeOrphanStyle.Render("No files found")
	}

	// Create a simple ASCII canvas
	canvasWidth := m.width - 6
	canvasHeight := m.height - 12
	if canvasWidth < 20 {
		canvasWidth = 20
	}
	if canvasHeight < 10 {
		canvasHeight = 10
	}

	// Initialize canvas
	canvas := make([][]rune, canvasHeight)
	for y := range canvas {
		canvas[y] = make([]rune, canvasWidth)
		for x := range canvas[y] {
			canvas[y][x] = ' '
		}
	}

	// Place nodes on canvas using force-directed-like positions
	nodePositions := make(map[string][2]int)
	n := len(m.nodes)

	for i, node := range m.nodes {
		// Simple circular layout
		angle := 2 * math.Pi * float64(i) / float64(n)
		cx := float64(canvasWidth) / 2
		cy := float64(canvasHeight) / 2
		radius := math.Min(cx, cy) * 0.7

		x := int(cx + radius*math.Cos(angle))
		y := int(cy + radius*math.Sin(angle))

		// Clamp to canvas bounds
		if x < 1 {
			x = 1
		}
		if x >= canvasWidth-1 {
			x = canvasWidth - 2
		}
		if y < 0 {
			y = 0
		}
		if y >= canvasHeight {
			y = canvasHeight - 1
		}

		nodePositions[node.ID] = [2]int{x, y}

		// Draw node
		symbol := '●'
		if node.IsFocused || i == m.cursor {
			symbol = '◉'
		} else if node.IsOrphan {
			symbol = '○'
		}
		canvas[y][x] = symbol
	}

	// Draw edges (simple direct lines)
	for _, edge := range m.edges {
		srcPos, srcOK := nodePositions[edge.Source]
		tgtPos, tgtOK := nodePositions[edge.Target]
		if srcOK && tgtOK {
			m.drawLine(canvas, srcPos[0], srcPos[1], tgtPos[0], tgtPos[1])
		}
	}

	// Convert canvas to string
	var b strings.Builder
	for _, row := range canvas {
		b.WriteString(edgeStyle.Render(string(row)) + "\n")
	}

	// Show focused node info
	if m.cursor < len(m.nodes) {
		node := m.nodes[m.cursor]
		info := fmt.Sprintf("\n%s [%d↗ %d↙]", node.Name, node.Links, node.Backlinks)
		b.WriteString(nodeFocusedStyle.Render(info))
	}

	return b.String()
}

func (m Model) drawLine(canvas [][]rune, x1, y1, x2, y2 int) {
	// Bresenham's line algorithm
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	if x1 > x2 {
		sx = -1
	}
	sy := 1
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy

	for {
		if x1 >= 0 && x1 < len(canvas[0]) && y1 >= 0 && y1 < len(canvas) {
			if canvas[y1][x1] == ' ' {
				// Choose line character based on direction
				if dx > dy*2 {
					canvas[y1][x1] = '─'
				} else if dy > dx*2 {
					canvas[y1][x1] = '│'
				} else {
					canvas[y1][x1] = '·'
				}
			}
		}

		if x1 == x2 && y1 == y2 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) GetNodePath(index int) string {
	if index >= 0 && index < len(m.nodes) {
		return m.nodes[index].ID
	}
	return ""
}
