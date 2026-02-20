package admin

import (
	"testing"
)

func TestResourceFieldConfig(t *testing.T) {
	res := NewResource(TestModel{})
	res.RegisterField("ID", "Identifier", true)
	res.RegisterField("Name", "Full Name", false)

	if len(res.Fields) != 2 {
		t.Errorf("Expected 2 fields")
	}

	// Test View Filtering
	res.SetIndexFields("Name")
	indexFields := res.GetFieldsFor("index")
	if len(indexFields) != 1 || indexFields[0].Name != "Name" {
		t.Errorf("Index fields not correctly filtered")
	}

	showFields := res.GetFieldsFor("show")
	if len(showFields) != 2 {
		t.Errorf("Show fields should default to all if not set")
	}
}

func TestResourceAssociations(t *testing.T) {
	res := NewResource(TestModel{})
	res.HasMany("Items", "Items List", "OtherModel", "ModelID")
	res.BelongsTo("ParentID", "Parent", "ParentModel", "ID")

	if len(res.Associations) != 2 {
		t.Errorf("Expected 2 associations")
	}

	if res.Associations[0].Type != "HasMany" {
		t.Errorf("First association should be HasMany")
	}
}

func TestResourceActions(t *testing.T) {
	res := NewResource(TestModel{})
	res.AddMemberAction("test", "Test Action", nil)
	res.AddCollectionAction("coll", "Coll Action", nil)
	res.AddBatchAction("batch", "Batch Action", nil)

	if len(res.MemberActions) != 1 {
		t.Errorf("Expected 1 member action")
	}
	if len(res.CollectionActions) != 1 {
		t.Errorf("Expected 1 collection action")
	}
	if len(res.BatchActions) != 1 {
		t.Errorf("Expected 1 batch action")
	}
}
