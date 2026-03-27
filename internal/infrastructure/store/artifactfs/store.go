package artifactfs

import (
	"os"
	"path/filepath"
)

type Store struct {
	root string
}

func New(root string) Store {
	return Store{root: root}
}

func (s Store) Put(relativePath string, data []byte) (string, error) {
	fullPath := filepath.Join(s.root, filepath.Clean(relativePath))

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", err
	}

	return fullPath, nil
}
