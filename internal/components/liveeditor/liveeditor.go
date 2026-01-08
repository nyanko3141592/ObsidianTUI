package liveeditor

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/internal/parser"
)

type Model struct {
	lines      []string
	cursorRow  int
	cursorCol  int
	offsetRow  int
	filePath   string
	modified   bool
	width      int
	height     int
	focused    bool
	renderer   *parser.MarkdownRenderer
	links      []parser.Link
	insertMode bool
}

type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Home       key.Binding
	End        key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Enter      key.Binding
	Backspace  key.Binding
	Delete     key.Binding
	Save       key.Binding
	InsertMode key.Binding
	NormalMode key.Binding
	FollowLink key.Binding
}

var DefaultKeyMap = KeyMap{
	Up:         key.NewBinding(key.WithKeys("up", "k")),
	Down:       key.NewBinding(key.WithKeys("down", "j")),
	Left:       key.NewBinding(key.WithKeys("left", "h")),
	Right:      key.NewBinding(key.WithKeys("right", "l")),
	Home:       key.NewBinding(key.WithKeys("home", "0")),
	End:        key.NewBinding(key.WithKeys("end", "$")),
	PageUp:     key.NewBinding(key.WithKeys("pgup", "ctrl+u")),
	PageDown:   key.NewBinding(key.WithKeys("pgdown", "ctrl+d")),
	Enter:      key.NewBinding(key.WithKeys("enter")),
	Backspace:  key.NewBinding(key.WithKeys("backspace")),
	Delete:     key.NewBinding(key.WithKeys("delete")),
	Save:       key.NewBinding(key.WithKeys("ctrl+s")),
	InsertMode: key.NewBinding(key.WithKeys("i", "a")),
	NormalMode: key.NewBinding(key.WithKeys("esc")),
	FollowLink: key.NewBinding(key.WithKeys("enter", "ctrl+]")),
}

var (
	headerStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	boldStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	italicStyle    = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("251"))
	linkStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Underline(true)
	codeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Background(lipgloss.Color("236"))
	tagStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("135"))
	mathStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	bulletStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	blockquoteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
	cursorStyle    = lipgloss.NewStyle().Reverse(true)
	lineNumStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(4)
)

type SaveRequestMsg struct {
	Path    string
	Content string
}

type LinkFollowMsg struct {
	Target string
}

func New() Model {
	return Model{
		lines:      []string{""},
		insertMode: false,
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
		if key.Matches(msg, DefaultKeyMap.Save) {
			return m, func() tea.Msg {
				return SaveRequestMsg{
					Path:    m.filePath,
					Content: m.Content(),
				}
			}
		}

		if key.Matches(msg, DefaultKeyMap.NormalMode) {
			m.insertMode = false
			return m, nil
		}

		if !m.insertMode {
			return m.handleNormalMode(msg)
		}
		return m.handleInsertMode(msg)

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			clickRow := msg.Y + m.offsetRow
			if clickRow < len(m.lines) {
				m.cursorRow = clickRow
				lineLen := utf8.RuneCountInString(m.lines[m.cursorRow])
				if msg.X-5 < lineLen {
					m.cursorCol = msg.X - 5
					if m.cursorCol < 0 {
						m.cursorCol = 0
					}
				} else {
					m.cursorCol = lineLen
				}
			}
		}
	}

	return m, nil
}

func (m Model) handleNormalMode(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, DefaultKeyMap.InsertMode):
		m.insertMode = true
		if msg.String() == "a" && m.cursorCol < utf8.RuneCountInString(m.currentLine()) {
			m.cursorCol++
		}

	case key.Matches(msg, DefaultKeyMap.Up):
		m.moveCursorUp()
	case key.Matches(msg, DefaultKeyMap.Down):
		m.moveCursorDown()
	case key.Matches(msg, DefaultKeyMap.Left):
		m.moveCursorLeft()
	case key.Matches(msg, DefaultKeyMap.Right):
		m.moveCursorRight()
	case key.Matches(msg, DefaultKeyMap.Home):
		m.cursorCol = 0
	case key.Matches(msg, DefaultKeyMap.End):
		m.cursorCol = utf8.RuneCountInString(m.currentLine())
	case key.Matches(msg, DefaultKeyMap.PageUp):
		for i := 0; i < m.height/2; i++ {
			m.moveCursorUp()
		}
	case key.Matches(msg, DefaultKeyMap.PageDown):
		for i := 0; i < m.height/2; i++ {
			m.moveCursorDown()
		}
	case key.Matches(msg, DefaultKeyMap.FollowLink):
		link := m.getLinkAtCursor()
		if link != nil {
			return m, func() tea.Msg {
				return LinkFollowMsg{Target: link.Target}
			}
		}

	case msg.String() == "w":
		m.moveWordForward()
	case msg.String() == "b":
		m.moveWordBackward()
	case msg.String() == "g":
		m.cursorRow = 0
		m.cursorCol = 0
		m.offsetRow = 0
	case msg.String() == "G":
		m.cursorRow = len(m.lines) - 1
		m.cursorCol = 0
	case msg.String() == "x":
		m.deleteChar()
		m.modified = true
	case msg.String() == "o":
		m.insertLineBelow()
		m.insertMode = true
		m.modified = true
	case msg.String() == "O":
		m.insertLineAbove()
		m.insertMode = true
		m.modified = true
	case msg.String() == "d":
		// dd to delete line - simplified
	}

	return m, nil
}

func (m Model) handleInsertMode(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		m.moveCursorUp()
	case key.Matches(msg, DefaultKeyMap.Down):
		m.moveCursorDown()
	case key.Matches(msg, DefaultKeyMap.Left):
		m.moveCursorLeft()
	case key.Matches(msg, DefaultKeyMap.Right):
		m.moveCursorRight()
	case key.Matches(msg, DefaultKeyMap.Enter):
		m.insertNewline()
		m.modified = true
	case key.Matches(msg, DefaultKeyMap.Backspace):
		m.backspace()
		m.modified = true
	case key.Matches(msg, DefaultKeyMap.Delete):
		m.deleteChar()
		m.modified = true
	default:
		if msg.Type == tea.KeyRunes {
			m.insertRunes(msg.Runes)
			m.modified = true
		}
	}

	return m, nil
}

func (m *Model) moveCursorUp() {
	if m.cursorRow > 0 {
		m.cursorRow--
		lineLen := utf8.RuneCountInString(m.lines[m.cursorRow])
		if m.cursorCol > lineLen {
			m.cursorCol = lineLen
		}
	}
	m.ensureCursorVisible()
}

func (m *Model) moveCursorDown() {
	if m.cursorRow < len(m.lines)-1 {
		m.cursorRow++
		lineLen := utf8.RuneCountInString(m.lines[m.cursorRow])
		if m.cursorCol > lineLen {
			m.cursorCol = lineLen
		}
	}
	m.ensureCursorVisible()
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
	line := m.currentLine()
	runes := []rune(line)
	col := m.cursorCol

	for col < len(runes) && !isWordChar(runes[col]) {
		col++
	}
	for col < len(runes) && isWordChar(runes[col]) {
		col++
	}

	if col >= len(runes) && m.cursorRow < len(m.lines)-1 {
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

	line := m.currentLine()
	runes := []rune(line)
	col := m.cursorCol

	if col > 0 {
		col--
	}
	for col > 0 && !isWordChar(runes[col]) {
		col--
	}
	for col > 0 && isWordChar(runes[col-1]) {
		col--
	}

	m.cursorCol = col
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func (m *Model) insertRunes(runes []rune) {
	line := m.currentLine()
	lineRunes := []rune(line)

	newLine := make([]rune, 0, len(lineRunes)+len(runes))
	newLine = append(newLine, lineRunes[:m.cursorCol]...)
	newLine = append(newLine, runes...)
	newLine = append(newLine, lineRunes[m.cursorCol:]...)

	m.lines[m.cursorRow] = string(newLine)
	m.cursorCol += len(runes)
}

func (m *Model) insertNewline() {
	line := m.currentLine()
	runes := []rune(line)

	before := string(runes[:m.cursorCol])
	after := string(runes[m.cursorCol:])

	m.lines[m.cursorRow] = before

	newLines := make([]string, 0, len(m.lines)+1)
	newLines = append(newLines, m.lines[:m.cursorRow+1]...)
	newLines = append(newLines, after)
	newLines = append(newLines, m.lines[m.cursorRow+1:]...)
	m.lines = newLines

	m.cursorRow++
	m.cursorCol = 0
	m.ensureCursorVisible()
}

func (m *Model) backspace() {
	if m.cursorCol > 0 {
		line := m.currentLine()
		runes := []rune(line)
		newLine := string(runes[:m.cursorCol-1]) + string(runes[m.cursorCol:])
		m.lines[m.cursorRow] = newLine
		m.cursorCol--
	} else if m.cursorRow > 0 {
		prevLine := m.lines[m.cursorRow-1]
		currLine := m.currentLine()
		m.cursorCol = utf8.RuneCountInString(prevLine)
		m.lines[m.cursorRow-1] = prevLine + currLine

		newLines := make([]string, 0, len(m.lines)-1)
		newLines = append(newLines, m.lines[:m.cursorRow]...)
		newLines = append(newLines, m.lines[m.cursorRow+1:]...)
		m.lines = newLines

		m.cursorRow--
	}
}

func (m *Model) deleteChar() {
	line := m.currentLine()
	runes := []rune(line)

	if m.cursorCol < len(runes) {
		newLine := string(runes[:m.cursorCol]) + string(runes[m.cursorCol+1:])
		m.lines[m.cursorRow] = newLine
	} else if m.cursorRow < len(m.lines)-1 {
		m.lines[m.cursorRow] = line + m.lines[m.cursorRow+1]
		newLines := make([]string, 0, len(m.lines)-1)
		newLines = append(newLines, m.lines[:m.cursorRow+1]...)
		newLines = append(newLines, m.lines[m.cursorRow+2:]...)
		m.lines = newLines
	}
}

func (m *Model) insertLineBelow() {
	newLines := make([]string, 0, len(m.lines)+1)
	newLines = append(newLines, m.lines[:m.cursorRow+1]...)
	newLines = append(newLines, "")
	newLines = append(newLines, m.lines[m.cursorRow+1:]...)
	m.lines = newLines
	m.cursorRow++
	m.cursorCol = 0
	m.ensureCursorVisible()
}

func (m *Model) insertLineAbove() {
	newLines := make([]string, 0, len(m.lines)+1)
	newLines = append(newLines, m.lines[:m.cursorRow]...)
	newLines = append(newLines, "")
	newLines = append(newLines, m.lines[m.cursorRow:]...)
	m.lines = newLines
	m.cursorCol = 0
	m.ensureCursorVisible()
}

func (m *Model) ensureCursorVisible() {
	if m.cursorRow < m.offsetRow {
		m.offsetRow = m.cursorRow
	}
	visibleHeight := m.height - 3
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	if m.cursorRow >= m.offsetRow+visibleHeight {
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

	headerStr := "NORMAL"
	headerBg := lipgloss.Color("57")
	if m.insertMode {
		headerStr = "INSERT"
		headerBg = lipgloss.Color("34")
	}

	modifiedStr := ""
	if m.modified {
		modifiedStr = " [+]"
	}

	fileName := m.filePath
	if fileName == "" {
		fileName = "No file"
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(headerBg).
		Width(m.width).
		Padding(0, 1).
		Render(headerStr + " | " + fileName + modifiedStr)

	b.WriteString(header + "\n")

	visibleHeight := m.height - 3
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	for i := 0; i < visibleHeight && m.offsetRow+i < len(m.lines); i++ {
		lineNum := m.offsetRow + i
		line := m.lines[lineNum]

		lineNumStr := lineNumStyle.Render(strings.Repeat(" ", 3-len(string(rune('0'+lineNum%10)))) + string(rune('0'+(lineNum+1)%10)) + " ")

		styledLine := m.renderLine(line, lineNum)

		b.WriteString(lineNumStr + styledLine + "\n")
	}

	for i := len(m.lines) - m.offsetRow; i < visibleHeight; i++ {
		b.WriteString(lineNumStyle.Render("~   ") + "\n")
	}

	return b.String()
}

func (m Model) renderLine(line string, lineNum int) string {
	if line == "" {
		if lineNum == m.cursorRow && m.focused {
			return cursorStyle.Render(" ")
		}
		return ""
	}

	runes := []rune(line)
	styled := m.styleLine(line)

	if lineNum == m.cursorRow && m.focused {
		styledRunes := []rune(styled)
		cursorPos := m.cursorCol
		if cursorPos > len(runes) {
			cursorPos = len(runes)
		}

		if cursorPos < len(runes) {
			beforeCursor := string(runes[:cursorPos])
			cursorChar := string(runes[cursorPos])
			afterCursor := string(runes[cursorPos+1:])

			beforeStyled := m.styleLine(beforeCursor)
			cursorStyled := cursorStyle.Render(cursorChar)
			afterStyled := m.styleLine(afterCursor)

			return beforeStyled + cursorStyled + afterStyled
		} else {
			return string(styledRunes) + cursorStyle.Render(" ")
		}
	}

	return styled
}

func (m Model) styleLine(line string) string {
	if strings.HasPrefix(line, "# ") {
		return headerStyle.Render(line)
	}
	if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
		return headerStyle.Render(line)
	}
	if strings.HasPrefix(line, "> ") {
		return blockquoteStyle.Render(line)
	}
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		return bulletStyle.Render(line[:2]) + m.styleInline(line[2:])
	}
	if len(line) > 0 && line[0] >= '0' && line[0] <= '9' && strings.Contains(line, ". ") {
		idx := strings.Index(line, ". ")
		return bulletStyle.Render(line[:idx+2]) + m.styleInline(line[idx+2:])
	}
	if strings.HasPrefix(line, "```") {
		return codeStyle.Render(line)
	}

	return m.styleInline(line)
}

func (m Model) styleInline(text string) string {
	result := text

	result = styleBetween(result, "**", "**", boldStyle)
	result = styleBetween(result, "__", "__", boldStyle)
	result = styleBetween(result, "*", "*", italicStyle)
	result = styleBetween(result, "_", "_", italicStyle)
	result = styleBetween(result, "`", "`", codeStyle)
	result = styleBetween(result, "[[", "]]", linkStyle)
	result = styleBetween(result, "$", "$", mathStyle)

	result = stylePattern(result, `#[a-zA-Z0-9_-]+`, tagStyle)

	return result
}

func styleBetween(text, start, end string, style lipgloss.Style) string {
	result := text
	for {
		startIdx := strings.Index(result, start)
		if startIdx == -1 {
			break
		}
		remaining := result[startIdx+len(start):]
		endIdx := strings.Index(remaining, end)
		if endIdx == -1 {
			break
		}

		before := result[:startIdx]
		content := remaining[:endIdx]
		after := remaining[endIdx+len(end):]

		result = before + style.Render(start+content+end) + after
	}
	return result
}

func stylePattern(text, pattern string, style lipgloss.Style) string {
	return text
}

func (m *Model) SetContent(content string, filePath string) {
	m.lines = strings.Split(content, "\n")
	if len(m.lines) == 0 {
		m.lines = []string{""}
	}
	m.filePath = filePath
	m.cursorRow = 0
	m.cursorCol = 0
	m.offsetRow = 0
	m.modified = false
	m.links = parser.ExtractAllLinks(content)
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.renderer == nil || m.width > 0 {
		m.renderer, _ = parser.NewMarkdownRenderer(width - 6)
	}
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
