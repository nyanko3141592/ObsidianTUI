package vault

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/takahashinaoki/obsidiantui/internal/parser"
)

type Vault struct {
	Path      string
	Files     map[string]*File
	Tags      map[string][]string
	Backlinks map[string][]string
}

type File struct {
	Path         string
	Name         string
	RelativePath string
	IsDir        bool
	Content      string
	Links        []parser.Link
	Tags         []string
	Modified     bool
}

func NewVault(path string) (*Vault, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrNotExist
	}

	v := &Vault{
		Path:      absPath,
		Files:     make(map[string]*File),
		Tags:      make(map[string][]string),
		Backlinks: make(map[string][]string),
	}

	if err := v.Scan(); err != nil {
		return nil, err
	}

	return v, nil
}

func (v *Vault) Scan() error {
	v.Files = make(map[string]*File)
	v.Tags = make(map[string][]string)
	v.Backlinks = make(map[string][]string)

	err := filepath.Walk(v.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name := info.Name()
		if strings.HasPrefix(name, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, _ := filepath.Rel(v.Path, path)

		if info.IsDir() {
			v.Files[relPath] = &File{
				Path:         path,
				Name:         name,
				RelativePath: relPath,
				IsDir:        true,
			}
			return nil
		}

		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			return nil
		}

		file := &File{
			Path:         path,
			Name:         name,
			RelativePath: relPath,
			IsDir:        false,
		}
		v.Files[relPath] = file

		return nil
	})

	if err != nil {
		return err
	}

	v.buildIndex()
	return nil
}

func (v *Vault) buildIndex() {
	contents := make(map[string]string)

	for relPath, file := range v.Files {
		if file.IsDir {
			continue
		}

		content, err := v.ReadFile(relPath)
		if err != nil {
			continue
		}
		contents[relPath] = content

		file.Links = parser.ExtractAllLinks(content)
		file.Tags = parser.ExtractUniqueTags(content)

		for _, tag := range file.Tags {
			v.Tags[tag] = append(v.Tags[tag], relPath)
		}
	}

	for relPath, file := range v.Files {
		if file.IsDir {
			continue
		}

		for _, link := range file.Links {
			if link.IsWikiLink {
				targetName := parser.ResolveWikiLink(link.Target, filepath.Dir(relPath), v.Path)
				targetPath := v.FindFile(targetName)
				if targetPath != "" {
					v.Backlinks[targetPath] = append(v.Backlinks[targetPath], relPath)
				}
			}
		}
	}
}

func (v *Vault) ReadFile(relPath string) (string, error) {
	file, ok := v.Files[relPath]
	if !ok {
		return "", os.ErrNotExist
	}

	if file.Content != "" {
		return file.Content, nil
	}

	content, err := os.ReadFile(file.Path)
	if err != nil {
		return "", err
	}

	file.Content = string(content)
	return file.Content, nil
}

func (v *Vault) WriteFile(relPath string, content string) error {
	file, ok := v.Files[relPath]
	if !ok {
		return os.ErrNotExist
	}

	if err := os.WriteFile(file.Path, []byte(content), 0644); err != nil {
		return err
	}

	file.Content = content
	file.Modified = false
	file.Links = parser.ExtractAllLinks(content)
	file.Tags = parser.ExtractUniqueTags(content)

	return nil
}

func (v *Vault) FindFile(name string) string {
	name = strings.ToLower(name)

	for relPath := range v.Files {
		baseName := strings.ToLower(filepath.Base(relPath))
		if baseName == name {
			return relPath
		}
	}

	for relPath := range v.Files {
		if strings.ToLower(relPath) == name {
			return relPath
		}
	}

	return ""
}

func (v *Vault) GetBacklinks(relPath string) []string {
	backlinks := v.Backlinks[relPath]
	sort.Strings(backlinks)
	return backlinks
}

func (v *Vault) GetFilesWithTag(tag string) []string {
	files := v.Tags[tag]
	sort.Strings(files)
	return files
}

func (v *Vault) Search(query string) []string {
	query = strings.ToLower(query)
	var results []string

	for relPath, file := range v.Files {
		if file.IsDir {
			continue
		}

		if strings.Contains(strings.ToLower(file.Name), query) {
			results = append(results, relPath)
			continue
		}

		content, _ := v.ReadFile(relPath)
		if strings.Contains(strings.ToLower(content), query) {
			results = append(results, relPath)
		}
	}

	sort.Strings(results)
	return results
}

func (v *Vault) CreateFile(relPath string) error {
	fullPath := filepath.Join(v.Path, relPath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	f.Close()

	v.Files[relPath] = &File{
		Path:         fullPath,
		Name:         filepath.Base(relPath),
		RelativePath: relPath,
		IsDir:        false,
	}

	return nil
}

func (v *Vault) DeleteFile(relPath string) error {
	file, ok := v.Files[relPath]
	if !ok {
		return os.ErrNotExist
	}

	if err := os.Remove(file.Path); err != nil {
		return err
	}

	delete(v.Files, relPath)
	return nil
}
