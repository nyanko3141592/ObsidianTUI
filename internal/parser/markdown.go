package parser

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
	width    int
}

func NewMarkdownRenderer(width int) (*MarkdownRenderer, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}

	return &MarkdownRenderer{renderer: renderer, width: width}, nil
}

func (m *MarkdownRenderer) Render(content string) (string, error) {
	content = RenderTeX(content)
	return m.renderer.Render(content)
}

func (m *MarkdownRenderer) RenderWithTeX(content string) (string, error) {
	content = RenderTeX(content)
	return m.renderer.Render(content)
}

func ExtractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

func ExtractFrontmatter(content string) (map[string]string, string) {
	frontmatter := make(map[string]string)

	if !strings.HasPrefix(content, "---") {
		return frontmatter, content
	}

	lines := strings.Split(content, "\n")
	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIndex = i
			break
		}
	}

	if endIndex == -1 {
		return frontmatter, content
	}

	for i := 1; i < endIndex; i++ {
		line := lines[i]
		if colonIndex := strings.Index(line, ":"); colonIndex != -1 {
			key := strings.TrimSpace(line[:colonIndex])
			value := strings.TrimSpace(line[colonIndex+1:])
			frontmatter[key] = value
		}
	}

	bodyLines := lines[endIndex+1:]
	body := strings.Join(bodyLines, "\n")
	body = strings.TrimPrefix(body, "\n")

	return frontmatter, body
}
