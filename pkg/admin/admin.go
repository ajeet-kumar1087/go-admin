package admin

import (
	"fmt"
	"gorm.io/gorm"
	"net/http"
)

// Registry keeps track of all registered admin resources and pages.
type Registry struct {
	DB        *gorm.DB
	Resources map[string]*Resource
	Pages     map[string]*Page
	Charts    []Chart
	Config    *Config
}

// Page represents a custom non-model page.
type Page struct {
	Name    string
	Group   string
	Handler http.HandlerFunc
}

// Chart represents a visual widget on the dashboard.
type Chart struct {
	Label string
	Type  string // bar, line, pie
	Data  func(db *gorm.DB) (labels []string, values []float64)
}

// NewRegistry initializes a new admin registry.
func NewRegistry(db *gorm.DB) *Registry {
	return &Registry{
		DB:        db,
		Resources: make(map[string]*Resource),
		Pages:     make(map[string]*Page),
		Charts:    []Chart{},
		Config:    DefaultConfig(),
	}
}

func (reg *Registry) SetConfig(config *Config) { reg.Config = config }

func (reg *Registry) AddChart(label, chartType string, provider func(db *gorm.DB) ([]string, []float64)) {
	reg.Charts = append(reg.Charts, Chart{Label: label, Type: chartType, Data: provider})
}

// AddPage registers a custom arbitrary page.
func (reg *Registry) AddPage(name, group string, handler http.HandlerFunc) {
	reg.Pages[name] = &Page{
		Name:    name,
		Group:   group,
		Handler: handler,
	}
}

func (reg *Registry) Register(model interface{}) *Resource {
	resource := NewResource(model)
	reg.Resources[resource.Name] = resource
	fmt.Printf("Registered resource: %s\n", resource.Name)
	return resource
}

func (reg *Registry) GetResource(name string) (*Resource, bool) {
	res, ok := reg.Resources[name]
	return res, ok
}

func (reg *Registry) ResourceNames() []string {
	names := make([]string, 0, len(reg.Resources))
	for name := range reg.Resources {
		names = append(names, name)
	}
	return names
}
