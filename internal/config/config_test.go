package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaultsWithoutFile(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Addr != ":8080" {
		t.Errorf("addr = %q", cfg.Server.Addr)
	}
	if cfg.Cache.TTL != 24*time.Hour {
		t.Errorf("ttl = %v", cfg.Cache.TTL)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := "server:\n  addr: \":9090\"\ncache:\n  ttl: 2h\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Addr != ":9090" {
		t.Errorf("addr = %q, want :9090", cfg.Server.Addr)
	}
	if cfg.Cache.TTL != 2*time.Hour {
		t.Errorf("ttl = %v", cfg.Cache.TTL)
	}
	if cfg.Database.DSN == "" {
		t.Error("未指定的字段应保留默认值")
	}
}

func TestEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  addr: \":9090\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("IPSPOT_SERVER_ADDR", ":7777")
	t.Setenv("IPSPOT_SOURCES_ABUSEIPDB_API_KEY", "secret")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Addr != ":7777" {
		t.Errorf("addr = %q, want :7777", cfg.Server.Addr)
	}
	if cfg.Sources.AbuseIPDB.APIKey != "secret" {
		t.Errorf("api key = %q", cfg.Sources.AbuseIPDB.APIKey)
	}
}

func TestLoadMissingFileFails(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("期望报错")
	}
}

func TestValidateRejectsBadTTL(t *testing.T) {
	cfg := Default()
	cfg.Cache.TTL = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("期望报错")
	}
}
