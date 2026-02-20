package admin

import (
	"html/template"
	"reflect"
)

func (reg *Registry) loadTemplates(contentTmpl string) *template.Template {
	return template.Must(template.ParseFS(templateFS, "templates/layout.html", contentTmpl))
}

func (reg *Registry) sliceToMap(res *Resource, fields []Field, slice reflect.Value) []map[string]interface{} {
	var data []map[string]interface{}
	for i := 0; i < slice.Len(); i++ { data = append(data, reg.itemToMap(res, fields, slice.Index(i))) }
	return data
}

func (reg *Registry) itemToMap(res *Resource, fields []Field, item reflect.Value) map[string]interface{} {
	m := make(map[string]interface{})
	item = reflect.Indirect(item)
	for _, f := range fields {
		fv := item.FieldByName(f.Name)
		if fv.IsValid() {
			val := fv.Interface()
			if f.Decorator != nil {
				m[f.Name] = f.Decorator(val)
			} else {
				m[f.Name] = val
			}
		}
	}
	idv := item.FieldByName("ID"); if idv.IsValid() { m["ID"] = idv.Interface() }
	return m
}
