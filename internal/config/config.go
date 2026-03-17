package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BindAddr      string `yaml:"-"`
	DataDir       string `yaml:"data_dir"`
	MasterKeyPath string `yaml:"master_key_path"`
	AuthStoreDir  string `yaml:"auth_store_dir"`
	CodexHome     string `yaml:"codex_home"`
	ProxyAPIKey   string `yaml:"codexsess_api_key"`
	ModelMappings map[string]string `yaml:"model_mappings"`
	LogLevel      string `yaml:"log_level"`
}

func Default() Config {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".codexsess")
	return Config{
		BindAddr:      resolveBindAddr(),
		DataDir:       base,
		MasterKeyPath: filepath.Join(base, "master.key"),
		AuthStoreDir:  filepath.Join(base, "auth-accounts"),
		CodexHome:     filepath.Join(home, ".codex"),
		ModelMappings: map[string]string{},
		LogLevel:      "info",
	}
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codexsess", "config.yaml")
}

func LoadOrInit() (Config, error) {
	cfg := Default()
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return cfg, err
	}
	p := configPath()
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		if cfg.ProxyAPIKey == "" {
			key, err := randomKey("sk-codexsess-")
			if err != nil {
				return cfg, err
			}
			cfg.ProxyAPIKey = key
		}
		if err := Save(cfg); err != nil {
			return cfg, err
		}
		cfg.BindAddr = resolveBindAddr()
		return cfg, nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	def := Default()
	if strings.TrimSpace(cfg.AuthStoreDir) == "" {
		cfg.AuthStoreDir = def.AuthStoreDir
	}
	if strings.TrimSpace(cfg.CodexHome) == "" {
		cfg.CodexHome = def.CodexHome
	}
	if cfg.ModelMappings == nil {
		cfg.ModelMappings = map[string]string{}
	}
	if cfg.ProxyAPIKey == "" {
		k, err := randomKey("sk-codexsess-")
		if err != nil {
			return cfg, err
		}
		cfg.ProxyAPIKey = k
		if err := Save(cfg); err != nil {
			return cfg, err
		}
	}
	cfg.BindAddr = resolveBindAddr()
	return cfg, nil
}

func Save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(configPath()), 0o700); err != nil {
		return err
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), b, 0o600)
}

func randomKey(prefix string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random key: %w", err)
	}
	return prefix + hex.EncodeToString(buf), nil
}

func resolveBindAddr() string {
	port := 3061
	if raw := strings.TrimSpace(os.Getenv("PORT")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 65535 {
			port = parsed
		}
	}
	return fmt.Sprintf("127.0.0.1:%d", port)
}
