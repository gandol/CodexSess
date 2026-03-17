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
