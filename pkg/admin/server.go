package admin

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed templates/*
var templateFS embed.FS

type PageData struct {
	SiteTitle        string
	Resources        map[string]*Resource
	GroupedResources map[string][]*Resource
	GroupedPages     map[string][]*Page
	CurrentResource  *Resource
	Fields           []Field
	Data             []map[string]interface{}
	Item             map[string]interface{}
	Filters          map[string]string
	User             *AdminUser
	Stats            []Stat
	Error            string
	Flash            string
	CSS              template.CSS
	Page, PerPage    int
	TotalPages       int
	TotalCount       int64
	HasPrev, HasNext bool
	PrevPage, NextPage int
	Scopes           []Scope
	CurrentScope     string
	Associations     map[string]AssociationData
	ChartData        []ChartWidget
	SortField        string
	SortOrder        string
	RenderedSidebars map[string]template.HTML
}

type ChartWidget struct {
	ID, Label, Type string
	Labels          []string
	Values          []float64
}

type AssociationData struct {
	Resource *Resource
	Fields   []Field
	Items    []map[string]interface{}
	Options  []map[string]interface{}
}

type Stat struct {
	Label string
	Value int64
}

func (reg *Registry) setFlash(w http.ResponseWriter, message string) {
	http.SetCookie(w, &http.Cookie{Name: "admin_flash", Value: message, Path: "/admin", HttpOnly: true})
}

func (reg *Registry) getFlash(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("admin_flash")
	if err != nil { return "" }
	http.SetCookie(w, &http.Cookie{Name: "admin_flash", Value: "", Path: "/admin", MaxAge: -1})
	return cookie.Value
}

func (reg *Registry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := strings.TrimPrefix(r.URL.Path, "/admin")
	
	if strings.HasPrefix(upath, "/uploads/") {
		http.ServeFile(w, r, filepath.Join(reg.Config.UploadDir, strings.TrimPrefix(upath, "/uploads/")))
		return
	}

	user, role := reg.GetUserFromRequest(r)

	if upath == "/login" {
		if r.Method == "POST" { reg.handleLogin(w, r); return }
		reg.renderLogin(w, r, ""); return
	}
	if upath == "/logout" { reg.handleLogout(w, r); return }
	if user == nil { http.Redirect(w, r, "/admin/login", 303); return }

	if upath == "" || upath == "/" { reg.renderDashboard(w, r, user); return }

	if strings.HasSuffix(upath, "/search") {
		parts := strings.Split(strings.TrimPrefix(upath, "/"), "/")
		reg.handleSearchAPI(parts[0], w, r); return
	}

	parts := strings.Split(strings.TrimPrefix(upath, "/"), "/")
	resourceName := parts[0]

	if page, ok := reg.Pages[resourceName]; ok {
		page.Handler(w, r); return
	}

	res, ok := reg.GetResource(resourceName)
	if !ok { http.NotFound(w, r); return }

	action := "list"
	if len(parts) > 1 && parts[1] != "" { action = parts[1] }

	if !reg.IsAllowed(role, resourceName, action) && 
	   action != "export" && action != "action" && action != "collection_action" && action != "batch_action" {
		http.Error(w, "Forbidden", 403); return
	}

	switch action {
	case "export": reg.handleExport(res, w, r)
	case "action": reg.handleCustomAction(res, w, r, false)
	case "collection_action": reg.handleCustomAction(res, w, r, true)
	case "batch_action": reg.handleBatchAction(res, w, r)
	case "save": reg.handleSave(res, w, r, user)
	case "new": reg.renderForm(res, nil, w, r, user)
	case "show":
		id := r.URL.Query().Get("id")
		item, _ := reg.Get(res.Name, id)
		reg.renderShow(res, item, w, r, user)
	case "edit":
		id := r.URL.Query().Get("id")
		item, _ := reg.Get(res.Name, id)
		reg.renderForm(res, item, w, r, user)
	case "delete":
		id := r.URL.Query().Get("id")
		reg.Delete(res.Name, id)
		reg.RecordAction(user, res.Name, id, "Delete", "Record deleted")
		reg.setFlash(w, fmt.Sprintf("%s deleted successfully", res.Name))
		http.Redirect(w, r, "/admin/"+res.Name, 303)
	default: reg.renderList(res, w, r, user)
	}
}
