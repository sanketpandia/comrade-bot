package connections

import "testing"

func TestNormalizeRecordMultipleSelectsList(t *testing.T) {
	entity := EntityCareerPilots
	mapping := Mapping{Fields: map[string]string{"assignedRoutes": "fldRoutes"}}
	schema := map[string]FieldMeta{"fldRoutes": {ID: "fldRoutes", Name: "Rotation", Type: "multipleSelects"}}
	raw := map[string]any{"Rotation": []any{"ABC-DEF", "GHI-JKL"}}
	rec, issues, err := NormalizeRecord(entity, mapping, schema, raw)
	if err != nil {
		t.Fatal(err)
	}
	val, ok := rec.Fields["assignedRoutes"].([]string)
	if !ok || len(val) != 2 {
		t.Fatalf("fields = %+v", rec.Fields)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestNormalizeRecordScalarFromMultiWarns(t *testing.T) {
	entity := EntityPilots
	mapping := Mapping{Fields: map[string]string{"region": "fldReg"}}
	schema := map[string]FieldMeta{"fldReg": {ID: "fldReg", Name: "Region", Type: "multipleSelects"}}
	raw := map[string]any{"Region": []any{"A", "B"}}
	rec, issues, err := NormalizeRecord(entity, mapping, schema, raw)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Fields["region"] != "A" {
		t.Fatalf("value = %v", rec.Fields["region"])
	}
	if len(issues) != 1 {
		t.Fatalf("issues = %+v", issues)
	}
}