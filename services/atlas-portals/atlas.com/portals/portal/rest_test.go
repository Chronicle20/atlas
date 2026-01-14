package portal_test

import (
	"atlas-portals/portal"
	"atlas-portals/test"
	"testing"
)

func TestExtract_ValidModel(t *testing.T) {
	fixture := test.PortalFixture{
		Id:          "42",
		Name:        "town_portal",
		Target:      "spawn_point",
		Type:        2,
		X:           150,
		Y:           -50,
		TargetMapId: 100000000,
		ScriptName:  "",
	}

	rm := fixture.ToRestModel()
	model, err := portal.Extract(rm)

	if err != nil {
		t.Fatalf("Extract() returned unexpected error: %v", err)
	}

	if model.Id() != 42 {
		t.Errorf("Id() = %d, want 42", model.Id())
	}

	if model.Target() != "spawn_point" {
		t.Errorf("Target() = %s, want spawn_point", model.Target())
	}

	if model.TargetMapId() != 100000000 {
		t.Errorf("TargetMapId() = %d, want 100000000", model.TargetMapId())
	}

	if model.ScriptName() != "" {
		t.Errorf("ScriptName() = %s, want empty string", model.ScriptName())
	}
}

func TestExtract_AllFieldsMapped(t *testing.T) {
	fixture := test.PortalFixture{
		Id:          "99",
		Name:        "test_name",
		Target:      "test_target",
		Type:        5,
		X:           123,
		Y:           456,
		TargetMapId: 200000000,
		ScriptName:  "test_script",
	}

	rm := fixture.ToRestModel()
	model, err := portal.Extract(rm)

	if err != nil {
		t.Fatalf("Extract() returned unexpected error: %v", err)
	}

	if model.Id() != 99 {
		t.Errorf("Id() = %d, want 99", model.Id())
	}

	if model.Target() != "test_target" {
		t.Errorf("Target() = %s, want test_target", model.Target())
	}

	if model.TargetMapId() != 200000000 {
		t.Errorf("TargetMapId() = %d, want 200000000", model.TargetMapId())
	}

	if model.ScriptName() != "test_script" {
		t.Errorf("ScriptName() = %s, want test_script", model.ScriptName())
	}
}

func TestExtract_InvalidId(t *testing.T) {
	rm := portal.RestModel{
		Name:        "test",
		Target:      "",
		Type:        0,
		X:           0,
		Y:           0,
		TargetMapId: 999999999,
		ScriptName:  "",
	}
	rm.SetID("not_a_number")

	_, err := portal.Extract(rm)

	if err == nil {
		t.Error("Extract() expected error for non-numeric ID, got nil")
	}
}

func TestExtract_EmptyId(t *testing.T) {
	rm := portal.RestModel{
		Name:        "test",
		Target:      "",
		Type:        0,
		X:           0,
		Y:           0,
		TargetMapId: 999999999,
		ScriptName:  "",
	}
	rm.SetID("")

	_, err := portal.Extract(rm)

	if err == nil {
		t.Error("Extract() expected error for empty ID, got nil")
	}
}
