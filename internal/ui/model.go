package ui

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/config"
	"github.com/takahashinaoki/obsidiantui/internal/components/backlinks"
	"github.com/takahashinaoki/obsidiantui/internal/components/cmdpalette"
	"github.com/takahashinaoki/obsidiantui/internal/components/filetree"
	"github.com/takahashinaoki/obsidiantui/internal/components/forwardlinks"
	"github.com/takahashinaoki/obsidiantui/internal/components/graph"
	"github.com/takahashinaoki/obsidiantui/internal/components/liveeditor"
	"github.com/takahashinaoki/obsidiantui/internal/components/outline"
	"github.com/takahashinaoki/obsidiantui/internal/components/preview"
	"github.com/takahashinaoki/obsidiantui/internal/components/search"
	"github.com/takahashinaoki/obsidiantui/internal/components/tagpane"
	"github.com/takahashinaoki/obsidiantui/internal/vault"
)

type Pane int

const (
	PaneFileTree Pane = iota
	PaneEditor
	PanePreview
)

type ViewMode int

const (
	ViewEdit ViewMode = iota
	ViewPreview
	ViewSplit
)

type Model struct {
	vault     *vault.Vault
	filetree  filetree.Model
	editor    liveeditor.Model
	preview   preview.Model
	search       search.Model
	backlinks    backlinks.Model
	forwardlinks forwardlinks.Model
	graph        graph.Model
	tagpane   tagpane.Model
	outline    outline.Model
	cmdpalette cmdpalette.Model
	help       help.Model
	keys      KeyMap

	activePane    Pane
	viewMode      ViewMode
	width         int
	height        int
	showHelp      bool
	statusMsg     string
	currentFile   string
	historyStack  []string
	cachedTreeW   int
	cachedContentW int
}

func NewModel(v *vault.Vault) Model {
	ft := filetree.New(v)
	ft.SetFocused(true)
	ed := liveeditor.New()
	pv := preview.New()
	pv.SetVault(v)
	sr := search.New(v)
	bl := backlinks.New(v)
	fl := forwardlinks.New(v)
	gr := graph.New(v)
	tp := tagpane.New(v)
	ol := outline.New()
	cp := cmdpalette.New()
	h := help.New()
	h.ShowAll = false

	return Model{
		vault:        v,
		filetree:     ft,
		editor:       ed,
		preview:      pv,
		search:       sr,
		backlinks:    bl,
		forwardlinks: fl,
		graph:        gr,
		tagpane:      tp,
		outline:      ol,
		cmdpalette:   cp,
		help:         h,
		keys:       DefaultKeyMap(),
		activePane: PaneFileTree,
		viewMode:   ViewEdit,
		statusMsg:  "Press ? for help | C-g:graph | Tab:switch pane",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		return m, nil

	case tea.KeyMsg:
		if m.cmdpalette.Active() {
			var cmd tea.Cmd
			m.cmdpalette, cmd = m.cmdpalette.Update(msg)
			return m, cmd
		}

		if m.search.Active() {
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			return m, cmd
		}

		if m.backlinks.Active() {
			var cmd tea.Cmd
			m.backlinks, cmd = m.backlinks.Update(msg)
			return m, cmd
		}

		if m.forwardlinks.Active() {
			var cmd tea.Cmd
			m.forwardlinks, cmd = m.forwardlinks.Update(msg)
			return m, cmd
		}

		if m.graph.Active() {
			var cmd tea.Cmd
			m.graph, cmd = m.graph.Update(msg)
			return m, cmd
		}

		if m.tagpane.Active() {
			var cmd tea.Cmd
			m.tagpane, cmd = m.tagpane.Update(msg)
			return m, cmd
		}

		if m.outline.Active() {
			var cmd tea.Cmd
			m.outline, cmd = m.outline.Update(msg)
			return m, cmd
		}

		if m.editor.InsertMode() {
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(msg)
			return m, cmd
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.editor.Modified() {
				m.statusMsg = "Unsaved changes! C-s:save C-c:force quit"
			} else {
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			m.help.ShowAll = m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.CmdPalette):
			m.cmdpalette.SetSize(m.width/2, m.height*3/4)
			return m, m.cmdpalette.Show()

		case key.Matches(msg, m.keys.FocusNext):
			m.cycleFocus(1)
			return m, nil

		case key.Matches(msg, m.keys.FocusPrev):
			m.cycleFocus(-1)
			return m, nil

		case key.Matches(msg, m.keys.FocusTree):
			m.setActivePane(PaneFileTree)
			return m, nil

		case key.Matches(msg, m.keys.FocusEdit):
			if m.viewMode == ViewPreview {
				m.setActivePane(PanePreview)
			} else {
				m.setActivePane(PaneEditor)
			}
			return m, nil

		case key.Matches(msg, m.keys.Search):
			m.search.SetSize(m.width/2, m.height/2)
			return m, m.search.Activate()

		case key.Matches(msg, m.keys.Backlinks):
			if m.currentFile != "" {
				m.backlinks.SetSize(m.width/2, m.height/2)
				m.backlinks.Show(m.currentFile)
			}
			return m, nil

		case key.Matches(msg, m.keys.ForwardLinks):
			if m.currentFile != "" {
				m.forwardlinks.SetSize(m.width/2, m.height/2)
				m.forwardlinks.Show(m.currentFile)
			}
			return m, nil

		case key.Matches(msg, m.keys.Graph):
			m.graph.SetSize(m.width*3/4, m.height*3/4)
			m.graph.Show(m.currentFile)
			return m, nil

		case key.Matches(msg, m.keys.Tags):
			m.tagpane.SetSize(m.width/2, m.height*3/4)
			m.tagpane.Show()
			return m, nil

		case key.Matches(msg, m.keys.Outline):
			if m.currentFile != "" {
				m.outline.SetSize(m.width/2, m.height*3/4)
				content := m.editor.Content()
				m.outline.SetContent(content, m.currentFile)
				m.outline.Show()
			}
			return m, nil

		case key.Matches(msg, m.keys.DailyNote):
			return m, m.openDailyNote()

		case key.Matches(msg, m.keys.ToggleView):
			m.cycleViewMode()
			m.updateLayout()
			return m, nil

		case key.Matches(msg, m.keys.ViewEdit):
			m.viewMode = ViewEdit
			m.updateLayout()
			return m, nil

		case key.Matches(msg, m.keys.ViewPrev):
			m.viewMode = ViewPreview
			m.updateLayout()
			return m, nil

		case key.Matches(msg, m.keys.ViewSplit):
			m.viewMode = ViewSplit
			m.updateLayout()
			return m, nil

		case key.Matches(msg, m.keys.Save):
			return m, m.saveCurrentFile()

		case key.Matches(msg, m.keys.Refresh):
			m.filetree.Refresh()
			m.statusMsg = "Vault refreshed"
			return m, nil

		case key.Matches(msg, m.keys.GoBack):
			return m, m.goBack()
		}

		return m, m.updateActivePane(msg)

	case tea.MouseMsg:
		if m.search.Active() || m.backlinks.Active() || m.forwardlinks.Active() || m.graph.Active() || m.tagpane.Active() || m.outline.Active() || m.cmdpalette.Active() {
			return m, nil
		}
		return m, m.handleMouseClick(msg)

	case filetree.FileSelectedMsg:
		return m, m.openFile(msg.Path)

	case search.FileSelectedMsg:
		return m, m.openFile(msg.Path)

	case search.SearchClosedMsg:
		return m, nil

	case backlinks.FileSelectedMsg:
		return m, m.openFile(msg.Path)

	case backlinks.BacklinksClosedMsg:
		return m, nil

	case forwardlinks.FileSelectedMsg:
		return m, m.openFile(msg.Path)

	case forwardlinks.ForwardLinksClosedMsg:
		return m, nil

	case graph.FileSelectedMsg:
		return m, m.openFile(msg.Path)

	case graph.GraphClosedMsg:
		return m, nil

	case tagpane.FileSelectedMsg:
		return m, m.openFile(msg.Path)

	case tagpane.TagPaneClosedMsg:
		return m, nil

	case outline.JumpToLineMsg:
		m.editor.JumpToLine(msg.Line)
		m.setActivePane(PaneEditor)
		return m, nil

	case outline.OutlineClosedMsg:
		return m, nil

	case cmdpalette.CommandMsg:
		return m, m.executeCommand(msg.ID)

	case cmdpalette.PaletteClosedMsg:
		return m, nil

	case liveeditor.SaveRequestMsg:
		return m, m.saveFile(msg.Path, msg.Content)

	case liveeditor.LinkFollowMsg:
		return m, m.followLink(msg.Target)

	case preview.LinkFollowMsg:
		return m, m.followLink(msg.Target)

	case fileSavedMsg:
		m.statusMsg = "File saved: " + msg.path
		m.editor.SetModified(false)
		return m, nil

	case fileOpenedMsg:
		m.currentFile = msg.path
		m.editor.SetContent(msg.content, msg.path)
		m.preview.SetContent(msg.content, msg.path)
		m.statusMsg = "Opened: " + msg.path
		if m.activePane == PaneFileTree {
			m.cycleFocus(1)
		}
		return m, nil

	case errMsg:
		m.statusMsg = "Error: " + msg.err.Error()
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	mainContent := m.renderMainContent()

	if m.search.Active() {
		overlay := m.search.View()
		mainContent = m.overlayCenter(mainContent, overlay)
	}

	if m.backlinks.Active() {
		overlay := m.backlinks.View()
		mainContent = m.overlayCenter(mainContent, overlay)
	}

	if m.forwardlinks.Active() {
		overlay := m.forwardlinks.View()
		mainContent = m.overlayCenter(mainContent, overlay)
	}

	if m.graph.Active() {
		overlay := m.graph.View()
		mainContent = m.overlayCenter(mainContent, overlay)
	}

	if m.tagpane.Active() {
		overlay := m.tagpane.View()
		mainContent = m.overlayCenter(mainContent, overlay)
	}

	if m.outline.Active() {
		overlay := m.outline.View()
		mainContent = m.overlayCenter(mainContent, overlay)
	}

	if m.cmdpalette.Active() {
		overlay := m.cmdpalette.View()
		mainContent = m.overlayCenter(mainContent, overlay)
	}

	statusBar := m.renderStatusBar()
	helpView := m.help.View(m.keys)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		statusBar,
		helpView,
	)
}

func (m *Model) updateLayout() {
	helpHeight := 1
	if m.showHelp {
		helpHeight = 4
	}
	statusHeight := 1
	contentHeight := m.height - statusHeight - helpHeight - 1
	if contentHeight < 5 {
		contentHeight = 5
	}

	treeWidth := m.width / 4
	if treeWidth < 20 {
		treeWidth = 20
	}
	if treeWidth > 40 {
		treeWidth = 40
	}

	contentWidth := m.width - treeWidth - 2
	if contentWidth < 20 {
		contentWidth = 20
	}

	m.cachedTreeW = treeWidth
	m.cachedContentW = contentWidth

	m.filetree.SetSize(treeWidth, contentHeight)

	switch m.viewMode {
	case ViewEdit:
		m.editor.SetSize(contentWidth, contentHeight)
	case ViewPreview:
		m.preview.SetSize(contentWidth, contentHeight)
	case ViewSplit:
		halfWidth := contentWidth / 2
		m.editor.SetSize(halfWidth, contentHeight)
		m.preview.SetSize(contentWidth-halfWidth, contentHeight)
	}

	m.search.SetSize(m.width/2, m.height/2)
	m.backlinks.SetSize(m.width/2, m.height/2)
}

func (m *Model) cycleFocus(direction int) {
	panes := []Pane{PaneFileTree}
	switch m.viewMode {
	case ViewEdit:
		panes = append(panes, PaneEditor)
	case ViewPreview:
		panes = append(panes, PanePreview)
	case ViewSplit:
		panes = append(panes, PaneEditor, PanePreview)
	}

	currentIdx := 0
	for i, p := range panes {
		if p == m.activePane {
			currentIdx = i
			break
		}
	}

	newIdx := (currentIdx + direction + len(panes)) % len(panes)
	m.activePane = panes[newIdx]

	m.filetree.SetFocused(m.activePane == PaneFileTree)
	m.editor.SetFocused(m.activePane == PaneEditor)
	m.preview.SetFocused(m.activePane == PanePreview)
}

func (m *Model) setActivePane(pane Pane) {
	m.activePane = pane
	m.filetree.SetFocused(pane == PaneFileTree)
	m.editor.SetFocused(pane == PaneEditor)
	m.preview.SetFocused(pane == PanePreview)
}

func (m *Model) goBack() tea.Cmd {
	if len(m.historyStack) > 1 {
		m.historyStack = m.historyStack[:len(m.historyStack)-1]
		prevFile := m.historyStack[len(m.historyStack)-1]
		return m.openFileWithoutHistory(prevFile)
	}
	m.statusMsg = "No history"
	return nil
}

func (m *Model) openFileWithoutHistory(path string) tea.Cmd {
	return func() tea.Msg {
		content, err := m.vault.ReadFile(path)
		if err != nil {
			return errMsg{err: err}
		}
		return fileOpenedMsg{path: path, content: content}
	}
}

func (m *Model) cycleViewMode() {
	switch m.viewMode {
	case ViewEdit:
		m.viewMode = ViewPreview
		if m.activePane == PaneEditor {
			m.activePane = PanePreview
		}
	case ViewPreview:
		m.viewMode = ViewSplit
	case ViewSplit:
		m.viewMode = ViewEdit
		if m.activePane == PanePreview {
			m.activePane = PaneEditor
		}
	}

	m.filetree.SetFocused(m.activePane == PaneFileTree)
	m.editor.SetFocused(m.activePane == PaneEditor)
	m.preview.SetFocused(m.activePane == PanePreview)
}

func (m *Model) updateActivePane(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch m.activePane {
	case PaneFileTree:
		m.filetree, cmd = m.filetree.Update(msg)
	case PaneEditor:
		m.editor, cmd = m.editor.Update(msg)
	case PanePreview:
		m.preview, cmd = m.preview.Update(msg)
	}

	return cmd
}

func (m *Model) handleMouseClick(msg tea.MouseMsg) tea.Cmd {
	// Handle focus change only on press
	if msg.Action == tea.MouseActionPress {
		if msg.X < m.cachedTreeW+1 {
			m.activePane = PaneFileTree
			m.filetree.SetFocused(true)
			m.editor.SetFocused(false)
			m.preview.SetFocused(false)
		} else {
			relX := msg.X - m.cachedTreeW - 2

			switch m.viewMode {
			case ViewEdit:
				m.activePane = PaneEditor
			case ViewPreview:
				m.activePane = PanePreview
			case ViewSplit:
				if relX < m.cachedContentW/2 {
					m.activePane = PaneEditor
				} else {
					m.activePane = PanePreview
				}
			}

			m.filetree.SetFocused(m.activePane == PaneFileTree)
			m.editor.SetFocused(m.activePane == PaneEditor)
			m.preview.SetFocused(m.activePane == PanePreview)
		}
	}

	// Create adjusted mouse message for each pane
	adjustedMsg := msg
	switch m.activePane {
	case PaneFileTree:
		// Adjust for border (1 pixel)
		adjustedMsg.X = msg.X - 1
		adjustedMsg.Y = msg.Y - 1
	case PaneEditor:
		// Adjust for tree width + borders
		adjustedMsg.X = msg.X - m.cachedTreeW - 2
		adjustedMsg.Y = msg.Y - 1
	case PanePreview:
		// Adjust for tree width + editor (in split) + borders
		if m.viewMode == ViewSplit {
			adjustedMsg.X = msg.X - m.cachedTreeW - m.cachedContentW/2 - 3
		} else {
			adjustedMsg.X = msg.X - m.cachedTreeW - 2
		}
		adjustedMsg.Y = msg.Y - 1
	}

	return m.updateActivePaneWithMsg(adjustedMsg)
}

func (m *Model) updateActivePaneWithMsg(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch m.activePane {
	case PaneFileTree:
		m.filetree, cmd = m.filetree.Update(msg)
	case PaneEditor:
		m.editor, cmd = m.editor.Update(msg)
	case PanePreview:
		m.preview, cmd = m.preview.Update(msg)
	}

	return cmd
}

func (m Model) renderMainContent() string {
	treeBorder := BorderInactiveStyle
	if m.activePane == PaneFileTree {
		treeBorder = BorderActiveStyle
	}

	treeView := treeBorder.Width(m.cachedTreeW).Render(m.filetree.View())

	var contentView string
	switch m.viewMode {
	case ViewEdit:
		contentView = m.editor.View()
	case ViewPreview:
		contentView = m.preview.View()
	case ViewSplit:
		contentView = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.editor.View(),
			m.preview.View(),
		)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, treeView, contentView)
}

func (m Model) renderStatusBar() string {
	vaultName := filepath.Base(m.vault.Path)

	left := StatusBarStyle.Render(" " + vaultName)

	// Mode and view
	modeStr := "NORMAL"
	if m.editor.InsertMode() {
		modeStr = "INSERT"
	}

	viewStr := ""
	switch m.viewMode {
	case ViewEdit:
		viewStr = "Edit"
	case ViewPreview:
		viewStr = "Preview"
	case ViewSplit:
		viewStr = "Split"
	}

	// Contextual keybindings hint
	var hints string
	if m.editor.InsertMode() {
		hints = "Esc:normal"
	} else {
		switch m.activePane {
		case PaneFileTree:
			hints = "Enter:open j/k:nav /:search"
		case PaneEditor:
			hints = "i:insert /:search C-e:view gd:link"
		case PanePreview:
			hints = "j/k:scroll Enter:link C-e:view"
		}
	}

	center := StatusBarStyle.Render(modeStr + " | " + viewStr + " | " + hints)

	// Right side: status message or modified indicator
	rightMsg := m.statusMsg
	if m.editor.Modified() && rightMsg == "" {
		rightMsg = "[modified]"
	}
	right := StatusBarStyle.Render(rightMsg + " ")

	leftWidth := lipgloss.Width(left)
	centerWidth := lipgloss.Width(center)
	rightWidth := lipgloss.Width(right)

	gap := m.width - leftWidth - centerWidth - rightWidth
	if gap < 0 {
		gap = 0
	}

	leftGap := gap / 2
	rightGap := gap - leftGap

	return left + strings.Repeat(" ", leftGap) + center + strings.Repeat(" ", rightGap) + right
}

func (m Model) overlayCenter(base, overlay string) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	overlayWidth := 0
	for _, line := range overlayLines {
		if w := lipgloss.Width(line); w > overlayWidth {
			overlayWidth = w
		}
	}

	startY := (len(baseLines) - len(overlayLines)) / 2
	startX := (m.width - overlayWidth) / 2

	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	for i, overlayLine := range overlayLines {
		lineIdx := startY + i
		if lineIdx >= len(baseLines) {
			break
		}

		baseLine := baseLines[lineIdx]
		baseRunes := []rune(baseLine)

		for len(baseRunes) < startX+len([]rune(overlayLine)) {
			baseRunes = append(baseRunes, ' ')
		}

		overlayRunes := []rune(overlayLine)
		for j, r := range overlayRunes {
			if startX+j < len(baseRunes) {
				baseRunes[startX+j] = r
			}
		}

		baseLines[lineIdx] = string(baseRunes)
	}

	return strings.Join(baseLines, "\n")
}

type fileSavedMsg struct {
	path string
}

type fileOpenedMsg struct {
	path    string
	content string
}

type errMsg struct {
	err error
}

func (m *Model) openFile(path string) tea.Cmd {
	return func() tea.Msg {
		content, err := m.vault.ReadFile(path)
		if err != nil {
			return errMsg{err: err}
		}

		m.historyStack = append(m.historyStack, path)

		config.AppConfig.LastOpenFile = path
		config.Save()

		return fileOpenedMsg{path: path, content: content}
	}
}

func (m *Model) saveCurrentFile() tea.Cmd {
	if m.currentFile == "" {
		return nil
	}
	return m.saveFile(m.currentFile, m.editor.Content())
}

func (m *Model) saveFile(path, content string) tea.Cmd {
	return func() tea.Msg {
		if err := m.vault.WriteFile(path, content); err != nil {
			return errMsg{err: err}
		}
		return fileSavedMsg{path: path}
	}
}

func (m *Model) followLink(target string) tea.Cmd {
	resolved := m.vault.FindFile(target + ".md")
	if resolved == "" {
		resolved = m.vault.FindFile(target)
	}

	if resolved == "" {
		m.statusMsg = "Link not found: " + target
		return nil
	}

	return m.openFile(resolved)
}

func (m *Model) openDailyNote() tea.Cmd {
	// Generate daily note filename (YYYY-MM-DD.md)
	today := time.Now().Format("2006-01-02")
	dailyPath := today + ".md"

	// Check if daily folder exists in vault
	dailyFolder := "Daily"
	if _, exists := m.vault.Files[dailyFolder]; exists {
		dailyPath = dailyFolder + "/" + today + ".md"
	}

	return func() tea.Msg {
		// Check if file exists
		content, err := m.vault.ReadFile(dailyPath)
		if err == nil {
			// File exists, open it
			m.historyStack = append(m.historyStack, dailyPath)
			config.AppConfig.LastOpenFile = dailyPath
			config.Save()
			return fileOpenedMsg{path: dailyPath, content: content}
		}

		// Create new daily note
		template := "# " + today + "\n\n"
		if err := m.vault.CreateFile(dailyPath); err != nil {
			return errMsg{err: err}
		}
		if err := m.vault.WriteFile(dailyPath, template); err != nil {
			return errMsg{err: err}
		}

		m.historyStack = append(m.historyStack, dailyPath)
		config.AppConfig.LastOpenFile = dailyPath
		config.Save()

		return fileOpenedMsg{path: dailyPath, content: template}
	}
}

func (m *Model) executeCommand(id string) tea.Cmd {
	switch id {
	case "search":
		m.search.SetSize(m.width/2, m.height/2)
		return m.search.Activate()
	case "graph":
		m.graph.SetSize(m.width*3/4, m.height*3/4)
		m.graph.Show(m.currentFile)
	case "tags":
		m.tagpane.SetSize(m.width/2, m.height*3/4)
		m.tagpane.Show()
	case "outline":
		if m.currentFile != "" {
			m.outline.SetSize(m.width/2, m.height*3/4)
			m.outline.SetContent(m.editor.Content(), m.currentFile)
			m.outline.Show()
		}
	case "backlinks":
		if m.currentFile != "" {
			m.backlinks.SetSize(m.width/2, m.height/2)
			m.backlinks.Show(m.currentFile)
		}
	case "forwardlinks":
		if m.currentFile != "" {
			m.forwardlinks.SetSize(m.width/2, m.height/2)
			m.forwardlinks.Show(m.currentFile)
		}
	case "daily":
		return m.openDailyNote()
	case "save":
		return m.saveCurrentFile()
	case "refresh":
		m.filetree.Refresh()
		m.statusMsg = "Vault refreshed"
	case "newfile":
		// TODO: implement new file dialog
		m.statusMsg = "New file: use Ctrl+N"
	case "help":
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
	case "edit":
		m.viewMode = ViewEdit
		m.updateLayout()
	case "preview":
		m.viewMode = ViewPreview
		m.updateLayout()
	case "split":
		m.viewMode = ViewSplit
		m.updateLayout()
	case "toggle":
		m.cycleViewMode()
		m.updateLayout()
	}
	return nil
}
