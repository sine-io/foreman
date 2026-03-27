package bootstrap

import (
	"os"
	"path/filepath"
)

func DefaultRuntimeRoot() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".foreman"), nil
}

func PrepareRuntime(cfg Config) error {
	for _, path := range []string{cfg.RuntimeRoot, cfg.ArtifactRoot} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}

	return nil
}
