package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Crawler  CrawlerConfig  `yaml:"crawler"`
	Database DatabaseConfig `yaml:"database"`
	Server   ServerConfig   `yaml:"server"`
	Sources  SourcesConfig  `yaml:"sources"`
}

// CrawlerConfig represents crawler configuration
type CrawlerConfig struct {
	Headless           bool          `yaml:"headless"`
	Timeout            time.Duration `yaml:"timeout"`
	WaitAfterLoad      time.Duration `yaml:"wait_after_load"`
	UserAgent          string        `yaml:"user_agent"`
	MaxConcurrent      int           `yaml:"max_concurrent"`
	RetryTimes         int           `yaml:"retry_times"`
	DisableImages      bool          `yaml:"disable_images"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// ServerConfig represents web server configuration
type ServerConfig struct {
	Port      int    `yaml:"port"`
	StaticDir string `yaml:"static_dir"`
}

// SourcesConfig represents sources configuration
type SourcesConfig struct {
	SUUMO  SourceConfig `yaml:"suumo"`
	HOMES  SourceConfig `yaml:"homes"`
	AtHome SourceConfig `yaml:"athome"`
}

// SourceConfig represents a single source configuration
type SourceConfig struct {
	Enabled  bool   `yaml:"enabled"`
	BaseURL  string `yaml:"base_url"`
	RentURL  string `yaml:"rent_url"`
	SaleURL  string `yaml:"sale_url"`
	MaxPages int    `yaml:"max_pages"`
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Default returns default configuration
func Default() *Config {
	return &Config{
		Crawler: CrawlerConfig{
			Headless:      true,
			Timeout:       60 * time.Second,
			WaitAfterLoad: 3 * time.Second,
			UserAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			MaxConcurrent: 2,
			RetryTimes:    3,
			DisableImages: false,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "",
			DBName:   "nagoya_properties",
			SSLMode:  "disable",
		},
		Server: ServerConfig{
			Port:      8080,
			StaticDir: "./static",
		},
		Sources: SourcesConfig{
			SUUMO: SourceConfig{
				Enabled:  true,
				BaseURL:  "https://suumo.jp",
				RentURL:  "https://suumo.jp/chintai/aichi/nc_nagoya/",
				SaleURL:  "https://suumo.jp/ms/chuko/aichi/nc_nagoya/",
				MaxPages: 5,
			},
			HOMES: SourceConfig{
				Enabled:  true,
				BaseURL:  "https://www.homes.co.jp",
				RentURL:  "https://www.homes.co.jp/chintai/aichi/nagoya/list/",
				SaleURL:  "https://www.homes.co.jp/chuko/aichi/nagoya/list/",
				MaxPages: 5,
			},
			AtHome: SourceConfig{
				Enabled:  true,
				BaseURL:  "https://www.at-home.co.jp",
				RentURL:  "https://www.at-home.co.jp/chintai/aichi/nagoya-city/list/",
				SaleURL:  "https://www.at-home.co.jp/chuko/aichi/nagoya-city/list/",
				MaxPages: 5,
			},
		},
	}
}
