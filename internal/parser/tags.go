package parser

import (
	"regexp"
	"sort"
	"strings"
)

var tagPattern = regexp.MustCompile(`(?:^|\s)#([a-zA-Z0-9_\-/]+)`)

type Tag struct {
	Name     string
	StartPos int
	EndPos   int
}

func ExtractTags(content string) []Tag {
	var tags []Tag
	matches := tagPattern.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			tagName := content[match[2]:match[3]]
			hashPos := strings.LastIndex(content[match[0]:match[1]], "#")
			startPos := match[0] + hashPos

			tags = append(tags, Tag{
				Name:     tagName,
				StartPos: startPos,
				EndPos:   match[1],
			})
		}
	}

	return tags
}

func ExtractUniqueTags(content string) []string {
	tags := ExtractTags(content)
	seen := make(map[string]bool)
	var unique []string

	for _, tag := range tags {
		if !seen[tag.Name] {
			seen[tag.Name] = true
			unique = append(unique, tag.Name)
		}
	}

	sort.Strings(unique)
	return unique
}

func FindTagAtPosition(content string, pos int) *Tag {
	tags := ExtractTags(content)
	for _, tag := range tags {
		if pos >= tag.StartPos && pos < tag.EndPos {
			return &tag
		}
	}
	return nil
}

func CollectAllTags(files map[string]string) map[string][]string {
	tagToFiles := make(map[string][]string)

	for filePath, content := range files {
		tags := ExtractUniqueTags(content)
		for _, tag := range tags {
			tagToFiles[tag] = append(tagToFiles[tag], filePath)
		}
	}

	for tag := range tagToFiles {
		sort.Strings(tagToFiles[tag])
	}

	return tagToFiles
}
