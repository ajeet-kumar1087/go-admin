package admin

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"html/template"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
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
	CSS              template.CSS
	Page             int
	PerPage          int
	TotalPages       int
	TotalCount       int64
	HasPrev          bool
	HasNext          bool
	PrevPage         int
	NextPage         int
	Scopes           []Scope
	CurrentScope     string
	Associations     map[string]AssociationData
	ChartData        []ChartWidget
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
	if strings.HasSuffix(upath, "/search") {
		parts := strings.Split(strings.TrimPrefix(upath, "/"), "/")
		reg.handleSearchAPI(parts[0], w, r); return
	}
	if upath == "" || upath == "/" { reg.renderDashboard(w, r, user); return }

	parts := strings.Split(strings.TrimPrefix(upath, "/"), "/")
	resourceName := parts[0]
	
	// Check if it's a Custom Page first
	if page, ok := reg.Pages[resourceName]; ok {
		page.Handler(w, r)
		return
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
		http.Redirect(w, r, "/admin/"+res.Name, 303)
	default: reg.renderList(res, w, r, user)
	}
}

// RenderCustomPage is a helper for developers to render content within the admin layout.
func (reg *Registry) RenderCustomPage(w http.ResponseWriter, r *http.Request, title string, content template.HTML) {
	user, _ := reg.GetUserFromRequest(r)
	styleContent, _ := templateFS.ReadFile("templates/style.css")
	
	// We'll use a dynamic template for custom pages
	tmpl := template.Must(template.ParseFS(templateFS, "templates/layout.html"))
	tmpl = template.Must(tmpl.New("title").Parse(title))
	tmpl = template.Must(tmpl.New("content").Parse(`<div style="padding: 2rem;">` + string(content) + `</div>`))
	
	pd := PageData{
		SiteTitle: reg.Config.SiteTitle, GroupedResources: reg.getGroupedResources(), GroupedPages: reg.getGroupedPages(),
		User: user, CSS: template.CSS(styleContent),
	}
	tmpl.ExecuteTemplate(w, "layout", pd)
}

func (reg *Registry) getGroupedPages() map[string][]*Page {
	groups := make(map[string][]*Page)
	for _, p := range reg.Pages {
		g := p.Group; if g == "" { g = "Default" }; groups[g] = append(groups[g], p)
	}
	return groups
}

// ... (Rest of the methods handleSave, renderList, etc. updated to include GroupedPages in PageData)

func (reg *Registry) renderDashboard(w http.ResponseWriter, r *http.Request, user *AdminUser) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var stats []Stat
	for name, res := range reg.Resources {
		var count int64
		reg.DB.Model(res.Model).Count(&count)
		stats = append(stats, Stat{Label: name, Value: count})
	}
	var widgets []ChartWidget
	for i, c := range reg.Charts {
		l, v := c.Data(reg.DB)
		widgets = append(widgets, ChartWidget{ID: fmt.Sprintf("chart-%d", i), Label: c.Label, Type: c.Type, Labels: l, Values: v})
	}
	styleContent, _ := templateFS.ReadFile("templates/style.css")
	tmpl := reg.loadTemplates("templates/dashboard.html")
	pd := PageData{SiteTitle: reg.Config.SiteTitle, GroupedResources: reg.getGroupedResources(), GroupedPages: reg.getGroupedPages(), User: user, Stats: stats, CSS: template.CSS(styleContent), ChartData: widgets}
	tmpl.ExecuteTemplate(w, "dashboard.html", pd)
}

func (reg *Registry) renderShow(res *Resource, item interface{}, w http.ResponseWriter, r *http.Request, user *AdminUser) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fields := res.GetFieldsFor("show")
	var itemMap map[string]interface{}
	assocData := make(map[string]AssociationData)
	if item != nil {
		itemMap = reg.itemToMap(res, fields, reflect.ValueOf(item))
		for _, assoc := range res.Associations {
			if assoc.Type == "HasMany" {
				targetRes, _ := reg.GetResource(assoc.ResourceName)
				targetFields := targetRes.GetFieldsFor("index")
				modelType := reflect.TypeOf(targetRes.Model)
				destSlice := reflect.MakeSlice(reflect.SliceOf(modelType), 0, 0); dest := reflect.New(destSlice.Type())
				reg.DB.Where(fmt.Sprintf("%s = ?", assoc.ForeignKey), itemMap["ID"]).Find(dest.Interface())
				assocData[assoc.Name] = AssociationData{Resource: targetRes, Fields: targetFields, Items: reg.sliceToMap(targetRes, targetFields, dest.Elem())}
			}
		}
	}
	styleContent, _ := templateFS.ReadFile("templates/style.css")
	tmpl := reg.loadTemplates("templates/show.html")
	pd := PageData{SiteTitle: reg.Config.SiteTitle, GroupedResources: reg.getGroupedResources(), GroupedPages: reg.getGroupedPages(), CurrentResource: res, Fields: fields, Item: itemMap, User: user, CSS: template.CSS(styleContent), Associations: assocData}
	tmpl.ExecuteTemplate(w, "show.html", pd)
}

func (reg *Registry) renderList(res *Resource, w http.ResponseWriter, r *http.Request, user *AdminUser) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fields := res.GetFieldsFor("index")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 { page = 1 }
	perPage := reg.Config.DefaultPerPage
	currentScope := r.URL.Query().Get("scope")
	query := reg.DB.Model(res.Model)
	if currentScope != "" {
		for _, s := range res.Scopes { if s.Name == currentScope { query = s.Handler(query); break } }
	}
	filters := make(map[string]string)
	for k, v := range r.URL.Query() {
		val := v[0]
		if val == "" { continue }
		filters[k] = val
		if strings.HasPrefix(k, "q_") { query = query.Where(fmt.Sprintf("%s LIKE ?", strings.TrimPrefix(k, "q_")), "%"+val+"%") } else if strings.HasPrefix(k, "min_") { query = query.Where(fmt.Sprintf("%s >= ?", strings.TrimPrefix(k, "min_")), val) } else if strings.HasPrefix(k, "max_") { query = query.Where(fmt.Sprintf("%s <= ?", strings.TrimPrefix(k, "max_")), val) }
	}
	var totalCount int64
	query.Count(&totalCount)
	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))
	modelType := reflect.TypeOf(res.Model)
	destSlice := reflect.MakeSlice(reflect.SliceOf(modelType), 0, 0); dest := reflect.New(destSlice.Type())
	query.Offset((page - 1) * perPage).Limit(perPage).Find(dest.Interface())
	data := reg.sliceToMap(res, fields, dest.Elem())
	styleContent, _ := templateFS.ReadFile("templates/style.css")
	tmpl := reg.loadTemplates("templates/index.html")
	pd := PageData{SiteTitle: reg.Config.SiteTitle, GroupedResources: reg.getGroupedResources(), GroupedPages: reg.getGroupedPages(), CurrentResource: res, Fields: fields, Data: data, Filters: filters, User: user, CSS: template.CSS(styleContent), Page: page, PerPage: perPage, TotalPages: totalPages, TotalCount: totalCount, HasPrev: page > 1, HasNext: page < totalPages, PrevPage: page - 1, NextPage: page + 1, Scopes: res.Scopes, CurrentScope: currentScope}
	tmpl.ExecuteTemplate(w, "index.html", pd)
}

func (reg *Registry) renderForm(res *Resource, item interface{}, w http.ResponseWriter, r *http.Request, user *AdminUser) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fields := res.GetFieldsFor("edit")
	var itemMap map[string]interface{}
	if item != nil { itemMap = reg.itemToMap(res, fields, reflect.ValueOf(item)) }
	assocData := make(map[string]AssociationData)
	for _, assoc := range res.Associations {
		if assoc.Type == "BelongsTo" {
			targetRes, _ := reg.GetResource(assoc.ResourceName)
			var count int64
			reg.DB.Model(targetRes.Model).Count(&count)
			if count < reg.Config.SearchThreshold {
				modelType := reflect.TypeOf(targetRes.Model)
				destSlice := reflect.MakeSlice(reflect.SliceOf(modelType), 0, 0); dest := reflect.New(destSlice.Type())
				reg.DB.Find(dest.Interface())
				assocData[assoc.Name] = AssociationData{Resource: targetRes, Options: reg.sliceToMap(targetRes, targetRes.Fields, dest.Elem())}
			} else { assocData[assoc.Name] = AssociationData{Resource: targetRes} }
		}
	}
	for _, f := range fields { if f.Searchable && f.SearchResource != "" { targetRes, _ := reg.GetResource(f.SearchResource); assocData[f.Name] = AssociationData{Resource: targetRes} } }
	styleContent, _ := templateFS.ReadFile("templates/style.css")
	tmpl := reg.loadTemplates("templates/form.html")
	pd := PageData{SiteTitle: reg.Config.SiteTitle, GroupedResources: reg.getGroupedResources(), GroupedPages: reg.getGroupedPages(), CurrentResource: res, Fields: fields, Item: itemMap, User: user, CSS: template.CSS(styleContent), Associations: assocData}
	tmpl.ExecuteTemplate(w, "form.html", pd)
}

func (reg *Registry) handleSave(res *Resource, w http.ResponseWriter, r *http.Request, user *AdminUser) {
	r.ParseMultipartForm(32 << 20)
	model := reflect.New(reflect.TypeOf(res.Model)).Interface()
	isUpdate, id := false, r.FormValue("ID")
	if id != "" && id != "0" { reg.DB.First(model, id); isUpdate = true }
	elem := reflect.ValueOf(model).Elem()
	for _, f := range res.Fields {
		if f.Readonly { continue }
		field := elem.FieldByName(f.Name)
		if !field.CanSet() { continue }
		if f.Type == "image" || f.Type == "file" {
			file, header, err := r.FormFile(f.Name)
			if err == nil {
				defer file.Close()
				os.MkdirAll(reg.Config.UploadDir, 0755)
				newName := fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(header.Filename))
				dst, _ := os.Create(filepath.Join(reg.Config.UploadDir, newName))
				defer dst.Close(); io.Copy(dst, file)
				field.SetString("/admin/uploads/" + newName)
			}
			continue
		}
		val := r.FormValue(f.Name)
		if field.Kind() == reflect.Float64 { fv, _ := strconv.ParseFloat(val, 64); field.SetFloat(fv) } else if field.Kind() == reflect.Uint { uv, _ := strconv.ParseUint(val, 10, 64); field.SetUint(uv) } else { field.SetString(val) }
	}
	reg.DB.Save(model)
	newID := fmt.Sprintf("%v", elem.FieldByName("ID").Interface())
	act := "Create"; if isUpdate { act = "Update" }
	reg.RecordAction(user, res.Name, newID, act, "Saved from form")
	http.Redirect(w, r, "/admin/"+res.Name, 303)
}

func (reg *Registry) handleBatchAction(res *Resource, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method not allowed", 405); return }
	r.ParseForm()
	actionName, ids := r.FormValue("action_name"), r.Form["ids"]
	if actionName == "" || len(ids) == 0 { http.Redirect(w, r, "/admin/"+res.Name, 303); return }
	for _, a := range res.BatchActions { if a.Name == actionName { a.Handler(res, ids, w, r); return } }
}

func (reg *Registry) handleExport(res *Resource, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s_export.csv", res.Name))
	writer := csv.NewWriter(w)
	defer writer.Flush()
	var h []string; for _, f := range res.Fields { h = append(h, f.Label) }; writer.Write(h)
	query := reg.DB.Model(res.Model)
	modelType := reflect.TypeOf(res.Model)
	destSlice := reflect.MakeSlice(reflect.SliceOf(modelType), 0, 0); dest := reflect.New(destSlice.Type())
	query.Find(dest.Interface()); items := dest.Elem()
	for i := 0; i < items.Len(); i++ {
		item := reflect.Indirect(items.Index(i)); var row []string
		for _, f := range res.Fields { row = append(row, fmt.Sprintf("%v", item.FieldByName(f.Name).Interface())) }
		writer.Write(row)
	}
}

func (reg *Registry) handleCustomAction(res *Resource, w http.ResponseWriter, r *http.Request, isCollection bool) {
	actionName := r.URL.Query().Get("name")
	var actions []Action
	if isCollection { actions = res.CollectionActions } else { actions = res.MemberActions }
	for _, a := range actions { if a.Name == actionName { a.Handler(res, w, r); return } }
}

func (reg *Registry) handleLogin(w http.ResponseWriter, r *http.Request) {
	email, password := r.FormValue("email"), r.FormValue("password")
	var user AdminUser
	if err := reg.DB.Where("email = ?", email).First(&user).Error; err != nil { reg.renderLogin(w, r, "Invalid credentials"); return }
	if !user.CheckPassword(password) { reg.renderLogin(w, r, "Invalid credentials"); return }
	sessionID := uuid.New().String()
	reg.DB.Create(&Session{ID: sessionID, UserID: user.ID, ExpiresAt: time.Now().Add(time.Duration(reg.Config.SessionTTL) * time.Hour)})
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: sessionID, Path: "/admin", HttpOnly: true})
	http.Redirect(w, r, "/admin", 303)
}

func (reg *Registry) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("admin_session")
	if cookie != nil { reg.DB.Delete(&Session{}, "id = ?", cookie.Value) }
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "", Path: "/admin", Expires: time.Unix(0, 0), HttpOnly: true})
	http.Redirect(w, r, "/admin/login", 303)
}

func (reg *Registry) renderLogin(w http.ResponseWriter, r *http.Request, errorMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, _ := template.ParseFS(templateFS, "templates/login.html")
	styleContent, _ := templateFS.ReadFile("templates/style.css")
	tmpl.Execute(w, PageData{SiteTitle: reg.Config.SiteTitle, Error: errorMsg, CSS: template.CSS(styleContent)})
}

func (reg *Registry) loadTemplates(contentTmpl string) *template.Template {
	return template.Must(template.ParseFS(templateFS, "templates/layout.html", contentTmpl))
}

func (reg *Registry) getGroupedResources() map[string][]*Resource {
	groups := make(map[string][]*Resource)
	for _, res := range reg.Resources { g := res.Group; if g == "" { g = "Default" }; groups[g] = append(groups[g], res) }
	return groups
}

func (reg *Registry) sliceToMap(res *Resource, fields []Field, slice reflect.Value) []map[string]interface{} {
	var data []map[string]interface{}
	for i := 0; i < slice.Len(); i++ { data = append(data, reg.itemToMap(res, fields, slice.Index(i))) }
	return data
}

func (reg *Registry) itemToMap(res *Resource, fields []Field, item reflect.Value) map[string]interface{} {
	m := make(map[string]interface{})
	item = reflect.Indirect(item)
	for _, f := range fields { fv := item.FieldByName(f.Name); if fv.IsValid() { m[f.Name] = fv.Interface() } }
	idv := item.FieldByName("ID"); if idv.IsValid() { m["ID"] = idv.Interface() }
	return m
}
