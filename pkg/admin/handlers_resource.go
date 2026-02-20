package admin

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/ajeet-kumar1087/go-admin/pkg/admin/models"
	"github.com/ajeet-kumar1087/go-admin/pkg/admin/resource"
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

func (reg *Registry) renderList(res *resource.Resource, w http.ResponseWriter, r *http.Request, user *models.AdminUser) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fields := res.GetFieldsFor("index")
	page, _ := strconv.Atoi(r.URL.Query().Get("page")); if page < 1 { page = 1 }
	perPage := reg.Config.DefaultPerPage
	currentScope := r.URL.Query().Get("scope")
	query := reg.DB.Model(res.Model)
	if currentScope != "" {
		for _, s := range res.Scopes { if s.Name == currentScope { query = s.Handler(query); break } }
	}
	sortField, sortOrder := r.URL.Query().Get("sort"), r.URL.Query().Get("order")
	if sortField != "" { if sortOrder != "desc" { sortOrder = "asc" }; query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder)) } else { query = query.Order("id desc") }
	filters := make(map[string]string)
	for k, v := range r.URL.Query() {
		val := v[0]; if val == "" { continue }; filters[k] = val
		if strings.HasPrefix(k, "q_") { query = query.Where(fmt.Sprintf("%s LIKE ?", strings.TrimPrefix(k, "q_")), "%"+val+"%") } else if strings.HasPrefix(k, "min_") { query = query.Where(fmt.Sprintf("%s >= ?", strings.TrimPrefix(k, "min_")), val) } else if strings.HasPrefix(k, "max_") { query = query.Where(fmt.Sprintf("%s <= ?", strings.TrimPrefix(k, "max_")), val) }
	}
	var totalCount int64; query.Count(&totalCount)
	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))
	modelType := reflect.TypeOf(res.Model)
	destSlice := reflect.MakeSlice(reflect.SliceOf(modelType), 0, 0); dest := reflect.New(destSlice.Type())
	query.Offset((page - 1) * perPage).Limit(perPage).Find(dest.Interface())
	data := reg.sliceToMap(res, fields, dest.Elem())
	styleContent, _ := templateFS.ReadFile("templates/style.css")
	tmpl := reg.loadTemplates("templates/index.html")
	pd := PageData{
		SiteTitle: reg.Config.SiteTitle, Resources: reg.Resources, GroupedResources: reg.getGroupedResources(), GroupedPages: reg.getGroupedPages(), 
		CurrentResource: res, Fields: fields, Data: data, Filters: filters, User: user, CSS: template.CSS(styleContent),
		Page: page, PerPage: perPage, TotalPages: totalPages, TotalCount: totalCount, HasPrev: page > 1, HasNext: page < totalPages, PrevPage: page - 1, NextPage: page + 1, Scopes: res.Scopes, CurrentScope: currentScope,
		Flash: reg.getFlash(w, r), SortField: sortField, SortOrder: sortOrder,
	}
	tmpl.ExecuteTemplate(w, "index.html", pd)
}

func (reg *Registry) renderShow(res *resource.Resource, item interface{}, w http.ResponseWriter, r *http.Request, user *models.AdminUser) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fields := res.GetFieldsFor("show")
	var itemMap map[string]interface{}
	assocData := make(map[string]AssociationData); renderedSidebars := make(map[string]template.HTML)
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
		for _, sb := range res.Sidebars { renderedSidebars[sb.Label] = sb.Handler(res, item) }
	}
	styleContent, _ := templateFS.ReadFile("templates/style.css")
	tmpl := reg.loadTemplates("templates/show.html")
	pd := PageData{SiteTitle: reg.Config.SiteTitle, Resources: reg.Resources, GroupedResources: reg.getGroupedResources(), GroupedPages: reg.getGroupedPages(), CurrentResource: res, Fields: fields, Item: itemMap, User: user, CSS: template.CSS(styleContent), Associations: assocData, Flash: reg.getFlash(w, r), RenderedSidebars: renderedSidebars}
	tmpl.ExecuteTemplate(w, "show.html", pd)
}

func (reg *Registry) renderForm(res *resource.Resource, item interface{}, w http.ResponseWriter, r *http.Request, user *models.AdminUser) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fields := res.GetFieldsFor("edit")
	var itemMap map[string]interface{}
	if item != nil { itemMap = reg.itemToMap(res, fields, reflect.ValueOf(item)) }
	assocData := make(map[string]AssociationData)
	for _, assoc := range res.Associations {
		if assoc.Type == "BelongsTo" {
			targetRes, _ := reg.GetResource(assoc.ResourceName)
			var count int64; reg.DB.Model(targetRes.Model).Count(&count)
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
	pd := PageData{SiteTitle: reg.Config.SiteTitle, Resources: reg.Resources, GroupedResources: reg.getGroupedResources(), GroupedPages: reg.getGroupedPages(), CurrentResource: res, Fields: fields, Item: itemMap, User: user, CSS: template.CSS(styleContent), Associations: assocData, Flash: reg.getFlash(w, r)}
	tmpl.ExecuteTemplate(w, "form.html", pd)
}

func (reg *Registry) handleSave(res *resource.Resource, w http.ResponseWriter, r *http.Request, user *models.AdminUser) {
	r.ParseMultipartForm(32 << 20)
	model := reflect.New(reflect.TypeOf(res.Model)).Interface()
	isUpdate, id := false, r.FormValue("ID")
	if id != "" && id != "0" { reg.DB.First(model, id); isUpdate = true }
	elem := reflect.ValueOf(model).Elem()
	for _, f := range res.Fields {
		if f.Readonly { continue }
		field := elem.FieldByName(f.Name); if !field.CanSet() { continue }
		if f.Type == "image" || f.Type == "file" {
			file, header, err := r.FormFile(f.Name)
			if err == nil {
				defer file.Close(); os.MkdirAll(reg.Config.UploadDir, 0755)
				newName := fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(header.Filename))
				dst, _ := os.Create(filepath.Join(reg.Config.UploadDir, newName)); defer dst.Close(); io.Copy(dst, file)
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
	reg.setFlash(w, fmt.Sprintf("%s saved successfully", res.Name))
	http.Redirect(w, r, "/admin/"+res.Name, 303)
}

func (reg *Registry) handleBatchAction(res *resource.Resource, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method not allowed", 405); return }
	r.ParseForm(); actionName, ids := r.FormValue("action_name"), r.Form["ids"]
	if actionName == "" || len(ids) == 0 { http.Redirect(w, r, "/admin/"+res.Name, 303); return }
	for _, a := range res.BatchActions { if a.Name == actionName { a.Handler(res, ids, w, r); return } }
}

func (reg *Registry) handleExport(res *resource.Resource, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s_export.csv", res.Name))
	writer := csv.NewWriter(w); defer writer.Flush()
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

func (reg *Registry) handleCustomAction(res *resource.Resource, w http.ResponseWriter, r *http.Request, isCollection bool) {
	actionName := r.URL.Query().Get("name")
	var actions []resource.Action
	if isCollection { actions = res.CollectionActions } else { actions = res.MemberActions }
	for _, a := range actions { if a.Name == actionName { a.Handler(res, w, r); return } }
}

func (reg *Registry) handleSearchAPI(resourceName string, w http.ResponseWriter, r *http.Request) {
	res, ok := reg.GetResource(resourceName); if !ok { http.Error(w, "Not found", 404); return }
	query := r.URL.Query().Get("q"); db := reg.DB.Model(res.Model); searchQuery := ""
	for _, f := range res.Fields { if f.Type == "text" { if searchQuery != "" { searchQuery += " OR " }; searchQuery += fmt.Sprintf("%s LIKE ?", f.Name) } }
	if searchQuery != "" { args := make([]interface{}, strings.Count(searchQuery, "?")); for i := range args { args[i] = "%" + query + "%" }; db = db.Where(searchQuery, args...) }
	var results []map[string]interface{}; modelType := reflect.TypeOf(res.Model)
	destSlice := reflect.MakeSlice(reflect.SliceOf(modelType), 0, 0); dest := reflect.New(destSlice.Type())
	db.Limit(10).Find(dest.Interface()); items := dest.Elem()
	for i := 0; i < items.Len(); i++ {
		item := reflect.Indirect(items.Index(i)); m := make(map[string]interface{})
		m["id"] = item.FieldByName("ID").Interface()
		if f := item.FieldByName("Name"); f.IsValid() { m["text"] = f.Interface() } else if f := item.FieldByName("Email"); f.IsValid() { m["text"] = f.Interface() } else { m["text"] = fmt.Sprintf("ID: %v", m["id"]) }
		results = append(results, m)
	}
	w.Header().Set("Content-Type", "application/json"); json.NewEncoder(w).Encode(results)
}
