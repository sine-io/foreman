package bootstrap

import (
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	RuntimeRoot  string
	DBPath       string
	ArtifactRoot string
	HTTPAddr     string
}

func LoadConfig() (Config, error) {
	defaultRoot, err := DefaultRuntimeRoot()
	if err != nil {
		return Config{}, err
	}

	v := viper.New()
	v.SetEnvPrefix("FOREMAN")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("runtime_root", defaultRoot)
	v.SetDefault("db_path", filepath.Join(defaultRoot, "foreman.db"))
	v.SetDefault("artifact_root", filepath.Join(defaultRoot, "artifacts"))
	v.SetDefault("http_addr", "127.0.0.1:8080")

	return Config{
		RuntimeRoot:  v.GetString("runtime_root"),
		DBPath:       v.GetString("db_path"),
		ArtifactRoot: v.GetString("artifact_root"),
		HTTPAddr:     v.GetString("http_addr"),
	}, nil
}
