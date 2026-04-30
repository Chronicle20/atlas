package factory

import (
	"atlas-character-factory/character"
	cmock "atlas-character-factory/character/mock"
	"atlas-character-factory/configuration"
	confmock "atlas-character-factory/configuration/mock"
	"atlas-character-factory/configuration/tenant/characters/preset"
	"atlas-character-factory/data"
	dmock "atlas-character-factory/data/mock"
	"context"
	"errors"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// mkCtx creates a context with a valid tenant.
func mkCtx(t *testing.T) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithContext(context.Background(), tn)
}

// minimalPreset returns a minimal valid preset suitable for happy-path tests.
func minimalPreset(id uuid.UUID) preset.RestModel {
	return preset.RestModel{
		Id: id.String(),
		Attributes: preset.Attributes{
			JobId: 112, Level: 200,
			Stats: preset.StatBlock{Str: 999, Hp: 30000, Mp: 6000},
			Equipment: []preset.EquipmentEntry{
				{TemplateId: 1002357, UseAverageStats: true},
				{TemplateId: 1402046, UseAverageStats: true},
			},
			Inventory: []preset.InventoryEntry{
				{TemplateId: 2000005, Quantity: 200},
			},
			Skills: []preset.SkillEntry{
				{SkillId: 1121008, Level: 5},
			},
		},
	}
}

func TestCreateFromPreset_InvalidPresetId(t *testing.T) {
	p := NewProcessorWithClients(logrus.StandardLogger(), &confmock.FakePresetClient{}, &cmock.FakeNameValidityClient{}, &dmock.FakeClient{})
	_, err := p.CreateFromPreset(mkCtx(t), PresetCreateRestModel{PresetId: "not-a-uuid"})
	if !errors.Is(err, ErrInvalidPresetId) {
		t.Fatalf("expected ErrInvalidPresetId, got %v", err)
	}
}

func TestCreateFromPreset_PresetNotFound(t *testing.T) {
	pc := &confmock.FakePresetClient{Err: configuration.ErrPresetNotFound}
	p := NewProcessorWithClients(logrus.StandardLogger(), pc, &cmock.FakeNameValidityClient{}, &dmock.FakeClient{})
	_, err := p.CreateFromPreset(mkCtx(t), PresetCreateRestModel{PresetId: uuid.New().String()})
	if !errors.Is(err, ErrPresetNotFound) {
		t.Fatalf("expected ErrPresetNotFound, got %v", err)
	}
}

func TestCreateFromPreset_NameInvalidLength(t *testing.T) {
	presetId := uuid.New()
	pc := &confmock.FakePresetClient{
		Presets: map[uuid.UUID]preset.RestModel{presetId: minimalPreset(presetId)},
	}
	nc := &cmock.FakeNameValidityClient{
		Result: character.NameValidityResult{Valid: false, Reason: "length"},
	}
	p := NewProcessorWithClients(logrus.StandardLogger(), pc, nc, &dmock.FakeClient{})
	_, err := p.CreateFromPreset(mkCtx(t), PresetCreateRestModel{PresetId: presetId.String(), Name: "x"})
	var nameErr *NameInvalidError
	if !errors.As(err, &nameErr) {
		t.Fatalf("expected *NameInvalidError, got %T: %v", err, err)
	}
	if nameErr.Reason != "length" {
		t.Fatalf("expected Reason 'length', got '%s'", nameErr.Reason)
	}
}

func TestCreateFromPreset_NameDuplicate(t *testing.T) {
	presetId := uuid.New()
	pc := &confmock.FakePresetClient{
		Presets: map[uuid.UUID]preset.RestModel{presetId: minimalPreset(presetId)},
	}
	nc := &cmock.FakeNameValidityClient{
		Result: character.NameValidityResult{Valid: false, Reason: "duplicate"},
	}
	p := NewProcessorWithClients(logrus.StandardLogger(), pc, nc, &dmock.FakeClient{})
	_, err := p.CreateFromPreset(mkCtx(t), PresetCreateRestModel{PresetId: presetId.String(), Name: "Dupe"})
	if !errors.Is(err, ErrNameDuplicate) {
		t.Fatalf("expected ErrNameDuplicate, got %v", err)
	}
}

func TestCreateFromPreset_EquipmentValidationFail(t *testing.T) {
	presetId := uuid.New()
	pc := &confmock.FakePresetClient{
		Presets: map[uuid.UUID]preset.RestModel{presetId: minimalPreset(presetId)},
	}
	nc := &cmock.FakeNameValidityClient{
		Result: character.NameValidityResult{Valid: true},
	}
	// FakeClient with no items — GetItemById returns ErrNotFound for all ids
	dc := &dmock.FakeClient{}
	p := NewProcessorWithClients(logrus.StandardLogger(), pc, nc, dc)
	_, err := p.CreateFromPreset(mkCtx(t), PresetCreateRestModel{PresetId: presetId.String(), Name: "Hero"})
	if !errors.Is(err, ErrPresetValidation) {
		t.Fatalf("expected ErrPresetValidation, got %v", err)
	}
}

func TestCreateFromPreset_SkillBatchFail(t *testing.T) {
	presetId := uuid.New()
	pc := &confmock.FakePresetClient{
		Presets: map[uuid.UUID]preset.RestModel{presetId: minimalPreset(presetId)},
	}
	nc := &cmock.FakeNameValidityClient{Result: character.NameValidityResult{Valid: true}}
	dc := &dmock.FakeClient{
		Items: map[uint32]data.ItemInfo{
			// All equipment and inventory items resolve, but skills fail
			1002357: {Id: 1002357, Equipable: true},
			1402046: {Id: 1402046, Equipable: true},
			2000005: {Id: 2000005, Equipable: false},
		},
		SkillsErr: errors.New("atlas-data connection refused"),
	}
	p := NewProcessorWithClients(logrus.StandardLogger(), pc, nc, dc)
	_, err := p.CreateFromPreset(mkCtx(t), PresetCreateRestModel{PresetId: presetId.String(), Name: "Hero"})
	if !errors.Is(err, ErrAtlasDataUnreachable) {
		t.Fatalf("expected ErrAtlasDataUnreachable, got %v", err)
	}
}

func TestCreateFromPreset_SkillNotFoundInResponse(t *testing.T) {
	presetId := uuid.New()
	pc := &confmock.FakePresetClient{
		Presets: map[uuid.UUID]preset.RestModel{presetId: minimalPreset(presetId)},
	}
	nc := &cmock.FakeNameValidityClient{Result: character.NameValidityResult{Valid: true}}
	dc := &dmock.FakeClient{
		Items: map[uint32]data.ItemInfo{
			1002357: {Id: 1002357, Equipable: true},
			1402046: {Id: 1402046, Equipable: true},
			2000005: {Id: 2000005, Equipable: false},
		},
		// Skills map is empty — GetSkillsByIds returns empty slice (no error)
		Skills: map[uint32]data.SkillInfo{},
	}
	p := NewProcessorWithClients(logrus.StandardLogger(), pc, nc, dc)
	_, err := p.CreateFromPreset(mkCtx(t), PresetCreateRestModel{PresetId: presetId.String(), Name: "Hero"})
	if !errors.Is(err, ErrPresetValidation) {
		t.Fatalf("expected ErrPresetValidation (skill not found), got %v", err)
	}
}

// TestBuildPresetCharacterCreationSaga_StepShape tests the pure saga-building helper.
func TestBuildPresetCharacterCreationSaga_StepShape(t *testing.T) {
	pr := preset.RestModel{
		Id: "test",
		Attributes: preset.Attributes{
			JobId: 112, Level: 200,
			Stats: preset.StatBlock{Str: 999, Hp: 30000, Mp: 6000},
			Equipment: []preset.EquipmentEntry{
				{TemplateId: 1002357, UseAverageStats: true},
				{TemplateId: 1402046, UseAverageStats: true},
			},
			Inventory: []preset.InventoryEntry{
				{TemplateId: 2000005, Quantity: 200},
			},
			Skills: []preset.SkillEntry{
				{SkillId: 1121008, Level: 5},
			},
		},
	}
	skillsById := map[uint32]data.SkillInfo{
		1121008: {Id: 1121008, MaxLevel: 5},
	}
	transactionId := uuid.New()
	sg := buildPresetCharacterCreationSaga(
		transactionId,
		PresetCreateRestModel{AccountId: 1, WorldId: 0, Name: "AdminHero"},
		pr,
		skillsById,
	)

	// 1 create_character + 1 award_asset_0 + 2 create_and_equip_asset + 1 create_skill_0 = 5
	const expectedSteps = 5
	if len(sg.Steps) != expectedSteps {
		t.Fatalf("expected %d steps, got %d", expectedSteps, len(sg.Steps))
	}

	if sg.TransactionId != transactionId {
		t.Errorf("TransactionId mismatch: expected %s, got %s", transactionId, sg.TransactionId)
	}

	// Verify step ordering
	expectedIds := []string{
		"create_character",
		"award_asset_0",
		"create_and_equip_asset_0",
		"create_and_equip_asset_1",
		"create_skill_0",
	}
	for i, stepId := range expectedIds {
		if sg.Steps[i].StepId != stepId {
			t.Errorf("step[%d]: expected StepId %q, got %q", i, stepId, sg.Steps[i].StepId)
		}
	}
}
