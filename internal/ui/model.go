package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/config"
	"github.com/takahashinaoki/obsidiantui/internal/components/backlinks"
	"github.com/takahashinaoki/obsidiantui/internal/components/editor"
	"github.com/takahashinaoki/obsidiantui/internal/components/filetree"
	"github.com/takahashinaoki/obsidiantui/internal/components/preview"
	"github.com/takahashinaoki/obsidiantui/internal/components/search"
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
	editor    editor.Model
	preview   preview.Model
	search    search.Model
	backlinks backlinks.Model
	help      help.Model
	keys      KeyMap

	activePane   Pane
	viewMode     ViewMode
	width        int
	height       int
	showHelp     bool
	statusMsg    string
	currentFile  string
	historyStack []string
}

func NewModel(v *vault.Vault) Model {
	ft := filetree.New(v)
	ft.SetFocused(true)
	ed := editor.New()
	pv := preview.New()
	sr := search.New(v)
	bl := backlinks.New(v)
	h := help.New()
	h.ShowAll = false

	return Model{
		vault:      v,
		filetree:   ft,
		editor:     ed,
		preview:    pv,
		search:     sr,
		backlinks:  bl,
		help:       h,
		keys:       DefaultKeyMap(),
		activePane: PaneFileTree,
		viewMode:   ViewSplit,
		statusMsg:  "Press ? for help",
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

		if m.editor.Mode() == editor.ModeInsert {
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(msg)
			return m, cmd
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.editor.Modified() {
				m.statusMsg = "Unsaved changes! Press ctrl+s to save or ctrl+c again to quit"
			} else {
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			m.help.ShowAll = m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.FocusNext):
			m.cycleFocus(1)
			return m, nil

		case key.Matches(msg, m.keys.FocusPrev):
			m.cycleFocus(-1)
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

		case key.Matches(msg, m.keys.ToggleView):
			m.cycleViewMode()
			m.updateLayout()
			return m, nil

		case key.Matches(msg, m.keys.Save):
			return m, m.saveCurrentFile()

		case key.Matches(msg, m.keys.Refresh):
			m.filetree.Refresh()
			m.statusMsg = "Vault refreshed"
			return m, nil
		}

		return m, m.updateActivePane(msg)

	case tea.MouseMsg:
		if m.search.Active() || m.backlinks.Active() {
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

	case editor.SaveRequestMsg:
		return m, m.saveFile(msg.Path, msg.Content)

	case editor.LinkFollowMsg:
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
	if msg.Action != tea.MouseActionPress {
		return nil
	}

	treeWidth := m.width / 4
	if treeWidth < 20 {
		treeWidth = 20
	}
	if treeWidth > 40 {
		treeWidth = 40
	}

	if msg.X < treeWidth {
		m.activePane = PaneFileTree
		m.filetree.SetFocused(true)
		m.editor.SetFocused(false)
		m.preview.SetFocused(false)

		var cmd tea.Cmd
		m.filetree, cmd = m.filetree.Update(msg)
		return cmd
	}

	contentWidth := m.width - treeWidth - 2
	relX := msg.X - treeWidth - 1

	switch m.viewMode {
	case ViewEdit:
		m.activePane = PaneEditor
	case ViewPreview:
		m.activePane = PanePreview
	case ViewSplit:
		if relX < contentWidth/2 {
			m.activePane = PaneEditor
		} else {
			m.activePane = PanePreview
		}
	}

	m.filetree.SetFocused(m.activePane == PaneFileTree)
	m.editor.SetFocused(m.activePane == PaneEditor)
	m.preview.SetFocused(m.activePane == PanePreview)

	return m.updateActivePane(msg)
}

func (m Model) renderMainContent() string {
	treeWidth := m.width / 4
	if treeWidth < 20 {
		treeWidth = 20
	}
	if treeWidth > 40 {
		treeWidth = 40
	}

	treeBorder := BorderInactiveStyle
	if m.activePane == PaneFileTree {
		treeBorder = BorderActiveStyle
	}

	treeView := treeBorder.Width(treeWidth).Render(m.filetree.View())

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

	modeStr := "NORMAL"
	if m.editor.Mode() == editor.ModeInsert {
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

	center := StatusBarStyle.Render(modeStr + " | " + viewStr)

	right := StatusBarStyle.Render(m.statusMsg + " ")

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
