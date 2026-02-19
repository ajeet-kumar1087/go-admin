package admin

import (
	"gorm.io/gorm"
	"net/http"
	"reflect"
)

type ActionHandler func(res *Resource, w http.ResponseWriter, r *http.Request)
type BatchActionHandler func(res *Resource, ids []string, w http.ResponseWriter, r *http.Request)
type ScopeFunc func(db *gorm.DB) *gorm.DB

type Action struct {
	Name    string
	Label   string
	Handler ActionHandler
}

type BatchAction struct {
	Name    string
	Label   string
	Handler BatchActionHandler
}

type Scope struct {
	Name    string
	Label   string
	Handler ScopeFunc
}

type Association struct {
	Type         string // HasMany, BelongsTo
	Name         string // The field name in the struct
	ResourceName string // The target resource name
	ForeignKey   string // The joining key
	Label        string
}

type Resource struct {
	Model             interface{}
	Name              string
	Path              string
	Group             string
	Fields            []Field
	IndexFields       []string
	ShowFields        []string
	EditFields        []string
	MemberActions     []Action
	CollectionActions []Action
	BatchActions      []BatchAction
	Scopes            []Scope
	Associations      []Association
	Attributes        map[string]interface{}
}

type Field struct {
	Name           string
	Label          string
	Type           string // text, select, number
	Options        []string
	Readonly       bool
	Searchable     bool   // Manual override for searchable select
	SearchResource string // Resource to search against
}

func NewResource(model interface{}) *Resource {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr { t = t.Elem() }
	name := t.Name()
	return &Resource{Model: model, Name: name, Path: "/" + name}
}

func (r *Resource) SetGroup(group string) *Resource {
	r.Group = group
	return r
}

func (r *Resource) RegisterField(name string, label string, readonly bool) *Resource {
	r.Fields = append(r.Fields, Field{Name: name, Label: label, Type: "text", Readonly: readonly})
	return r
}

func (r *Resource) AddMemberAction(name, label string, handler ActionHandler) *Resource {
	r.MemberActions = append(r.MemberActions, Action{Name: name, Label: label, Handler: handler})
	return r
}

func (r *Resource) AddCollectionAction(name, label string, handler ActionHandler) *Resource {
	r.CollectionActions = append(r.CollectionActions, Action{Name: name, Label: label, Handler: handler})
	return r
}

func (r *Resource) AddBatchAction(name, label string, handler BatchActionHandler) *Resource {
	r.BatchActions = append(r.BatchActions, BatchAction{Name: name, Label: label, Handler: handler})
	return r
}

func (r *Resource) AddScope(name, label string, handler ScopeFunc) *Resource {
	r.Scopes = append(r.Scopes, Scope{Name: name, Label: label, Handler: handler})
	return r
}

func (r *Resource) HasMany(name, label, targetResource, foreignKey string) *Resource {
	r.Associations = append(r.Associations, Association{
		Type: "HasMany", Name: name, Label: label, ResourceName: targetResource, ForeignKey: foreignKey,
	})
	return r
}

func (r *Resource) BelongsTo(name, label, targetResource, foreignKey string) *Resource {
	r.Associations = append(r.Associations, Association{
		Type: "BelongsTo", Name: name, Label: label, ResourceName: targetResource, ForeignKey: foreignKey,
	})
	return r
}

// SetSearchable forces a field to use the Select2-style searchable input.
func (r *Resource) SetSearchable(fieldName string, targetResource string) *Resource {
	for i, f := range r.Fields {
		if f.Name == fieldName {
			r.Fields[i].Searchable = true
			r.Fields[i].SearchResource = targetResource
			break
		}
	}
	return r
}

func (r *Resource) SetFieldType(name string, fieldType string, options ...string) *Resource {
	for i, f := range r.Fields {
		if f.Name == name {
			r.Fields[i].Type = fieldType
			r.Fields[i].Options = options
			break
		}
	}
	return r
}

func (r *Resource) SetIndexFields(names ...string) *Resource { r.IndexFields = names; return r }
func (r *Resource) SetShowFields(names ...string) *Resource { r.ShowFields = names; return r }
func (r *Resource) SetEditFields(names ...string) *Resource { r.EditFields = names; return r }

func (r *Resource) GetFieldsFor(view string) []Field {
	var names []string
	switch view {
	case "index": names = r.IndexFields
	case "show": names = r.ShowFields
	case "edit": names = r.EditFields
	}
	if len(names) == 0 { return r.Fields }
	var result []Field
	for _, name := range names {
		for _, f := range r.Fields {
			if f.Name == name { result = append(result, f); break }
		}
	}
	return result
}
