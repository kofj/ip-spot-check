package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Server struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type Database struct {
	Driver       string `yaml:"driver"`
	DSN          string `yaml:"dsn"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

type Log struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type SourceMaxMind struct {
	DBPath    string `yaml:"db_path"`
	ASNDBPath string `yaml:"asn_db_path"`
}

type SourceAbuseIPDB struct {
	APIKey     string `yaml:"api_key"`
	DailyLimit int    `yaml:"daily_limit"`
}

type SourceRDAP struct {
	BaseURL string        `yaml:"base_url"`
	Timeout time.Duration `yaml:"timeout"`
}

type Sources struct {
	MaxMind   SourceMaxMind   `yaml:"maxmind"`
	AbuseIPDB SourceAbuseIPDB `yaml:"abuseipdb"`
	RDAP      SourceRDAP      `yaml:"rdap"`
}

type Cache struct {
	Backend string        `yaml:"backend"`
	TTL     time.Duration `yaml:"ttl"`
}

type Config struct {
	Server   Server   `yaml:"server"`
	Database Database `yaml:"database"`
	Log      Log      `yaml:"log"`
	Sources  Sources  `yaml:"sources"`
	Cache    Cache    `yaml:"cache"`
}

func Default() Config {
	return Config{
		Server: Server{
			Addr:         ":8080",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		Database: Database{
			Driver:       "sqlite",
			DSN:          "file:./data/ipspot.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)",
			MaxOpenConns: 4,
		},
		Log: Log{Level: "info", Format: "json"},
		Sources: Sources{
			MaxMind:   SourceMaxMind{DBPath: "./data/GeoLite2-City.mmdb", ASNDBPath: "./data/GeoLite2-ASN.mmdb"},
			AbuseIPDB: SourceAbuseIPDB{DailyLimit: 1000},
			RDAP:      SourceRDAP{BaseURL: "https://rdap.org", Timeout: 5 * time.Second},
		},
		Cache: Cache{Backend: "memory", TTL: 24 * time.Hour},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()

	if path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("read config %s: %w", path, err)
		}
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			return cfg, fmt.Errorf("parse config %s: %w", path, err)
		}
	}

	if err := applyEnv(&cfg); err != nil {
		return cfg, err
	}
	return cfg, cfg.Validate()
}

func applyEnv(cfg *Config) error {
	str := func(key string, dst *string) {
		if v, ok := os.LookupEnv(key); ok {
			*dst = v
		}
	}
	str("IPSPOT_SERVER_ADDR", &cfg.Server.Addr)
	str("IPSPOT_DATABASE_DSN", &cfg.Database.DSN)
	str("IPSPOT_LOG_LEVEL", &cfg.Log.Level)
	str("IPSPOT_LOG_FORMAT", &cfg.Log.Format)
	str("IPSPOT_SOURCES_ABUSEIPDB_API_KEY", &cfg.Sources.AbuseIPDB.APIKey)
	str("IPSPOT_SOURCES_MAXMIND_DB_PATH", &cfg.Sources.MaxMind.DBPath)
	str("IPSPOT_CACHE_BACKEND", &cfg.Cache.Backend)

	if v, ok := os.LookupEnv("IPSPOT_DATABASE_MAX_OPEN_CONNS"); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("IPSPOT_DATABASE_MAX_OPEN_CONNS: %w", err)
		}
		cfg.Database.MaxOpenConns = n
	}
	return nil
}

func (c Config) Validate() error {
	var problems []string
	if c.Server.Addr == "" {
		problems = append(problems, "server.addr 不能为空")
	}
	if c.Database.Driver == "" {
		problems = append(problems, "database.driver 不能为空")
	}
	if c.Database.DSN == "" {
		problems = append(problems, "database.dsn 不能为空")
	}
	if c.Database.MaxOpenConns < 1 {
		problems = append(problems, "database.max_open_conns 必须 >= 1")
	}
	if c.Cache.TTL <= 0 {
		problems = append(problems, "cache.ttl 必须 > 0")
	}
	if len(problems) > 0 {
		return fmt.Errorf("配置无效: %s", strings.Join(problems, "; "))
	}
	return nil
}
