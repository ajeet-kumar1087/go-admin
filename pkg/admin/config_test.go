package admin

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	conf := DefaultConfig()
	if conf.SiteTitle != "Go Admin" {
		t.Errorf("Expected default title 'Go Admin'")
	}
	if conf.DefaultPerPage != 10 {
		t.Errorf("Expected default per page 10")
	}
}

func TestLoadConfig(t *testing.T) {
	yamlContent := `
site_title: "Test Shop"
default_per_page: 25
`
	err := os.WriteFile("test_config.yml", []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove("test_config.yml")

	conf, err := LoadConfig("test_config.yml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if conf.SiteTitle != "Test Shop" {
		t.Errorf("Expected title 'Test Shop', got %s", conf.SiteTitle)
	}
	if conf.DefaultPerPage != 25 {
		t.Errorf("Expected per page 25, got %d", conf.DefaultPerPage)
	}
}
