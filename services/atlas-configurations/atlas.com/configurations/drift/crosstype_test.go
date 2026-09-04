package drift_test

import (
	"atlas-configurations/drift"
	"atlas-configurations/templates"
	"atlas-configurations/tenants"
	"atlas-configurations/tenants/worlds"
	"encoding/json"
	"testing"
)

// crossTypeDoc is a document covering every comparable section with
// content: socket, characters (both templates and presets, including the
// ap/sp fields task-289 Task 1 added to templates' preset to match
// tenants'), npcs, cashShop and mapleLife. region/majorVersion/minorVersion
// are present but excluded from comparison (drift.Excluded), as are the
// omitted worlds/diagnostics/environment/id keys.
const crossTypeDoc = `{
  "region": "GMS",
  "majorVersion": 83,
  "minorVersion": 1,
  "usesPin": true,
  "socket": {"handlers": [{"opCode": "0x01", "validator": "v", "handler": "h"}]},
  "characters": {
    "templates": [{"jobIndex": 0, "subJobIndex": 0, "gender": 0, "mapId": 40000}],
    "presets": [{"id": "11111111-1111-1111-1111-111111111111", "attributes": {"name": "n", "jobId": 100, "level": 10, "stats": {"str": 4, "dex": 4, "int": 4, "luk": 4, "hp": 50, "mp": 5}}}]
  },
  "npcs": [{"npcId": 9000, "impl": "shop"}],
  "cashShop": {"commodities": {"hourlyExpirations": [{"templateId": 1, "hours": 2}]}},
  "mapleLife": {"looks": [{"gender": 0, "faces": [20000], "hairs": [30000], "hairColors": [0], "skinColors": [0]}]}
}`

// TestIdenticalDocumentsHashIdenticallyAcrossPackages is the regression
// guard for FR-2.6: templates.RestModel and tenants.RestModel are distinct
// Go types with identical JSON tags on every comparable field, so the same
// document must canonicalize and hash identically regardless of which type
// decoded it. It fails the moment someone adds a field to one model and
// not the other.
func TestIdenticalDocumentsHashIdenticallyAcrossPackages(t *testing.T) {
	var tmpl templates.RestModel
	if err := json.Unmarshal([]byte(crossTypeDoc), &tmpl); err != nil {
		t.Fatalf("unmarshal into templates.RestModel failed: %v", err)
	}
	var tenant tenants.RestModel
	if err := json.Unmarshal([]byte(crossTypeDoc), &tenant); err != nil {
		t.Fatalf("unmarshal into tenants.RestModel failed: %v", err)
	}

	tmplDoc, err := drift.Canonicalize(tmpl)
	if err != nil {
		t.Fatalf("Canonicalize(templates.RestModel) failed: %v", err)
	}
	tenantDoc, err := drift.Canonicalize(tenant)
	if err != nil {
		t.Fatalf("Canonicalize(tenants.RestModel) failed: %v", err)
	}

	tmplAgg, err := drift.Aggregate(tmplDoc)
	if err != nil {
		t.Fatalf("Aggregate(templates doc) failed: %v", err)
	}
	tenantAgg, err := drift.Aggregate(tenantDoc)
	if err != nil {
		t.Fatalf("Aggregate(tenants doc) failed: %v", err)
	}
	if tmplAgg != tenantAgg {
		t.Errorf("aggregate differs across packages:\n templates=%s\n tenants  =%s", tmplAgg, tenantAgg)
	}

	tmplSections, err := drift.Sections(tmplDoc)
	if err != nil {
		t.Fatalf("Sections(templates doc) failed: %v", err)
	}
	tenantSections, err := drift.Sections(tenantDoc)
	if err != nil {
		t.Fatalf("Sections(tenants doc) failed: %v", err)
	}
	for _, name := range drift.All() {
		a, b := tmplSections[name], tenantSections[name]
		if a != b {
			t.Errorf("section %q differs: templates=%s tenants=%s", name, a, b)
		}
	}
}

// TestTenantOnlyDiagnosticsDoNotAffectHashes asserts that diagnostics --
// tenant-only, never carried by templates -- never participates in the
// drift hash (drift.Excluded).
func TestTenantOnlyDiagnosticsDoNotAffectHashes(t *testing.T) {
	var base tenants.RestModel
	if err := json.Unmarshal([]byte(crossTypeDoc), &base); err != nil {
		t.Fatalf("unmarshal into tenants.RestModel failed: %v", err)
	}
	baseDoc, err := drift.Canonicalize(base)
	if err != nil {
		t.Fatalf("Canonicalize(base) failed: %v", err)
	}
	baseAgg, err := drift.Aggregate(baseDoc)
	if err != nil {
		t.Fatalf("Aggregate(base) failed: %v", err)
	}

	var withDiagnostics tenants.RestModel
	if err := json.Unmarshal([]byte(crossTypeDoc), &withDiagnostics); err != nil {
		t.Fatalf("unmarshal into tenants.RestModel failed: %v", err)
	}
	withDiagnostics.Diagnostics.TracePackets = true

	diagDoc, err := drift.Canonicalize(withDiagnostics)
	if err != nil {
		t.Fatalf("Canonicalize(withDiagnostics) failed: %v", err)
	}
	diagAgg, err := drift.Aggregate(diagDoc)
	if err != nil {
		t.Fatalf("Aggregate(withDiagnostics) failed: %v", err)
	}

	if baseAgg != diagAgg {
		t.Errorf("diagnostics changed the aggregate: base=%s withDiagnostics=%s", baseAgg, diagAgg)
	}
}

// TestWorldsDoNotAffectHashes asserts that worlds -- tenant-owned (design
// D3) -- never participates in the drift hash (drift.Excluded).
func TestWorldsDoNotAffectHashes(t *testing.T) {
	var base tenants.RestModel
	if err := json.Unmarshal([]byte(crossTypeDoc), &base); err != nil {
		t.Fatalf("unmarshal into tenants.RestModel failed: %v", err)
	}
	baseDoc, err := drift.Canonicalize(base)
	if err != nil {
		t.Fatalf("Canonicalize(base) failed: %v", err)
	}
	baseAgg, err := drift.Aggregate(baseDoc)
	if err != nil {
		t.Fatalf("Aggregate(base) failed: %v", err)
	}

	var withWorlds tenants.RestModel
	if err := json.Unmarshal([]byte(crossTypeDoc), &withWorlds); err != nil {
		t.Fatalf("unmarshal into tenants.RestModel failed: %v", err)
	}
	withWorlds.Worlds = append(withWorlds.Worlds, worlds.RestModel{Name: "w"})

	worldsDoc, err := drift.Canonicalize(withWorlds)
	if err != nil {
		t.Fatalf("Canonicalize(withWorlds) failed: %v", err)
	}
	worldsAgg, err := drift.Aggregate(worldsDoc)
	if err != nil {
		t.Fatalf("Aggregate(withWorlds) failed: %v", err)
	}

	if baseAgg != worldsAgg {
		t.Errorf("worlds changed the aggregate: base=%s withWorlds=%s", baseAgg, worldsAgg)
	}
}
