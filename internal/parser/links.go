package parser

import (
	"regexp"
	"strings"
)

var (
	wikiLinkPattern     = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)
	markdownLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

type Link struct {
	Target      string
	DisplayText string
	StartPos    int
	EndPos      int
	IsWikiLink  bool
}

func ExtractWikiLinks(content string) []Link {
	var links []Link
	matches := wikiLinkPattern.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			target := content[match[2]:match[3]]
			displayText := target

			if match[4] != -1 && match[5] != -1 {
				displayText = content[match[4]:match[5]]
			}

			links = append(links, Link{
				Target:      target,
				DisplayText: displayText,
				StartPos:    match[0],
				EndPos:      match[1],
				IsWikiLink:  true,
			})
		}
	}

	return links
}

func ExtractMarkdownLinks(content string) []Link {
	var links []Link
	matches := markdownLinkPattern.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			displayText := content[match[2]:match[3]]
			target := content[match[4]:match[5]]

			links = append(links, Link{
				Target:      target,
				DisplayText: displayText,
				StartPos:    match[0],
				EndPos:      match[1],
				IsWikiLink:  false,
			})
		}
	}

	return links
}

func ExtractAllLinks(content string) []Link {
	wikiLinks := ExtractWikiLinks(content)
	mdLinks := ExtractMarkdownLinks(content)
	return append(wikiLinks, mdLinks...)
}

func ResolveWikiLink(linkTarget string, currentDir string, vaultPath string) string {
	target := strings.TrimSpace(linkTarget)

	if strings.Contains(target, "#") {
		parts := strings.SplitN(target, "#", 2)
		target = parts[0]
	}

	if !strings.HasSuffix(target, ".md") {
		target = target + ".md"
	}

	return target
}

func FindLinkAtPosition(content string, pos int) *Link {
	links := ExtractAllLinks(content)
	for _, link := range links {
		if pos >= link.StartPos && pos < link.EndPos {
			return &link
		}
	}
	return nil
}
