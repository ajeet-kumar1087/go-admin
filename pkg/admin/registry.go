package admin

import (
	"fmt"
	"github.com/ajeet-kumar1087/go-admin/pkg/admin/config"
	"github.com/ajeet-kumar1087/go-admin/pkg/admin/models"
	"github.com/ajeet-kumar1087/go-admin/pkg/admin/resource"
	"gorm.io/gorm"
	"net/http"
	"time"
)

// Public Type Aliases
type Resource = resource.Resource
type Field = resource.Field
type Config = config.Config
type AdminUser = models.AdminUser
type Session = models.Session
type Permission = models.Permission
type AuditLog = models.AuditLog
type Scope = resource.Scope

type Registry struct {
	DB        *gorm.DB
	Resources map[string]*resource.Resource
	Pages     map[string]*Page
	Charts    []Chart
	Config    *config.Config
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
		DB: db, Resources: make(map[string]*resource.Resource), Pages: make(map[string]*Page), 
		Charts: []Chart{}, Config: config.DefaultConfig(),
	}
}

// Public Factory Functions (Facades)
func DefaultConfig() *config.Config { return config.DefaultConfig() }
func LoadConfig(path string) (*config.Config, error) { return config.LoadConfig(path) }
func NewResource(model interface{}) *resource.Resource { return resource.NewResource(model) }

func (reg *Registry) SetConfig(c *config.Config) { reg.Config = c }

func (reg *Registry) AddChart(l, t string, p func(db *gorm.DB) ([]string, []float64)) {
	reg.Charts = append(reg.Charts, Chart{Label: l, Type: t, Data: p})
}

func (reg *Registry) AddPage(n, g string, h http.HandlerFunc) {
	reg.Pages[n] = &Page{Name: n, Group: g, Handler: h}
}

func (reg *Registry) Register(m interface{}) *resource.Resource {
	res := resource.NewResource(m)
	reg.Resources[res.Name] = res
	fmt.Printf("Registered resource: %s\n", res.Name)
	return res
}

func (reg *Registry) GetResource(n string) (*resource.Resource, bool) {
	res, ok := reg.Resources[n]; return res, ok
}

func (reg *Registry) ResourceNames() []string {
	names := make([]string, 0, len(reg.Resources))
	for n := range reg.Resources { names = append(names, n) }
	return names
}

func (reg *Registry) getGroupedResources() map[string][]*resource.Resource {
	groups := make(map[string][]*resource.Resource)
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

func (reg *Registry) RecordAction(user *models.AdminUser, resName, recordID, action, changes string) {
	reg.DB.Create(&models.AuditLog{
		UserID: user.ID, UserEmail: user.Email, ResourceName: resName, 
		RecordID: recordID, Action: action, Changes: changes, CreatedAt: time.Now(),
	})
}
