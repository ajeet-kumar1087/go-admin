package admin

import (
	"fmt"
	"html/template"
	"net/http"
)

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
	pd := PageData{
		SiteTitle: reg.Config.SiteTitle, GroupedResources: reg.getGroupedResources(), GroupedPages: reg.getGroupedPages(), 
		User: user, Stats: stats, CSS: template.CSS(styleContent), ChartData: widgets,
		Flash: reg.getFlash(w, r),
	}
	tmpl.ExecuteTemplate(w, "dashboard.html", pd)
}

func (reg *Registry) RenderCustomPage(w http.ResponseWriter, r *http.Request, title string, content template.HTML) {
	user, _ := reg.GetUserFromRequest(r)
	styleContent, _ := templateFS.ReadFile("templates/style.css")
	tmpl := template.Must(template.ParseFS(templateFS, "templates/layout.html"))
	tmpl = template.Must(tmpl.New("title").Parse(title))
	tmpl = template.Must(tmpl.New("content").Parse(`<div style="padding: 2rem;">` + string(content) + `</div>`))
	pd := PageData{
		SiteTitle: reg.Config.SiteTitle, GroupedResources: reg.getGroupedResources(), GroupedPages: reg.getGroupedPages(),
		User: user, CSS: template.CSS(styleContent),
		Flash: reg.getFlash(w, r),
	}
	tmpl.ExecuteTemplate(w, "layout", pd)
}
