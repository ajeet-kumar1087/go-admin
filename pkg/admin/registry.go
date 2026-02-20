package admin

import (
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type Registry struct {
	DB        *gorm.DB
	Resources map[string]*Resource
	Pages     map[string]*Page
	Charts    []Chart
	Config    *Config
}

type Page struct {
	Name, Group string
	Handler     http.HandlerFunc
}

type Chart struct {
	Label string
	Type  string
	Data  func(db *gorm.DB) (labels []string, values []float64)
}

func NewRegistry(db *gorm.DB) *Registry {
	return &Registry{
		DB: db, Resources: make(map[string]*Resource), Pages: make(map[string]*Page), 
		Charts: []Chart{}, Config: DefaultConfig(),
	}
}

func (reg *Registry) SetConfig(c *Config) { reg.Config = c }

func (reg *Registry) AddChart(l, t string, p func(db *gorm.DB) ([]string, []float64)) {
	reg.Charts = append(reg.Charts, Chart{Label: l, Type: t, Data: p})
}

func (reg *Registry) AddPage(n, g string, h http.HandlerFunc) {
	reg.Pages[n] = &Page{Name: n, Group: g, Handler: h}
}

func (reg *Registry) Register(m interface{}) *Resource {
	res := NewResource(m)
	reg.Resources[res.Name] = res
	fmt.Printf("Registered resource: %s\n", res.Name)
	return res
}

func (reg *Registry) GetResource(n string) (*Resource, bool) {
	res, ok := reg.Resources[n]; return res, ok
}

func (reg *Registry) ResourceNames() []string {
	names := make([]string, 0, len(reg.Resources))
	for n := range reg.Resources { names = append(names, n) }
	return names
}

func (reg *Registry) getGroupedResources() map[string][]*Resource {
	groups := make(map[string][]*Resource)
	for _, r := range reg.Resources {
		g := r.Group; if g == "" { g = "Default" }; groups[g] = append(groups[g], r)
	}
	return groups
}

func (reg *Registry) getGroupedPages() map[string][]*Page {
	groups := make(map[string][]*Page)
	for _, p := range reg.Pages {
		g := p.Group; if g == "" { g = "Default" }; groups[g] = append(groups[g], p)
	}
	return groups
}

func (reg *Registry) RecordAction(user *AdminUser, resName, recordID, action, changes string) {
	reg.DB.Create(&AuditLog{
		UserID: user.ID, UserEmail: user.Email, ResourceName: resName, 
		RecordID: recordID, Action: action, Changes: changes, CreatedAt: time.Now(),
	})
}
