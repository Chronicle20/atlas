package tenants

import (
	"atlas-configurations/templates"
	"atlas-configurations/tenants/npcs"
	"atlas-configurations/tenants/worlds"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// testTemplateEntity is a SQLite-compatible mirror of templates.Entity,
// mapped to the same table name so templates.Processor's real queries
// find the rows this test seeds. Copied from
// templates/processor_test.go:22-33.
type testTemplateEntity struct {
	Id           uuid.UUID       `gorm:"type:text;primaryKey"`
	Region       string          `gorm:"not null"`
	MajorVersion uint16          `gorm:"not null"`
	MinorVersion uint16          `gorm:"not null"`
	Data         json.RawMessage `gorm:"type:text;not null"`
	Environment  string          `gorm:"not null;default:''"`
}

func (testTemplateEntity) TableName() string { return "templates" }

// setupViewTestDB extends setupTestDB (processor_test.go:55) with the
// templates table, so one DB serves both processors.
func setupViewTestDB(t *testing.T) *gorm.DB {
	db := setupTestDB(t)
	if err := db.AutoMigrate(&testTemplateEntity{}); err != nil {
		t.Fatalf("failed to migrate templates table: %v", err)
	}
	return db
}

// seedTemplate writes a template row directly and returns its id.
func seedTemplate(t *testing.T, db *gorm.DB, region string, major, minor uint16, mutate func(*templates.RestModel)) uuid.UUID {
	t.Helper()
	rm := templates.RestModel{
		Region:       region,
		MajorVersion: major,
		MinorVersion: minor,
		UsesPin:      true,
	}
	if mutate != nil {
		mutate(&rm)
	}
	data, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("failed to marshal template: %v", err)
	}
	id := uuid.New()
	e := testTemplateEntity{
		Id:           id,
		Region:       region,
		MajorVersion: major,
		MinorVersion: minor,
		Data:         data,
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("failed to seed template: %v", err)
	}
	return id
}

// seedTenant builds a tenant via createTestRestModel, applies mutate, and
// persists it through the processor under test.
func seedTenant(t *testing.T, db *gorm.DB, p Processor, region string, major, minor uint16, mutate func(*RestModel)) uuid.UUID {
	t.Helper()
	rm := createTestRestModel(region, major, minor)
	if mutate != nil {
		mutate(&rm)
	}
	id, err := p.Create(rm)
	if err != nil {
		t.Fatalf("failed to seed tenant: %v", err)
	}
	return id
}

// tenantFromTemplate converts a template's RestModel into a tenant
// RestModel by round-tripping through JSON: both types share JSON tags for
// every comparable section, so the two documents canonicalize identically.
func tenantFromTemplate(t *testing.T, tmpl templates.RestModel) RestModel {
	t.Helper()
	b, err := json.Marshal(tmpl)
	if err != nil {
		t.Fatalf("failed to marshal template: %v", err)
	}
	var rm RestModel
	if err := json.Unmarshal(b, &rm); err != nil {
		t.Fatalf("failed to unmarshal into tenant RestModel: %v", err)
	}
	return rm
}

func assertAllSectionsFalse(t *testing.T, sd map[string]bool, except ...string) {
	t.Helper()
	exceptions := make(map[string]bool, len(except))
	for _, e := range except {
		exceptions[e] = true
	}
	for name, v := range sd {
		want := exceptions[name]
		if v != want {
			t.Errorf("SectionDrift[%q] = %v, want %v", name, v, want)
		}
	}
}

func TestView(t *testing.T) {
	t.Run("NoBaselineReportsUnknownNotDrift", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		id := seedTenant(t, db, p, "GMS", 83, 1, nil)

		v, err := p.ViewByIdProvider(id)()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.BaselineTemplateId != "" {
			t.Errorf("expected empty BaselineTemplateId, got %q", v.BaselineTemplateId)
		}
		if v.BaselineRevision != "" {
			t.Errorf("expected empty BaselineRevision, got %q", v.BaselineRevision)
		}
		if v.StoredRevision == "" {
			t.Error("expected non-empty StoredRevision")
		}
		if v.TemplateDrift {
			t.Error("expected TemplateDrift false")
		}
		assertAllSectionsFalse(t, v.SectionDrift)
	})

	t.Run("IdenticalDocumentsReportNoDrift", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		templateId := seedTemplate(t, db, "GMS", 83, 1, nil)
		tmplRow, err := templates.NewProcessor(l, ctx, db).GetById(templateId)
		if err != nil {
			t.Fatalf("failed to load seeded template: %v", err)
		}

		tenantRM := tenantFromTemplate(t, tmplRow)
		id, err := p.Create(tenantRM)
		if err != nil {
			t.Fatalf("failed to create tenant: %v", err)
		}

		v, err := p.ViewByIdProvider(id)()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.BaselineTemplateId != templateId.String() {
			t.Errorf("expected BaselineTemplateId %q, got %q", templateId.String(), v.BaselineTemplateId)
		}
		if v.BaselineRevision != v.StoredRevision {
			t.Errorf("expected BaselineRevision == StoredRevision, got %q != %q", v.BaselineRevision, v.StoredRevision)
		}
		if v.TemplateDrift {
			t.Error("expected TemplateDrift false")
		}
		assertAllSectionsFalse(t, v.SectionDrift)
	})

	t.Run("PropertiesEditFlipsOnlyProperties", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		templateId := seedTemplate(t, db, "GMS", 83, 1, nil)
		tmplRow, err := templates.NewProcessor(l, ctx, db).GetById(templateId)
		if err != nil {
			t.Fatalf("failed to load seeded template: %v", err)
		}

		tenantRM := tenantFromTemplate(t, tmplRow)
		tenantRM.UsesPin = false
		id, err := p.Create(tenantRM)
		if err != nil {
			t.Fatalf("failed to create tenant: %v", err)
		}

		v, err := p.ViewByIdProvider(id)()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !v.TemplateDrift {
			t.Error("expected TemplateDrift true")
		}
		if !v.SectionDrift["properties"] {
			t.Error("expected SectionDrift[properties] true")
		}
		assertAllSectionsFalse(t, v.SectionDrift, "properties")
	})

	t.Run("NpcsEditFlipsOnlyNpcs", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		templateId := seedTemplate(t, db, "GMS", 83, 1, nil)
		tmplRow, err := templates.NewProcessor(l, ctx, db).GetById(templateId)
		if err != nil {
			t.Fatalf("failed to load seeded template: %v", err)
		}

		tenantRM := tenantFromTemplate(t, tmplRow)
		tenantRM.NPCs = []npcs.RestModel{{NPCId: 9000, Impl: "shop"}}
		id, err := p.Create(tenantRM)
		if err != nil {
			t.Fatalf("failed to create tenant: %v", err)
		}

		v, err := p.ViewByIdProvider(id)()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !v.TemplateDrift {
			t.Error("expected TemplateDrift true")
		}
		if !v.SectionDrift["npcs"] {
			t.Error("expected SectionDrift[npcs] true")
		}
		assertAllSectionsFalse(t, v.SectionDrift, "npcs")
	})

	t.Run("WorldsEditFlipsNothing", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		templateId := seedTemplate(t, db, "GMS", 83, 1, nil)
		tmplRow, err := templates.NewProcessor(l, ctx, db).GetById(templateId)
		if err != nil {
			t.Fatalf("failed to load seeded template: %v", err)
		}

		tenantRM := tenantFromTemplate(t, tmplRow)
		tenantRM.Worlds = []worlds.RestModel{{Name: "Scania"}}
		id, err := p.Create(tenantRM)
		if err != nil {
			t.Fatalf("failed to create tenant: %v", err)
		}

		v, err := p.ViewByIdProvider(id)()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.TemplateDrift {
			t.Error("expected TemplateDrift false")
		}
		assertAllSectionsFalse(t, v.SectionDrift)
	})

	t.Run("DiagnosticsEditFlipsNothing", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		templateId := seedTemplate(t, db, "GMS", 83, 1, nil)
		tmplRow, err := templates.NewProcessor(l, ctx, db).GetById(templateId)
		if err != nil {
			t.Fatalf("failed to load seeded template: %v", err)
		}

		tenantRM := tenantFromTemplate(t, tmplRow)
		tenantRM.Diagnostics.TracePackets = true
		id, err := p.Create(tenantRM)
		if err != nil {
			t.Fatalf("failed to create tenant: %v", err)
		}

		v, err := p.ViewByIdProvider(id)()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.TemplateDrift {
			t.Error("expected TemplateDrift false")
		}
		assertAllSectionsFalse(t, v.SectionDrift)
	})

	t.Run("SectionDriftAlwaysCarriesSixKeys", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		id := seedTenant(t, db, p, "GMS", 83, 1, nil)

		v, err := p.ViewByIdProvider(id)()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(v.SectionDrift) != 6 {
			t.Errorf("expected 6 SectionDrift keys, got %d", len(v.SectionDrift))
		}
		for _, name := range []string{"properties", "socket", "characters", "npcs", "cashShop", "mapleLife"} {
			if _, ok := v.SectionDrift[name]; !ok {
				t.Errorf("expected SectionDrift to contain %q", name)
			}
		}
	})

	t.Run("UnwiredTemplatesProcessorDegrades", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db)

		seedTemplate(t, db, "GMS", 83, 1, nil)
		id := seedTenant(t, db, p, "GMS", 83, 1, nil)

		v, err := p.ViewByIdProvider(id)()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.BaselineTemplateId != "" {
			t.Errorf("expected empty BaselineTemplateId, got %q", v.BaselineTemplateId)
		}
		if v.TemplateDrift {
			t.Error("expected TemplateDrift false")
		}
		assertAllSectionsFalse(t, v.SectionDrift)
	})
}

// countingTemplates counts GetByRegionAndVersion calls so FR-3.4 is
// ENFORCED rather than described: a per-row lookup on a paged list is not
// acceptable (NFR-1).
type countingTemplates struct {
	templates.Processor
	calls int
}

func (c *countingTemplates) GetByRegionAndVersion(region string, major, minor uint16) (templates.RestModel, error) {
	c.calls++
	return c.Processor.GetByRegionAndVersion(region, major, minor)
}

func TestAllViewProviderResolvesEachBaselineOnce(t *testing.T) {
	db := setupViewTestDB(t)
	l := testLogger()
	ctx := context.Background()

	seedTemplate(t, db, "GMS", 83, 1, nil)
	seedTemplate(t, db, "GMS", 84, 1, nil)

	p := NewProcessor(l, ctx, db)
	stub := &countingTemplates{Processor: templates.NewProcessor(l, ctx, db)}
	p = p.WithTemplates(stub)

	for i := 0; i < 4; i++ {
		seedTenant(t, db, p, "GMS", 83, 1, nil)
	}
	for i := 0; i < 2; i++ {
		seedTenant(t, db, p, "GMS", 84, 1, nil)
	}

	paged, err := p.AllViewProvider(model.Page{Number: 1, Size: 250})()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paged.Items) != 6 {
		t.Errorf("expected 6 items, got %d", len(paged.Items))
	}
	if stub.calls != 2 {
		t.Errorf("GetByRegionAndVersion called %d times, want 2 (one per distinct region/version key)", stub.calls)
	}
}
