package preset

import (
	"atlas-configurations/data"
	"atlas-configurations/data/mock"
	"context"
	"errors"
	"strings"
	"testing"
)

// helpers

var (
	goodHat   = data.ItemInfo{Id: 1002357, Equipable: true}   // bucket 100
	goodGlove = data.ItemInfo{Id: 1082000, Equipable: true}   // bucket 108
	goodHat2  = data.ItemInfo{Id: 1002500, Equipable: true}   // bucket 100 — collides with goodHat
	badItem   = data.ItemInfo{Id: 1002357, Equipable: false}  // equipable=false
	goodUsage = data.ItemInfo{Id: 2000000, Equipable: false}  // use item
	goodSkill = data.SkillInfo{Id: 1121008, Name: "Hero's Will", MaxLevel: 5}
)

func makeClient() *mock.FakeClient {
	return &mock.FakeClient{
		Skills: map[uint32]data.SkillInfo{
			goodSkill.Id: goodSkill,
		},
		Items: map[uint32]data.ItemInfo{
			goodHat.Id:   goodHat,
			goodGlove.Id: goodGlove,
			goodHat2.Id:  goodHat2,
			goodUsage.Id: goodUsage,
		},
	}
}

func validPreset() RestModel {
	return RestModel{
		Id: "00000000-0000-0000-0000-000000000001",
		Attributes: Attributes{
			Name:      "Hero",
			Level:     200,
			JobId:     112, // HeroId
			Gender:    0,
			Equipment: []EquipmentEntry{{TemplateId: goodHat.Id, UseAverageStats: true}},
			Inventory: []InventoryEntry{{TemplateId: goodUsage.Id, Quantity: 10}},
			Skills:    []SkillEntry{{SkillId: goodSkill.Id, Level: 5}},
		},
	}
}

func hasError(errs []ValidationError, field, substr string) bool {
	for _, e := range errs {
		if e.Field == field && strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}

// ── happy path ────────────────────────────────────────────────────────────────

func TestValidator_AllGood(t *testing.T) {
	v := NewValidator(makeClient())
	_, errs := v.Validate(context.Background(), []RestModel{validPreset()})
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %+v", errs)
	}
}

// ── R-1: UUID auto-assignment ─────────────────────────────────────────────────

func TestValidator_AssignsMissingId(t *testing.T) {
	v := NewValidator(makeClient())
	p := validPreset()
	p.Id = ""
	out, _ := v.Validate(context.Background(), []RestModel{p})
	if out[0].Id == "" {
		t.Fatal("expected UUID to be assigned to empty Id")
	}
}

func TestValidator_ErrorCarriesAssignedId(t *testing.T) {
	v := NewValidator(makeClient())
	p := validPreset()
	p.Id = ""
	p.Attributes.Name = "" // force a validation error
	out, errs := v.Validate(context.Background(), []RestModel{p})
	if out[0].Id == "" {
		t.Fatal("expected UUID assigned")
	}
	if len(errs) == 0 {
		t.Fatal("expected at least one error")
	}
	if errs[0].PresetId != out[0].Id {
		t.Fatalf("error presetId %q != assigned id %q", errs[0].PresetId, out[0].Id)
	}
}

// ── table-driven failure tests ────────────────────────────────────────────────

type ruleCase struct {
	name    string
	mutate  func(p *RestModel)
	field   string
	msgPart string
}

func TestValidator_Rules(t *testing.T) {
	cases := []ruleCase{
		// R-1 name
		{
			name:    "name empty",
			mutate:  func(p *RestModel) { p.Attributes.Name = "" },
			field:   "name",
			msgPart: "1..64",
		},
		{
			name:    "name too long",
			mutate:  func(p *RestModel) { p.Attributes.Name = strings.Repeat("x", 65) },
			field:   "name",
			msgPart: "1..64",
		},
		// R-2 description
		{
			name:    "description too long",
			mutate:  func(p *RestModel) { p.Attributes.Description = strings.Repeat("x", 513) },
			field:   "description",
			msgPart: "512",
		},
		// R-3 jobId
		{
			name:    "unknown jobId",
			mutate:  func(p *RestModel) { p.Attributes.JobId = 99999 },
			field:   "jobId",
			msgPart: "unknown job id",
		},
		// R-4 gender
		{
			name:    "gender out of range",
			mutate:  func(p *RestModel) { p.Attributes.Gender = 5 },
			field:   "gender",
			msgPart: "0 or 1",
		},
		// R-5 level
		{
			name:    "level zero",
			mutate:  func(p *RestModel) { p.Attributes.Level = 0 },
			field:   "level",
			msgPart: "[1,250]",
		},
		{
			name:    "level 251",
			mutate:  func(p *RestModel) { p.Attributes.Level = 251 },
			field:   "level",
			msgPart: "[1,250]",
		},
		// R-6 equipment not found
		{
			name: "equipment templateId not found",
			mutate: func(p *RestModel) {
				p.Attributes.Equipment = []EquipmentEntry{{TemplateId: 9999999}}
			},
			field:   "equipment[0].templateId",
			msgPart: "not found",
		},
		// R-6 equipment not equippable
		{
			name: "equipment not equippable",
			mutate: func(p *RestModel) {
				p.Attributes.Equipment = []EquipmentEntry{{TemplateId: goodUsage.Id}}
			},
			field:   "equipment[0].templateId",
			msgPart: "not equippable",
		},
		// R-7 slot collision
		{
			name: "equipment slot collision",
			mutate: func(p *RestModel) {
				// goodHat (1002357, bucket 100) and goodHat2 (1002500, bucket 100) collide
				p.Attributes.Equipment = []EquipmentEntry{
					{TemplateId: goodHat.Id},
					{TemplateId: goodHat2.Id},
				}
			},
			field:   "equipment[1].templateId",
			msgPart: "slot collision",
		},
		// R-8 inventory not found
		{
			name: "inventory templateId not found",
			mutate: func(p *RestModel) {
				p.Attributes.Inventory = []InventoryEntry{{TemplateId: 9999999, Quantity: 1}}
			},
			field:   "inventory[0].templateId",
			msgPart: "not found",
		},
		// R-9 inventory quantity zero
		{
			name: "inventory quantity zero",
			mutate: func(p *RestModel) {
				p.Attributes.Inventory = []InventoryEntry{{TemplateId: goodUsage.Id, Quantity: 0}}
			},
			field:   "inventory[0].quantity",
			msgPart: "≥1",
		},
		// R-10 skill not found
		{
			name: "skill id not found",
			mutate: func(p *RestModel) {
				p.Attributes.Skills = []SkillEntry{{SkillId: 9999999, Level: 1}}
			},
			field:   "skills[0].skillId",
			msgPart: "not found",
		},
		// R-11 skill level > maxLevel
		{
			name: "skill level exceeds maxLevel",
			mutate: func(p *RestModel) {
				p.Attributes.Skills = []SkillEntry{{SkillId: goodSkill.Id, Level: goodSkill.MaxLevel + 1}}
			},
			field:   "skills[0].level",
			msgPart: "[1,maxLevel]",
		},
		// R-11 skill level zero
		{
			name: "skill level zero",
			mutate: func(p *RestModel) {
				p.Attributes.Skills = []SkillEntry{{SkillId: goodSkill.Id, Level: 0}}
			},
			field:   "skills[0].level",
			msgPart: "[1,maxLevel]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := NewValidator(makeClient())
			p := validPreset()
			tc.mutate(&p)
			_, errs := v.Validate(context.Background(), []RestModel{p})
			if !hasError(errs, tc.field, tc.msgPart) {
				t.Fatalf("expected error on field %q containing %q, got: %+v", tc.field, tc.msgPart, errs)
			}
		})
	}
}

// R-12: skills batch lookup failure → single error on "skills" field.
func TestValidator_SkillsBatchError(t *testing.T) {
	fake := makeClient()
	fake.SkillsErr = errors.New("atlas-data unreachable")
	v := NewValidator(fake)
	p := validPreset()
	_, errs := v.Validate(context.Background(), []RestModel{p})
	if !hasError(errs, "skills", "atlas-data lookup failed") {
		t.Fatalf("expected skills batch error, got: %+v", errs)
	}
}

// description exactly 512 chars should be fine.
func TestValidator_DescriptionExactly512(t *testing.T) {
	v := NewValidator(makeClient())
	p := validPreset()
	p.Attributes.Description = strings.Repeat("x", 512)
	_, errs := v.Validate(context.Background(), []RestModel{p})
	if hasError(errs, "description", "") {
		t.Fatalf("512-char description should not produce error, got: %+v", errs)
	}
}

// level 250 is valid.
func TestValidator_Level250Valid(t *testing.T) {
	v := NewValidator(makeClient())
	p := validPreset()
	p.Attributes.Level = 250
	_, errs := v.Validate(context.Background(), []RestModel{p})
	if hasError(errs, "level", "") {
		t.Fatalf("level 250 should be valid, got: %+v", errs)
	}
}

// Two different slot buckets should not collide.
func TestValidator_EquipmentDifferentSlots(t *testing.T) {
	v := NewValidator(makeClient())
	p := validPreset()
	// goodHat bucket 100, goodGlove bucket 108 — no collision
	p.Attributes.Equipment = []EquipmentEntry{
		{TemplateId: goodHat.Id},
		{TemplateId: goodGlove.Id},
	}
	_, errs := v.Validate(context.Background(), []RestModel{p})
	if hasError(errs, "equipment[1].templateId", "collision") {
		t.Fatalf("different slot buckets should not collide, got: %+v", errs)
	}
}
