package tenants

import (
	"atlas-configurations/data"
	"atlas-configurations/data/mock"
	"atlas-configurations/drift"
	"atlas-configurations/scope"
	"atlas-configurations/templates"
	tmplpreset "atlas-configurations/templates/characters/preset"
	tmplhandler "atlas-configurations/templates/socket/handler"
	tmplwriter "atlas-configurations/templates/socket/writer"
	"atlas-configurations/tenants/characters"
	"atlas-configurations/tenants/characters/preset"
	"atlas-configurations/tenants/diagnostics"
	"atlas-configurations/tenants/npcs"
	"atlas-configurations/tenants/worlds"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// outboxTenantEnvelopeUsesPin decodes the same wire shape as
// outboxTenantEnvelope (processor_test.go), plus config.usesPin, so a reset
// test can assert the enqueued outbox payload carries the POST-reset
// UsesPin, not just environment/diagnostics.
type outboxTenantEnvelopeUsesPin struct {
	Config struct {
		UsesPin bool `json:"usesPin"`
	} `json:"config"`
}

// latestOutboxTenantEnvelopeUsesPin mirrors latestOutboxTenantEnvelope
// (processor_test.go:711-724) but decodes into
// outboxTenantEnvelopeUsesPin.
func latestOutboxTenantEnvelopeUsesPin(t *testing.T, db *gorm.DB, topic string) outboxTenantEnvelopeUsesPin {
	t.Helper()
	var row outboxlib.Entity
	if err := db.Where("topic = ?", topic).Order("id DESC").First(&row).Error; err != nil {
		t.Fatalf("failed to load outbox row for topic %q: %v", topic, err)
	}
	var env outboxTenantEnvelopeUsesPin
	if err := json.Unmarshal(row.MessageValue, &env); err != nil {
		t.Fatalf("failed to decode outbox message value: %v", err)
	}
	return env
}

// storedEntity loads the raw persisted row so a test can assert on the
// exact bytes atlas-configurations wrote, not on a decoded copy.
func storedEntity(t *testing.T, db *gorm.DB, id uuid.UUID) testEntity {
	t.Helper()
	var e testEntity
	if err := db.Where("id = ?", id).First(&e).Error; err != nil {
		t.Fatalf("failed to load stored entity %s: %v", id, err)
	}
	return e
}

func TestResetById(t *testing.T) {
	t.Run("WholeDocumentClearsAllDrift", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		templateId := seedTemplate(t, db, "GMS", 83, 1, func(rm *templates.RestModel) {
			rm.UsesPin = true
		})
		tmplRow, err := templates.NewProcessor(l, ctx, db).GetById(templateId)
		if err != nil {
			t.Fatalf("failed to load seeded template: %v", err)
		}

		id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
			rm.UsesPin = false
			rm.NPCs = []npcs.RestModel{{NPCId: 9000, Impl: "shop"}}
		})

		v, err := p.ResetById(id, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.TemplateDrift {
			t.Error("expected TemplateDrift false")
		}
		assertAllSectionsFalse(t, v.SectionDrift)

		stored, err := p.GetById(id)
		if err != nil {
			t.Fatalf("failed to reload tenant: %v", err)
		}
		if !stored.UsesPin {
			t.Error("expected stored UsesPin true")
		}
		wantNPCs, err := json.Marshal(tmplRow.NPCs)
		if err != nil {
			t.Fatalf("failed to marshal template NPCs: %v", err)
		}
		gotNPCs, err := json.Marshal(stored.NPCs)
		if err != nil {
			t.Fatalf("failed to marshal stored NPCs: %v", err)
		}
		if !bytes.Equal(wantNPCs, gotNPCs) {
			t.Errorf("stored NPCs = %s, want %s", gotNPCs, wantNPCs)
		}
	})

	t.Run("ScopedResetLeavesOtherSectionsDrifted", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		seedTemplate(t, db, "GMS", 83, 1, func(rm *templates.RestModel) {
			rm.UsesPin = true
		})

		id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
			rm.UsesPin = false
			rm.NPCs = []npcs.RestModel{{NPCId: 9000, Impl: "shop"}}
		})

		v, err := p.ResetById(id, []string{"npcs"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.SectionDrift["npcs"] {
			t.Error("expected SectionDrift[npcs] false")
		}
		if !v.SectionDrift["properties"] {
			t.Error("expected SectionDrift[properties] true")
		}
		if !v.TemplateDrift {
			t.Error("expected TemplateDrift true")
		}

		stored, err := p.GetById(id)
		if err != nil {
			t.Fatalf("failed to reload tenant: %v", err)
		}
		if stored.UsesPin {
			t.Error("expected stored UsesPin to still be false")
		}
	})

	t.Run("PreservesTheFR44Set", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		seedTemplate(t, db, "GMS", 83, 1, func(rm *templates.RestModel) {
			rm.UsesPin = true
		})

		id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
			rm.UsesPin = false
			rm.Worlds = []worlds.RestModel{{Name: "w0", ServerMessage: "keep"}}
			rm.Diagnostics = diagnostics.RestModel{TracePackets: true}
		})

		before, err := p.GetById(id)
		if err != nil {
			t.Fatalf("failed to load tenant before reset: %v", err)
		}

		if _, err := p.ResetById(id, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		after, err := p.GetById(id)
		if err != nil {
			t.Fatalf("failed to reload tenant: %v", err)
		}

		assertJSONFieldEqual(t, "Id", before.Id, after.Id)
		assertJSONFieldEqual(t, "Region", before.Region, after.Region)
		assertJSONFieldEqual(t, "MajorVersion", before.MajorVersion, after.MajorVersion)
		assertJSONFieldEqual(t, "MinorVersion", before.MinorVersion, after.MinorVersion)
		assertJSONFieldEqual(t, "Environment", before.Environment, after.Environment)
		assertJSONFieldEqual(t, "Worlds", before.Worlds, after.Worlds)
		assertJSONFieldEqual(t, "Diagnostics", before.Diagnostics, after.Diagnostics)

		if !after.UsesPin {
			t.Error("expected stored UsesPin to come from the template (true)")
		}
	})

	t.Run("Idempotent", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		seedTemplate(t, db, "GMS", 83, 1, nil)
		id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
			rm.UsesPin = false
		})

		if _, err := p.ResetById(id, nil); err != nil {
			t.Fatalf("first reset: unexpected error: %v", err)
		}
		afterFirst := storedEntity(t, db, id)

		if _, err := p.ResetById(id, nil); err != nil {
			t.Fatalf("second reset: unexpected error: %v", err)
		}
		afterSecond := storedEntity(t, db, id)

		if !bytes.Equal(afterFirst.Data, afterSecond.Data) {
			t.Errorf("Entity.Data changed between resets:\nfirst:  %s\nsecond: %s", afterFirst.Data, afterSecond.Data)
		}
	})

	t.Run("WritesHistoryBeforeTheChange", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		seedTemplate(t, db, "GMS", 83, 1, nil)
		id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
			rm.UsesPin = false
		})

		if _, err := p.ResetById(id, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var historyCount int64
		if err := db.Model(&HistoryEntity{}).Where("tenant_id = ?", id).Count(&historyCount).Error; err != nil {
			t.Fatalf("failed to count history: %v", err)
		}
		if historyCount != 1 {
			t.Fatalf("expected 1 history record, got %d", historyCount)
		}

		var h testHistoryEntity
		if err := db.Where("tenant_id = ?", id).First(&h).Error; err != nil {
			t.Fatalf("failed to load history row: %v", err)
		}
		var historyRM RestModel
		if err := json.Unmarshal(h.Data, &historyRM); err != nil {
			t.Fatalf("failed to unmarshal history data: %v", err)
		}
		if historyRM.UsesPin {
			t.Error("expected history row to carry the PRE-reset UsesPin (false)")
		}
	})

	t.Run("EnqueuesStatusOutbox", func(t *testing.T) {
		t.Setenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS", "tenant-status")
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		seedTemplate(t, db, "GMS", 83, 1, func(rm *templates.RestModel) {
			rm.UsesPin = true
		})
		id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
			rm.UsesPin = false
		})

		if _, err := p.ResetById(id, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		envelope := latestOutboxTenantEnvelopeUsesPin(t, db, "tenant-status")
		if !envelope.Config.UsesPin {
			t.Error("expected outbox tenant config to carry the post-reset UsesPin (true, from the template)")
		}
	})

	t.Run("UnknownSectionRejected", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		seedTemplate(t, db, "GMS", 83, 1, nil)
		id := seedTenant(t, db, p, "GMS", 83, 1, nil)
		before := storedEntity(t, db, id)

		_, err := p.ResetById(id, []string{"worlds"})
		if !errors.Is(err, drift.ErrUnknownSection) {
			t.Fatalf("ResetById err = %v, want errors.Is drift.ErrUnknownSection", err)
		}

		after := storedEntity(t, db, id)
		if !bytes.Equal(before.Data, after.Data) {
			t.Error("expected stored row unchanged after a rejected reset")
		}
	})

	t.Run("UnknownSectionRejectedBeforeIO", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		_, err := p.ResetById(uuid.New(), []string{"nonsense"})
		if !errors.Is(err, drift.ErrUnknownSection) {
			t.Fatalf("ResetById err = %v, want errors.Is drift.ErrUnknownSection", err)
		}
		if errors.Is(err, ErrTenantNotFound) {
			t.Error("a section-name typo must be rejected before the tenant lookup, not surface as ErrTenantNotFound")
		}
	})

	t.Run("MissingTenantIs404", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		_, err := p.ResetById(uuid.New(), nil)
		if !errors.Is(err, ErrTenantNotFound) {
			t.Fatalf("ResetById err = %v, want errors.Is ErrTenantNotFound", err)
		}
	})

	t.Run("NoBaselineIs409", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		ctx := context.Background()
		p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))

		id := seedTenant(t, db, p, "GMS", 99, 9, nil)

		_, err := p.ResetById(id, nil)
		if !errors.Is(err, ErrNoBaselineTemplate) {
			t.Fatalf("ResetById err = %v, want errors.Is ErrNoBaselineTemplate", err)
		}
	})

	// CrossEnvironmentReadIs404 pins the design's explicit confidentiality
	// choice (design.md §3.5, step 2): ResetById reads the entity through
	// byIdEntityProvider, which is environment-SCOPED. A caller whose
	// environment does not match the row's therefore cannot find the row at
	// all -- the read 404s before update()'s AuthorizeWrite (the only
	// producer of scope.ErrCrossEnvironmentWrite in this call graph) is ever
	// reached. A caller who cannot read the row learns nothing about it, not
	// even that it exists in another environment.
	t.Run("CrossEnvironmentReadIs404", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		mainCtx := env.WithContext(context.Background(), env.Id("main"))
		mainP := NewProcessor(l, mainCtx, db).WithTemplates(templates.NewProcessor(l, mainCtx, db))

		seedTemplate(t, db, "GMS", 83, 1, nil)
		id := seedTenant(t, db, mainP, "GMS", 83, 1, nil)

		otherCtx := env.WithContext(context.Background(), env.Id("other"))
		otherP := NewProcessor(l, otherCtx, db).WithTemplates(templates.NewProcessor(l, otherCtx, db))

		_, err := otherP.ResetById(id, nil)
		if !errors.Is(err, ErrTenantNotFound) {
			t.Fatalf("ResetById err = %v, want errors.Is ErrTenantNotFound", err)
		}
		if errors.Is(err, scope.ErrCrossEnvironmentWrite) {
			t.Error("a cross-environment read must 404, not surface scope.ErrCrossEnvironmentWrite")
		}
	})
}

// TestResetById_PersistsBaselinePresetIdsVerbatim pins the §3.6 trap: the
// preset validator's mutation (assigning a uuid to an empty preset Id) must
// be discarded, so a baseline preset with an empty id is persisted with an
// empty id.
func TestResetById_PersistsBaselinePresetIdsVerbatim(t *testing.T) {
	db := setupViewTestDB(t)
	l := testLogger()
	ctx := context.Background()

	seedTemplate(t, db, "GMS", 83, 1, func(rm *templates.RestModel) {
		rm.Characters.Presets = []tmplpreset.RestModel{{
			Id: "",
			Attributes: tmplpreset.Attributes{
				Name:  "Hero",
				Level: 10,
				JobId: 100,
			},
		}}
	})

	fake := &mock.ProcessorMock{Items: map[uint32]data.ItemInfo{}, Skills: map[uint32]data.SkillInfo{}}
	p := NewProcessor(l, ctx, db).
		WithTemplates(templates.NewProcessor(l, ctx, db)).
		WithValidator(preset.NewValidator(fake))

	id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
		rm.Characters = characters.RestModel{
			Presets: []preset.RestModel{{
				Id: "some-id",
				Attributes: preset.Attributes{
					Name:  "Drifted",
					Level: 5,
					JobId: 100,
				},
			}},
		}
	})

	v, err := p.ResetById(id, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.SectionDrift["characters"] {
		t.Error("expected SectionDrift[characters] false")
	}

	stored, err := p.GetById(id)
	if err != nil {
		t.Fatalf("failed to reload tenant: %v", err)
	}
	if len(stored.Characters.Presets) != 1 {
		t.Fatalf("expected 1 stored preset, got %d", len(stored.Characters.Presets))
	}
	if stored.Characters.Presets[0].Id != "" {
		t.Errorf("expected stored preset Id to remain empty (validator mutation discarded), got %q", stored.Characters.Presets[0].Id)
	}
}

// TestResetById_ValidationFailureIsNotPersisted pins FR-4.9: if the merged
// document violates socketValidate, ResetById returns a
// *validationFailureError and never writes.
func TestResetById_ValidationFailureIsNotPersisted(t *testing.T) {
	db := setupViewTestDB(t)
	l := testLogger()
	ctx := context.Background()

	seedTemplate(t, db, "GMS", 83, 1, func(rm *templates.RestModel) {
		rm.Socket.Handlers = []tmplhandler.RestModel{
			{OpCode: "0x01", Validator: "NoOpValidator", Handler: "LoginHandle", Services: []string{"login"}},
		}
		rm.Socket.Writers = []tmplwriter.RestModel{
			{OpCode: "0x00", Writer: "AuthSuccess", Services: []string{"login"}},
		}
		// Conflicting unsupported state: LoginHandle is both a live
		// handler and marked unsupported (socket_validation_test.go's
		// TestUpdateById_RejectsConflictingUnsupportedState fixture).
		rm.Socket.Unsupported.Handlers = []string{"LoginHandle"}
	})

	p := NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))
	id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
		rm.UsesPin = false
	})
	before := storedEntity(t, db, id)

	_, err := p.ResetById(id, nil)
	var ve *validationFailureError
	if !errors.As(err, &ve) {
		t.Fatalf("ResetById err = %v, want *validationFailureError", err)
	}

	after := storedEntity(t, db, id)
	if !bytes.Equal(before.Data, after.Data) {
		t.Error("expected stored row unchanged after a validation failure")
	}
}

func assertJSONFieldEqual(t *testing.T, name string, before, after any) {
	t.Helper()
	b, err := json.Marshal(before)
	if err != nil {
		t.Fatalf("failed to marshal before.%s: %v", name, err)
	}
	a, err := json.Marshal(after)
	if err != nil {
		t.Fatalf("failed to marshal after.%s: %v", name, err)
	}
	if !bytes.Equal(b, a) {
		t.Errorf("%s changed across reset:\nbefore: %s\nafter:  %s", name, b, a)
	}
}
