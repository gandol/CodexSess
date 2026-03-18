package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.BindAddr == "" {
		t.Fatal("BindAddr empty")
	}
	if cfg.DataDir == "" {
		t.Fatal("DataDir empty")
	}
	if cfg.MasterKeyPath == "" {
		t.Fatal("MasterKeyPath empty")
	}
}

func TestResolveBindAddr_DefaultIsAllInterfaces(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("CODEXSESS_BIND_ADDR", "")
	if got := resolveBindAddr(); got != "0.0.0.0:3061" {
		t.Fatalf("unexpected default bind addr: %q", got)
	}
}

func TestResolveBindAddr_UsesPortEnv(t *testing.T) {
	t.Setenv("PORT", "4021")
	t.Setenv("CODEXSESS_BIND_ADDR", "")
	if got := resolveBindAddr(); got != "0.0.0.0:4021" {
		t.Fatalf("unexpected bind addr with PORT: %q", got)
	}
}

func TestResolveBindAddr_UsesExplicitBindAddr(t *testing.T) {
	t.Setenv("PORT", "3061")
	t.Setenv("CODEXSESS_BIND_ADDR", "127.0.0.1:9090")
	if got := resolveBindAddr(); got != "127.0.0.1:9090" {
		t.Fatalf("unexpected explicit bind addr: %q", got)
	}
}

func TestResolveCodexBin_Default(t *testing.T) {
	t.Setenv("CODEXSESS_CODEX_BIN", "")
	if got := resolveCodexBin(""); got != "codex" {
		t.Fatalf("unexpected default codex bin: %q", got)
	}
}

func TestResolveCodexBin_UsesConfigValue(t *testing.T) {
	t.Setenv("CODEXSESS_CODEX_BIN", "")
	if got := resolveCodexBin("/usr/local/bin/codex"); got != "/usr/local/bin/codex" {
		t.Fatalf("unexpected codex bin from config: %q", got)
	}
}

func TestResolveCodexBin_EnvOverride(t *testing.T) {
	t.Setenv("CODEXSESS_CODEX_BIN", "/opt/codex/bin/codex")
	if got := resolveCodexBin("codex"); got != "/opt/codex/bin/codex" {
		t.Fatalf("unexpected codex bin from env override: %q", got)
	}
}
