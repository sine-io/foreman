package artifactfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	root string
}

func New(root string) Store {
	return Store{root: root}
}

func (s Store) Put(relativePath string, data []byte) (string, error) {
	fullPath, err := s.resolvePath(relativePath)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", err
	}

	return fullPath, nil
}

func (s Store) ReadPreview(path string, maxBytes int) (string, bool, error) {
	if maxBytes < 0 {
		return "", false, fmt.Errorf("maxBytes must be non-negative")
	}

	fullPath, err := s.resolvePath(path)
	if err != nil {
		return "", false, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return "", false, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, int64(maxBytes+1)))
	if err != nil {
		return "", false, err
	}

	truncated := len(data) > maxBytes
	if truncated {
		data = data[:maxBytes]
	}

	return string(data), truncated, nil
}

func (s Store) ResolveDisplayPath(path string) (string, error) {
	fullPath, err := s.resolvePath(path)
	if err != nil {
		return "", err
	}

	rootPath, err := s.rootPath()
	if err != nil {
		return "", err
	}

	relativePath, err := filepath.Rel(rootPath, fullPath)
	if err != nil {
		return "", err
	}

	if relativePath == "." {
		return "", nil
	}

	return filepath.ToSlash(relativePath), nil
}

func (s Store) resolvePath(path string) (string, error) {
	rootPath, err := s.rootPath()
	if err != nil {
		return "", err
	}

	candidate := filepath.Clean(path)
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(rootPath, candidate)
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return "", err
	}

	relativePath, err := filepath.Rel(rootPath, candidate)
	if err != nil {
		return "", err
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("artifact path %q escapes root %q", path, rootPath)
	}

	return candidate, nil
}

func (s Store) rootPath() (string, error) {
	return filepath.Abs(filepath.Clean(s.root))
}
