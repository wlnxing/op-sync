package openlistsync

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

type pathFilter struct {
	patterns []string
}

func newPathFilter(patterns []string) (*pathFilter, error) {
	normalized := normalizePatterns(patterns)
	for _, p := range normalized {
		if !isValidPattern(p) {
			return nil, fmt.Errorf("invalid blacklist pattern: %s", p)
		}
	}
	return &pathFilter{patterns: normalized}, nil
}

func (f *pathFilter) count() int {
	if f == nil {
		return 0
	}
	return len(f.patterns)
}

func (f *pathFilter) match(relPath string) bool {
	if f == nil || len(f.patterns) == 0 {
		return false
	}

	relPath = normalizeRelativePath(relPath)
	baseName := path.Base(relPath)
	for _, pattern := range f.patterns {
		if strings.Contains(pattern, "/") {
			if ok, _ := path.Match(pattern, relPath); ok {
				return true
			}
			continue
		}
		if ok, _ := path.Match(pattern, baseName); ok {
			return true
		}
	}
	return false
}

func normalizePatterns(patterns []string) []string {
	normalized := make([]string, 0, len(patterns))
	for _, p := range patterns {
		p = normalizePattern(p)
		if p == "" {
			continue
		}
		if !slices.Contains(normalized, p) {
			normalized = append(normalized, p)
		}
	}
	return normalized
}

func normalizePattern(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	return p
}

func normalizeRelativePath(relPath string) string {
	relPath = strings.TrimSpace(relPath)
	relPath = strings.ReplaceAll(relPath, "\\", "/")
	relPath = strings.TrimPrefix(relPath, "./")
	relPath = strings.TrimPrefix(relPath, "/")
	relPath = path.Clean(relPath)
	if relPath == "." {
		return ""
	}
	return relPath
}

func isValidPattern(p string) bool {
	if p == "" {
		return true
	}
	_, err := path.Match(p, "x")
	return err == nil
}
