package liveeditor

import (
	"strconv"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/internal/parser"
)

type Model struct {
	lines       []string
	cursorRow   int
	cursorCol   int
	offsetRow   int
	filePath    string
	modified    bool
	width       int
	height      int
	focused     bool
	links       []parser.Link
	insertMode  bool
	styledCache map[int]string
	cacheValid  map[int]bool
}

var (
	headerStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	linkStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	codeStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	tagStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("135"))
	mathStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	bulletStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	blockquoteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	cursorStyle     = lipgloss.NewStyle().Reverse(true)
	lineNumStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	normalHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 1)
	insertHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("34")).Padding(0, 1)
)

var indentCache = make([]string, 20)

type SaveRequestMsg struct {
	Path    string
	Content string
}

type LinkFollowMsg struct {
	Target string
}

func New() Model {
	return Model{
		lines:       []string{""},
		insertMode:  false,
		styledCache: make(map[int]string),
		cacheValid:  make(map[int]bool),
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
		if msg.String() == "ctrl+s" {
			return m, func() tea.Msg {
				return SaveRequestMsg{Path: m.filePath, Content: m.Content()}
			}
		}

		if msg.String() == "esc" {
			m.insertMode = false
			return m, nil
		}

		if !m.insertMode {
			return m.handleNormalMode(msg)
		}
		return m.handleInsertMode(msg)

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			for i := 0; i < 3; i++ {
				m.moveCursorUp()
			}
		case tea.MouseButtonWheelDown:
			for i := 0; i < 3; i++ {
				m.moveCursorDown()
			}
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress {
				// Y=0 is header, Y=1+ are content lines
				clickRow := msg.Y - 1 + m.offsetRow
				if clickRow >= 0 && clickRow < len(m.lines) {
					m.cursorRow = clickRow
					// X: subtract line number width (4 digits + 1 space = 5)
					col := msg.X - 5
					if col < 0 {
						col = 0
					}
					lineLen := utf8.RuneCountInString(m.lines[m.cursorRow])
					if col > lineLen {
						col = lineLen
					}
					m.cursorCol = col
					m.ensureCursorVisible()
				}
			}
		}
	}

	return m, nil
}

func (m Model) handleNormalMode(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "i":
		m.insertMode = true
	case "a":
		m.insertMode = true
		if m.cursorCol < utf8.RuneCountInString(m.currentLine()) {
			m.cursorCol++
		}
	case "I":
		m.insertMode = true
		m.cursorCol = 0
	case "A":
		m.insertMode = true
		m.cursorCol = utf8.RuneCountInString(m.currentLine())
	case "o":
		m.insertLineBelow()
		m.insertMode = true
	case "O":
		m.insertLineAbove()
		m.insertMode = true
	case "up", "k":
		m.moveCursorUp()
	case "down", "j":
		m.moveCursorDown()
	case "left", "h":
		m.moveCursorLeft()
	case "right", "l":
		m.moveCursorRight()
	case "home", "0":
		m.cursorCol = 0
	case "end", "$":
		m.cursorCol = utf8.RuneCountInString(m.currentLine())
	case "ctrl+u", "pgup":
		for i := 0; i < m.height/2; i++ {
			m.moveCursorUp()
		}
	case "ctrl+d", "pgdown":
		for i := 0; i < m.height/2; i++ {
			m.moveCursorDown()
		}
	case "g":
		m.cursorRow, m.cursorCol, m.offsetRow = 0, 0, 0
	case "G":
		m.cursorRow = len(m.lines) - 1
		m.cursorCol = 0
		m.ensureCursorVisible()
	case "w":
		m.moveWordForward()
	case "b":
		m.moveWordBackward()
	case "x":
		m.deleteChar()
	case "enter", "ctrl+]":
		if link := m.getLinkAtCursor(); link != nil {
			return m, func() tea.Msg { return LinkFollowMsg{Target: link.Target} }
		}
	}
	return m, nil
}

func (m Model) handleInsertMode(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		m.moveCursorUp()
	case "down":
		m.moveCursorDown()
	case "left":
		m.moveCursorLeft()
	case "right":
		m.moveCursorRight()
	case "enter":
		m.insertNewline()
	case "backspace":
		m.backspace()
	case "delete":
		m.deleteChar()
	case "home":
		m.cursorCol = 0
	case "end":
		m.cursorCol = utf8.RuneCountInString(m.currentLine())
	default:
		if msg.Type == tea.KeyRunes {
			m.insertRunes(msg.Runes)
		}
	}
	return m, nil
}

func (m *Model) invalidateCache(row int) {
	m.cacheValid[row] = false
}

func (m *Model) invalidateAllCache() {
	m.cacheValid = make(map[int]bool)
}

func (m *Model) moveCursorUp() {
	if m.cursorRow > 0 {
		m.cursorRow--
		if lineLen := utf8.RuneCountInString(m.lines[m.cursorRow]); m.cursorCol > lineLen {
			m.cursorCol = lineLen
		}
		m.ensureCursorVisible()
	}
}

func (m *Model) moveCursorDown() {
	if m.cursorRow < len(m.lines)-1 {
		m.cursorRow++
		if lineLen := utf8.RuneCountInString(m.lines[m.cursorRow]); m.cursorCol > lineLen {
			m.cursorCol = lineLen
		}
		m.ensureCursorVisible()
	}
}

func (m *Model) moveCursorLeft() {
	if m.cursorCol > 0 {
		m.cursorCol--
	} else if m.cursorRow > 0 {
		m.cursorRow--
		m.cursorCol = utf8.RuneCountInString(m.lines[m.cursorRow])
	}
}

func (m *Model) moveCursorRight() {
	lineLen := utf8.RuneCountInString(m.currentLine())
	if m.cursorCol < lineLen {
		m.cursorCol++
	} else if m.cursorRow < len(m.lines)-1 {
		m.cursorRow++
		m.cursorCol = 0
	}
}

func (m *Model) moveWordForward() {
	line := []rune(m.currentLine())
	col := m.cursorCol
	for col < len(line) && !isWordChar(line[col]) {
		col++
	}
	for col < len(line) && isWordChar(line[col]) {
		col++
	}
	if col >= len(line) && m.cursorRow < len(m.lines)-1 {
		m.cursorRow++
		m.cursorCol = 0
	} else {
		m.cursorCol = col
	}
}

func (m *Model) moveWordBackward() {
	if m.cursorCol == 0 && m.cursorRow > 0 {
		m.cursorRow--
		m.cursorCol = utf8.RuneCountInString(m.lines[m.cursorRow])
		return
	}
	line := []rune(m.currentLine())
	col := m.cursorCol
	if col > 0 {
		col--
	}
	for col > 0 && !isWordChar(line[col]) {
		col--
	}
	for col > 0 && isWordChar(line[col-1]) {
		col--
	}
	m.cursorCol = col
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func (m *Model) insertRunes(runes []rune) {
	line := []rune(m.currentLine())
	newLine := make([]rune, 0, len(line)+len(runes))
	newLine = append(newLine, line[:m.cursorCol]...)
	newLine = append(newLine, runes...)
	newLine = append(newLine, line[m.cursorCol:]...)
	m.lines[m.cursorRow] = string(newLine)
	m.cursorCol += len(runes)
	m.modified = true
	m.invalidateCache(m.cursorRow)
}

func (m *Model) insertNewline() {
	line := []rune(m.currentLine())
	before, after := string(line[:m.cursorCol]), string(line[m.cursorCol:])
	m.lines[m.cursorRow] = before
	m.lines = append(m.lines[:m.cursorRow+1], append([]string{after}, m.lines[m.cursorRow+1:]...)...)
	m.cursorRow++
	m.cursorCol = 0
	m.modified = true
	m.invalidateAllCache()
	m.ensureCursorVisible()
}

func (m *Model) backspace() {
	if m.cursorCol > 0 {
		line := []rune(m.currentLine())
		m.lines[m.cursorRow] = string(line[:m.cursorCol-1]) + string(line[m.cursorCol:])
		m.cursorCol--
		m.modified = true
		m.invalidateCache(m.cursorRow)
	} else if m.cursorRow > 0 {
		prevLine := m.lines[m.cursorRow-1]
		m.cursorCol = utf8.RuneCountInString(prevLine)
		m.lines[m.cursorRow-1] = prevLine + m.currentLine()
		m.lines = append(m.lines[:m.cursorRow], m.lines[m.cursorRow+1:]...)
		m.cursorRow--
		m.modified = true
		m.invalidateAllCache()
	}
}

func (m *Model) deleteChar() {
	line := []rune(m.currentLine())
	if m.cursorCol < len(line) {
		m.lines[m.cursorRow] = string(line[:m.cursorCol]) + string(line[m.cursorCol+1:])
		m.modified = true
		m.invalidateCache(m.cursorRow)
	} else if m.cursorRow < len(m.lines)-1 {
		m.lines[m.cursorRow] = m.currentLine() + m.lines[m.cursorRow+1]
		m.lines = append(m.lines[:m.cursorRow+1], m.lines[m.cursorRow+2:]...)
		m.modified = true
		m.invalidateAllCache()
	}
}

func (m *Model) insertLineBelow() {
	m.lines = append(m.lines[:m.cursorRow+1], append([]string{""}, m.lines[m.cursorRow+1:]...)...)
	m.cursorRow++
	m.cursorCol = 0
	m.modified = true
	m.invalidateAllCache()
	m.ensureCursorVisible()
}

func (m *Model) insertLineAbove() {
	m.lines = append(m.lines[:m.cursorRow], append([]string{""}, m.lines[m.cursorRow:]...)...)
	m.cursorCol = 0
	m.modified = true
	m.invalidateAllCache()
	m.ensureCursorVisible()
}

func (m *Model) ensureCursorVisible() {
	visibleHeight := m.height - 2
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	if m.cursorRow < m.offsetRow {
		m.offsetRow = m.cursorRow
	} else if m.cursorRow >= m.offsetRow+visibleHeight {
		m.offsetRow = m.cursorRow - visibleHeight + 1
	}
}

func (m *Model) currentLine() string {
	if m.cursorRow < len(m.lines) {
		return m.lines[m.cursorRow]
	}
	return ""
}

func (m Model) View() string {
	var b strings.Builder
	b.Grow(m.width * m.height)

	// Header - use cached styles
	mod := ""
	if m.modified {
		mod = " [+]"
	}
	file := m.filePath
	if file == "" {
		file = "No file"
	}

	if m.insertMode {
		b.WriteString(insertHeader.Render("INSERT | " + file + mod))
	} else {
		b.WriteString(normalHeader.Render("NORMAL | " + file + mod))
	}
	b.WriteByte('\n')

	// Lines
	visibleHeight := m.height - 2
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	for i := 0; i < visibleHeight; i++ {
		lineNum := m.offsetRow + i
		if lineNum >= len(m.lines) {
			b.WriteString("~   \n")
			continue
		}

		// Line number - use strconv
		n := lineNum + 1
		if n < 10 {
			b.WriteString("   ")
			b.WriteString(strconv.Itoa(n))
		} else if n < 100 {
			b.WriteString("  ")
			b.WriteString(strconv.Itoa(n))
		} else if n < 1000 {
			b.WriteByte(' ')
			b.WriteString(strconv.Itoa(n))
		} else {
			b.WriteString(strconv.Itoa(n))
		}
		b.WriteByte(' ')

		// Line content
		line := m.lines[lineNum]
		if lineNum == m.cursorRow && m.focused {
			b.WriteString(m.renderLineWithCursor(line))
		} else {
			b.WriteString(m.getStyledLine(lineNum, line))
		}
		b.WriteByte('\n')
	}

	return b.String()
}

func (m *Model) getStyledLine(lineNum int, line string) string {
	if m.cacheValid[lineNum] {
		return m.styledCache[lineNum]
	}
	styled := m.styleLine(line)
	m.styledCache[lineNum] = styled
	m.cacheValid[lineNum] = true
	return styled
}

func (m Model) renderLineWithCursor(line string) string {
	if line == "" {
		return cursorStyle.Render(" ")
	}

	runes := []rune(line)
	col := m.cursorCol
	if col > len(runes) {
		col = len(runes)
	}

	if col < len(runes) {
		before := m.styleLine(string(runes[:col]))
		cursor := cursorStyle.Render(string(runes[col]))
		after := m.styleLine(string(runes[col+1:]))
		return before + cursor + after
	}
	return m.styleLine(line) + cursorStyle.Render(" ")
}

func (m Model) styleLine(line string) string {
	if line == "" {
		return ""
	}

	// Fast path for common cases
	if line[0] == '#' {
		return headerStyle.Render(line)
	}
	if len(line) > 1 && line[0] == '>' && line[1] == ' ' {
		return blockquoteStyle.Render(line)
	}
	if len(line) > 1 && (line[0] == '-' || line[0] == '*') && line[1] == ' ' {
		return bulletStyle.Render(line[:2]) + m.styleInline(line[2:])
	}
	if len(line) >= 3 && line[0] == '`' && line[1] == '`' && line[2] == '`' {
		return codeStyle.Render(line)
	}

	return m.styleInline(line)
}

func (m Model) styleInline(text string) string {
	if len(text) == 0 {
		return ""
	}

	// Quick check if any styling needed
	if !strings.ContainsAny(text, "*_`[$#") {
		return text
	}

	var result strings.Builder
	result.Grow(len(text) * 2)
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		// Wiki links [[...]]
		if i+1 < len(runes) && runes[i] == '[' && runes[i+1] == '[' {
			end := findClosing(runes, i+2, ']', ']')
			if end != -1 {
				result.WriteString(linkStyle.Render(string(runes[i : end+2])))
				i = end + 2
				continue
			}
		}

		// Inline code `...`
		if runes[i] == '`' {
			end := findSingle(runes, i+1, '`')
			if end != -1 {
				result.WriteString(codeStyle.Render(string(runes[i : end+1])))
				i = end + 1
				continue
			}
		}

		// Math $...$
		if runes[i] == '$' {
			end := findSingle(runes, i+1, '$')
			if end != -1 {
				result.WriteString(mathStyle.Render(string(runes[i : end+1])))
				i = end + 1
				continue
			}
		}

		// Tags #tag
		if runes[i] == '#' && (i == 0 || runes[i-1] == ' ') {
			end := i + 1
			for end < len(runes) && isTagChar(runes[end]) {
				end++
			}
			if end > i+1 {
				result.WriteString(tagStyle.Render(string(runes[i:end])))
				i = end
				continue
			}
		}

		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

func findClosing(runes []rune, start int, c1, c2 rune) int {
	for i := start; i < len(runes)-1; i++ {
		if runes[i] == c1 && runes[i+1] == c2 {
			return i
		}
	}
	return -1
}

func findSingle(runes []rune, start int, c rune) int {
	for i := start; i < len(runes); i++ {
		if runes[i] == c {
			return i
		}
	}
	return -1
}

func isTagChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '/'
}

func (m *Model) SetContent(content string, filePath string) {
	m.lines = strings.Split(content, "\n")
	if len(m.lines) == 0 {
		m.lines = []string{""}
	}
	m.filePath = filePath
	m.cursorRow, m.cursorCol, m.offsetRow = 0, 0, 0
	m.modified = false
	m.links = parser.ExtractAllLinks(content)
	m.invalidateAllCache()
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

func (m Model) Content() string {
	return strings.Join(m.lines, "\n")
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

func (m Model) InsertMode() bool {
	return m.insertMode
}

func (m *Model) getLinkAtCursor() *parser.Link {
	content := m.Content()
	pos := 0
	for i := 0; i < m.cursorRow && i < len(m.lines); i++ {
		pos += utf8.RuneCountInString(m.lines[i]) + 1
	}
	pos += m.cursorCol
	return parser.FindLinkAtPosition(content, pos)
}

func (m Model) CursorRow() int {
	return m.cursorRow
}
