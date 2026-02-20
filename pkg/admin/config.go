package admin

import (
	"gopkg.in/yaml.v3"
	"os"
)

// Config holds the configuration for the admin panel.
type Config struct {
	SiteTitle       string `yaml:"site_title"`
	DefaultPerPage  int    `yaml:"default_per_page"`
	ThemeColor      string `yaml:"theme_color"`
	SessionTTL      int    `yaml:"session_ttl_hours"`
	SearchThreshold int64  `yaml:"search_threshold"`
	UploadDir       string `yaml:"upload_dir"`
}

// DefaultConfig returns a sane default configuration.
func DefaultConfig() *Config {
	return &Config{
		SiteTitle:       "Go Admin",
		DefaultPerPage:  10,
		ThemeColor:      "#2563eb",
		SessionTTL:      24,
		SearchThreshold: 50,
		UploadDir:       "uploads",
	}
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()
	file, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err := d.Decode(&config); err != nil {
		return nil, err
	}
	return config, nil
}
