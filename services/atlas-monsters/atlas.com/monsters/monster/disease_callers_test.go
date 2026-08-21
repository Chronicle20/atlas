package monster

import (
	"atlas-monsters/monster/information"
	"atlas-monsters/monster/mobskill"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
)

// TestExecuteDispel_TargetsOnlyInBoxCharacters verifies that executeDispel
// inherits box scoping purely by sharing getDiseaseTargets: of the three
// characters on the field, only the two inside the mob's bounding box are
// dispelled.
func TestExecuteDispel_TargetsOnlyInBoxCharacters(t *testing.T) {
	p, events := newRecordingProcessor(t, newTestTenant(t))
	p.inFieldFn = func(_ field.Model) ([]uint32, error) {
		return []uint32{1, 2, 3}, nil
	}
	positions := map[uint32][2]int16{
		1: {110, 205},
		2: {400, 205},
		3: {112, 205},
	}
	p.positionFn = func(id uint32) (int16, int16, error) {
		pos := positions[id]
		return pos[0], pos[1], nil
	}

	m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()
	sd := mobskill.NewModelBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(2).Build()

	p.executeDispel(m, sd, byte(monster2.SkillTypeDispel))

	if len(*events) != 2 {
		t.Fatalf("expected 2 events; got %d (%v)", len(*events), *events)
	}
	for _, e := range *events {
		if e.Topic != EnvCommandTopicCharacterBuff {
			t.Fatalf("expected topic %s; got %s", EnvCommandTopicCharacterBuff, e.Topic)
		}
	}
}

// TestExecuteDispel_NoCapForNonSeduce verifies FR-3.1: the SEDUCE-only cap
// does not limit dispel, even when the box holds more characters than
// SetCount specifies.
func TestExecuteDispel_NoCapForNonSeduce(t *testing.T) {
	p, events := newRecordingProcessor(t, newTestTenant(t))
	p.inFieldFn = func(_ field.Model) ([]uint32, error) {
		return []uint32{1, 2, 3, 4}, nil
	}
	positions := map[uint32][2]int16{
		1: {110, 205},
		2: {111, 205},
		3: {112, 205},
		4: {113, 205},
	}
	p.positionFn = func(id uint32) (int16, int16, error) {
		pos := positions[id]
		return pos[0], pos[1], nil
	}

	m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()
	sd := mobskill.NewModelBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(2).Build()

	p.executeDispel(m, sd, byte(monster2.SkillTypeDispel))

	if len(*events) != 4 {
		t.Fatalf("expected 4 events; got %d (%v)", len(*events), *events)
	}
	for _, e := range *events {
		if e.Topic != EnvCommandTopicCharacterBuff {
			t.Fatalf("expected topic %s; got %s", EnvCommandTopicCharacterBuff, e.Topic)
		}
	}
}

// TestExecuteBanish_TargetsOnlyInBoxCharacters verifies that executeBanish
// inherits box scoping purely by sharing getDiseaseTargets.
func TestExecuteBanish_TargetsOnlyInBoxCharacters(t *testing.T) {
	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBanish(information.Banish{MapId: 104000000}).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	p, events := newRecordingProcessor(t, newTestTenant(t))
	p.inFieldFn = func(_ field.Model) ([]uint32, error) {
		return []uint32{1, 2, 3}, nil
	}
	positions := map[uint32][2]int16{
		1: {110, 205},
		2: {400, 205},
		3: {112, 205},
	}
	p.positionFn = func(id uint32) (int16, int16, error) {
		pos := positions[id]
		return pos[0], pos[1], nil
	}

	m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()
	sd := mobskill.NewModelBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(2).Build()

	p.executeBanish(m, sd, byte(monster2.SkillTypeBanish))

	if len(*events) != 2 {
		t.Fatalf("expected 2 events; got %d (%v)", len(*events), *events)
	}
	for _, e := range *events {
		if e.Topic != EnvCommandTopicPortal {
			t.Fatalf("expected topic %s; got %s", EnvCommandTopicPortal, e.Topic)
		}
	}
}

// TestExecuteBanish_NoBanishMapEmitsNothing verifies the early-return
// behavior when the monster has no banish map configured is unchanged.
func TestExecuteBanish_NoBanishMapEmitsNothing(t *testing.T) {
	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	p, events := newRecordingProcessor(t, newTestTenant(t))
	p.inFieldFn = func(_ field.Model) ([]uint32, error) {
		return []uint32{1, 2, 3}, nil
	}
	positions := map[uint32][2]int16{
		1: {110, 205},
		2: {400, 205},
		3: {112, 205},
	}
	p.positionFn = func(id uint32) (int16, int16, error) {
		pos := positions[id]
		return pos[0], pos[1], nil
	}

	m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()
	sd := mobskill.NewModelBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(2).Build()

	p.executeBanish(m, sd, byte(monster2.SkillTypeBanish))

	if len(*events) != 0 {
		t.Fatalf("expected 0 events; got %d (%v)", len(*events), *events)
	}
}
