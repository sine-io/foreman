package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfigUsesDefaultRuntimeRoot(t *testing.T) {
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Contains(t, cfg.RuntimeRoot, ".foreman")
}
