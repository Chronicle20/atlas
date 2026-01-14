package portal_test

import (
	"atlas-portals/portal"
	"atlas-portals/test"
	"testing"
)

func TestModel_HasScript(t *testing.T) {
	tests := []struct {
		name       string
		scriptName string
		want       bool
	}{
		{"with script", "portal_script", true},
		{"with different script", "another_script", true},
		{"without script", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := test.DefaultPortalFixture()
			fixture.ScriptName = tt.scriptName
			rm := fixture.ToRestModel()

			model, err := portal.Extract(rm)
			if err != nil {
				t.Fatalf("Extract() returned unexpected error: %v", err)
			}

			if got := model.HasScript(); got != tt.want {
				t.Errorf("HasScript() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModel_HasTargetMap(t *testing.T) {
	tests := []struct {
		name        string
		targetMapId uint32
		want        bool
	}{
		{"with target map", 100000000, true},
		{"with another target map", 200000000, true},
		{"with target map id 0", 0, true},
		{"without target map (999999999)", 999999999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := test.DefaultPortalFixture()
			fixture.TargetMapId = tt.targetMapId
			rm := fixture.ToRestModel()

			model, err := portal.Extract(rm)
			if err != nil {
				t.Fatalf("Extract() returned unexpected error: %v", err)
			}

			if got := model.HasTargetMap(); got != tt.want {
				t.Errorf("HasTargetMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModel_String(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		portalNm string
		want     string
	}{
		{"basic portal", "1", "spawn", "1 - spawn"},
		{"numbered portal", "42", "portal_42", "42 - portal_42"},
		{"empty name", "5", "", "5 - "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := test.DefaultPortalFixture()
			fixture.Id = tt.id
			fixture.Name = tt.portalNm
			rm := fixture.ToRestModel()

			model, err := portal.Extract(rm)
			if err != nil {
				t.Fatalf("Extract() returned unexpected error: %v", err)
			}

			if got := model.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModel_ScriptName(t *testing.T) {
	fixture := test.PortalWithScript("my_portal_script")
	rm := fixture.ToRestModel()

	model, err := portal.Extract(rm)
	if err != nil {
		t.Fatalf("Extract() returned unexpected error: %v", err)
	}

	if got := model.ScriptName(); got != "my_portal_script" {
		t.Errorf("ScriptName() = %q, want %q", got, "my_portal_script")
	}
}

func TestModel_Target(t *testing.T) {
	fixture := test.PortalWithTarget(100000000, "target_portal_name")
	rm := fixture.ToRestModel()

	model, err := portal.Extract(rm)
	if err != nil {
		t.Fatalf("Extract() returned unexpected error: %v", err)
	}

	if got := model.Target(); got != "target_portal_name" {
		t.Errorf("Target() = %q, want %q", got, "target_portal_name")
	}
}

func TestModel_TargetMapId(t *testing.T) {
	fixture := test.PortalWithTarget(123456789, "target")
	rm := fixture.ToRestModel()

	model, err := portal.Extract(rm)
	if err != nil {
		t.Fatalf("Extract() returned unexpected error: %v", err)
	}

	if got := model.TargetMapId(); got != 123456789 {
		t.Errorf("TargetMapId() = %d, want %d", got, 123456789)
	}
}

func TestModel_Id(t *testing.T) {
	fixture := test.DefaultPortalFixture()
	fixture.Id = "777"
	rm := fixture.ToRestModel()

	model, err := portal.Extract(rm)
	if err != nil {
		t.Fatalf("Extract() returned unexpected error: %v", err)
	}

	if got := model.Id(); got != 777 {
		t.Errorf("Id() = %d, want %d", got, 777)
	}
}
