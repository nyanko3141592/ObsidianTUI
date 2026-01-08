package vault

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/takahashinaoki/obsidiantui/internal/parser"
)

type Vault struct {
	Path         string
	Files        map[string]*File
	Tags         map[string][]string
	Backlinks    map[string][]string
	indexed      bool
	indexing     bool
	mu           sync.RWMutex
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

	if err := v.ScanFiles(); err != nil {
		return nil, err
	}

	go v.BuildIndexAsync()

	return v, nil
}

func (v *Vault) ScanFiles() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.Files = make(map[string]*File)

	return filepath.Walk(v.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
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

		v.Files[relPath] = &File{
			Path:         path,
			Name:         name,
			RelativePath: relPath,
			IsDir:        false,
		}

		return nil
	})
}

func (v *Vault) BuildIndexAsync() {
	v.mu.Lock()
	if v.indexing || v.indexed {
		v.mu.Unlock()
		return
	}
	v.indexing = true
	v.mu.Unlock()

	tags := make(map[string][]string)
	backlinks := make(map[string][]string)

	v.mu.RLock()
	files := make(map[string]*File)
	for k, f := range v.Files {
		files[k] = f
	}
	v.mu.RUnlock()

	for relPath, file := range files {
		if file.IsDir {
			continue
		}

		content, err := os.ReadFile(file.Path)
		if err != nil {
			continue
		}

		contentStr := string(content)
		links := parser.ExtractAllLinks(contentStr)
		fileTags := parser.ExtractUniqueTags(contentStr)

		v.mu.Lock()
		if f, ok := v.Files[relPath]; ok {
			f.Links = links
			f.Tags = fileTags
		}
		v.mu.Unlock()

		for _, tag := range fileTags {
			tags[tag] = append(tags[tag], relPath)
		}

		for _, link := range links {
			if link.IsWikiLink {
				targetName := parser.ResolveWikiLink(link.Target, filepath.Dir(relPath), v.Path)
				targetPath := v.findFileInternal(files, targetName)
				if targetPath != "" {
					backlinks[targetPath] = append(backlinks[targetPath], relPath)
				}
			}
		}
	}

	v.mu.Lock()
	v.Tags = tags
	v.Backlinks = backlinks
	v.indexed = true
	v.indexing = false
	v.mu.Unlock()
}

func (v *Vault) findFileInternal(files map[string]*File, name string) string {
	name = strings.ToLower(name)

	for relPath := range files {
		baseName := strings.ToLower(filepath.Base(relPath))
		if baseName == name {
			return relPath
		}
	}

	for relPath := range files {
		if strings.ToLower(relPath) == name {
			return relPath
		}
	}

	return ""
}

func (v *Vault) Scan() error {
	if err := v.ScanFiles(); err != nil {
		return err
	}
	v.mu.Lock()
	v.indexed = false
	v.mu.Unlock()
	go v.BuildIndexAsync()
	return nil
}

func (v *Vault) ReadFile(relPath string) (string, error) {
	v.mu.RLock()
	file, ok := v.Files[relPath]
	v.mu.RUnlock()

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

	v.mu.Lock()
	file.Content = string(content)
	v.mu.Unlock()

	return file.Content, nil
}

func (v *Vault) WriteFile(relPath string, content string) error {
	v.mu.RLock()
	file, ok := v.Files[relPath]
	v.mu.RUnlock()

	if !ok {
		return os.ErrNotExist
	}

	if err := os.WriteFile(file.Path, []byte(content), 0644); err != nil {
		return err
	}

	v.mu.Lock()
	file.Content = content
	file.Modified = false
	file.Links = parser.ExtractAllLinks(content)
	file.Tags = parser.ExtractUniqueTags(content)
	v.mu.Unlock()

	return nil
}

func (v *Vault) FindFile(name string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()

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
	v.mu.RLock()
	defer v.mu.RUnlock()

	backlinks := v.Backlinks[relPath]
	result := make([]string, len(backlinks))
	copy(result, backlinks)
	sort.Strings(result)
	return result
}

func (v *Vault) GetFilesWithTag(tag string) []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	files := v.Tags[tag]
	result := make([]string, len(files))
	copy(result, files)
	sort.Strings(result)
	return result
}

func (v *Vault) Search(query string) []string {
	v.mu.RLock()
	files := make(map[string]*File)
	for k, f := range v.Files {
		files[k] = f
	}
	v.mu.RUnlock()

	query = strings.ToLower(query)
	var results []string

	for relPath, file := range files {
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

	v.mu.Lock()
	v.Files[relPath] = &File{
		Path:         fullPath,
		Name:         filepath.Base(relPath),
		RelativePath: relPath,
		IsDir:        false,
	}
	v.mu.Unlock()

	return nil
}

func (v *Vault) DeleteFile(relPath string) error {
	v.mu.RLock()
	file, ok := v.Files[relPath]
	v.mu.RUnlock()

	if !ok {
		return os.ErrNotExist
	}

	if err := os.Remove(file.Path); err != nil {
		return err
	}

	v.mu.Lock()
	delete(v.Files, relPath)
	v.mu.Unlock()

	return nil
}

func (v *Vault) IsIndexed() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.indexed
}
