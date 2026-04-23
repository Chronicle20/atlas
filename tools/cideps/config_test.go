package main

import "testing"

func TestLoadConfig_Simple(t *testing.T) {
	cfg, err := LoadConfig("testdata/simple/services.json")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Services) != 1 || cfg.Services[0].Name != "svc-a" {
		t.Errorf("services=%+v", cfg.Services)
	}
	if len(cfg.Libraries) != 2 {
		t.Errorf("libraries=%+v", cfg.Libraries)
	}
}

func TestEnrich_DockerContextFallback(t *testing.T) {
	cfg, err := LoadConfig("testdata/transitive/services.json")
	if err != nil {
		t.Fatal(err)
	}
	svcRows := cfg.EnrichDockerServices([]string{"svc-a", "svc-b"})
	got := make(map[string]string)
	for _, r := range svcRows {
		got[r.Name] = r.DockerContext
	}
	if got["svc-a"] != "." {
		t.Errorf("svc-a docker_context=%q want .", got["svc-a"])
	}
	if got["svc-b"] != "services/svc-b" {
		t.Errorf("svc-b docker_context=%q want services/svc-b (fallback to path)", got["svc-b"])
	}
}

func TestEnrich_GoServicesFiltersType(t *testing.T) {
	cfg := &Config{
		Services: []ServiceEntry{
			{Name: "a", Type: "go-service", Path: "p", ModulePath: "mp", DockerImage: "di"},
			{Name: "b", Type: "static-service", Path: "p", DockerImage: "di"},
		},
	}
	rows := cfg.EnrichGoServices([]string{"a", "b"})
	if len(rows) != 1 || rows[0].Name != "a" {
		t.Errorf("rows=%+v want only go-service a", rows)
	}
}

// Non-Go services (type=node-service, type=static-service) must still land in
// the docker-services matrix when they're in the affected set — their Docker
// images are built by the same pipeline.
func TestEnrichDockerServices_IncludesNonGoServices(t *testing.T) {
	cfg := &Config{
		Services: []ServiceEntry{
			{Name: "atlas-ui", Type: "node-service", Path: "services/atlas-ui", DockerImage: "ghcr.io/x/ui", DockerContext: "."},
			{Name: "atlas-assets", Type: "static-service", Path: "services/atlas-assets", DockerImage: "ghcr.io/x/assets"},
			{Name: "atlas-account", Type: "go-service", Path: "services/atlas-account", ModulePath: "services/atlas-account/atlas.com/account", DockerImage: "ghcr.io/x/account"},
		},
	}
	rows := cfg.EnrichDockerServices([]string{"atlas-ui", "atlas-assets", "atlas-account"})
	got := make(map[string]string)
	for _, r := range rows {
		got[r.Name] = r.DockerImage
	}
	if got["atlas-ui"] == "" {
		t.Errorf("atlas-ui missing from docker rows: %+v", rows)
	}
	if got["atlas-assets"] == "" {
		t.Errorf("atlas-assets missing from docker rows: %+v", rows)
	}
	if got["atlas-account"] == "" {
		t.Errorf("atlas-account missing from docker rows: %+v", rows)
	}
}

func TestEnrich_GoLibraries_CoverageDefaultZero(t *testing.T) {
	cfg, err := LoadConfig("testdata/transitive/services.json")
	if err != nil {
		t.Fatal(err)
	}
	rows := cfg.EnrichGoLibraries([]string{"lib-a", "lib-c"})
	got := make(map[string]int)
	for _, r := range rows {
		got[r.Name] = r.CoverageThreshold
	}
	if got["lib-a"] != 0 {
		t.Errorf("lib-a coverage_threshold=%d want 0", got["lib-a"])
	}
	if got["lib-c"] != 80 {
		t.Errorf("lib-c coverage_threshold=%d want 80", got["lib-c"])
	}
}

func TestEnrich_UnknownNameProducesWarning(t *testing.T) {
	cfg, err := LoadConfig("testdata/transitive/services.json")
	if err != nil {
		t.Fatal(err)
	}
	rows := cfg.EnrichGoServices([]string{"svc-a", "ghost"})
	var warnings []string
	cfg.Warnings(&warnings)
	if len(rows) != 1 || rows[0].Name != "svc-a" {
		t.Errorf("rows=%+v; expected only svc-a", rows)
	}
	if len(warnings) != 1 {
		t.Errorf("warnings=%v; expected 1", warnings)
	}
}
