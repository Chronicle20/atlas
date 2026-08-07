package dispel

import (
	"atlas-channel/data/skill/effect"
	"context"
	"errors"
	"io"
	"testing"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func tl() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func recip(id uint32) channelhandler.PartyRecipient {
	return channelhandler.NewPartyRecipientBuilder().SetId(id).Build()
}

func testInfo() packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(2311001).
		SetSkillLevel(1).
		SetAffectedPartyMemberBitmap(0x30).
		Build()
}

// cancelCall records a single CancelByTypes invocation.
type cancelCall struct {
	characterId uint32
	types       []string
}

func TestDispelRegisteredOnIdentity(t *testing.T) {
	h, ok := channelhandler.Lookup(skill2.PriestDispel)
	if !ok {
		t.Fatalf("expected skill2.PriestDispel to be registered")
	}
	if h == nil {
		t.Fatalf("expected non-nil handler for skill2.PriestDispel")
	}
}

func TestDispelCuresCasterAndMembersInOrder(t *testing.T) {
	origSelect, origProp := selectPartyMembersFunc, propRollFunc
	t.Cleanup(func() {
		selectPartyMembersFunc = origSelect
		propRollFunc = origProp
	})

	selectPartyMembersFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, bitmap byte) []channelhandler.PartyRecipient {
		return []channelhandler.PartyRecipient{recip(2), recip(3)}
	}
	propRollFunc = func(prop float64) bool { return true }

	var calls []cancelCall
	origCancel := cancelByTypesFunc
	t.Cleanup(func() { cancelByTypesFunc = origCancel })
	cancelByTypesFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, types []string) error {
		calls = append(calls, cancelCall{characterId: characterId, types: types})
		return nil
	}

	err := Apply(tl())(context.Background())(nil, field.NewBuilder(0, 0, 1).Build(), 1, testInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	if len(calls) != 3 {
		t.Fatalf("expected 3 cure calls, got %d", len(calls))
	}
	wantOrder := []uint32{1, 2, 3}
	for i, c := range calls {
		if c.characterId != wantOrder[i] {
			t.Errorf("call %d characterId = %d, want %d", i, c.characterId, wantOrder[i])
		}
	}
	wantTypes := []string{"CURSE", "DARKNESS", "POISON", "SEAL", "WEAKEN", "SLOW"}
	for i, c := range calls {
		if len(c.types) != len(wantTypes) {
			t.Fatalf("call %d types = %v, want %v", i, c.types, wantTypes)
		}
		for j, ty := range wantTypes {
			if c.types[j] != ty {
				t.Errorf("call %d types[%d] = %q, want %q", i, j, c.types[j], ty)
			}
		}
	}
}

func TestDispelSelectorReceivesCastArgs(t *testing.T) {
	origSelect, origProp := selectPartyMembersFunc, propRollFunc
	t.Cleanup(func() {
		selectPartyMembersFunc = origSelect
		propRollFunc = origProp
	})
	propRollFunc = func(prop float64) bool { return true }

	var gotF field.Model
	var gotCaster uint32
	var gotBitmap byte
	var called bool
	selectPartyMembersFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, bitmap byte) []channelhandler.PartyRecipient {
		called = true
		gotF = f
		gotCaster = casterId
		gotBitmap = bitmap
		return nil
	}

	origCancel := cancelByTypesFunc
	t.Cleanup(func() { cancelByTypesFunc = origCancel })
	cancelByTypesFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, types []string) error {
		return nil
	}

	wantF := field.NewBuilder(0, 0, 1).Build()
	_ = Apply(tl())(context.Background())(nil, wantF, 42, testInfo(), effect.Model{})

	if !called {
		t.Fatalf("expected selectPartyMembersFunc to be called")
	}
	if gotCaster != 42 {
		t.Errorf("selector casterId = %d, want 42", gotCaster)
	}
	if gotBitmap != 0x30 {
		t.Errorf("selector bitmap = %#x, want 0x30", gotBitmap)
	}
	if gotF != wantF {
		t.Errorf("selector f = %v, want %v", gotF, wantF)
	}
}

func TestDispelEmptySelectorCastsCasterOnly(t *testing.T) {
	origSelect, origProp := selectPartyMembersFunc, propRollFunc
	t.Cleanup(func() {
		selectPartyMembersFunc = origSelect
		propRollFunc = origProp
	})
	selectPartyMembersFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, bitmap byte) []channelhandler.PartyRecipient {
		return nil
	}
	propRollFunc = func(prop float64) bool { return true }

	var calls []cancelCall
	origCancel := cancelByTypesFunc
	t.Cleanup(func() { cancelByTypesFunc = origCancel })
	cancelByTypesFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, types []string) error {
		calls = append(calls, cancelCall{characterId: characterId, types: types})
		return nil
	}

	err := Apply(tl())(context.Background())(nil, field.NewBuilder(0, 0, 1).Build(), 7, testInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(calls) != 1 || calls[0].characterId != 7 {
		t.Fatalf("expected single caster-only cure call, got %v", calls)
	}
}

func TestDispelPropRollPerRecipient(t *testing.T) {
	origSelect, origProp := selectPartyMembersFunc, propRollFunc
	t.Cleanup(func() {
		selectPartyMembersFunc = origSelect
		propRollFunc = origProp
	})
	selectPartyMembersFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, bitmap byte) []channelhandler.PartyRecipient {
		return []channelhandler.PartyRecipient{recip(2), recip(3)}
	}

	var rollCount int
	propRollFunc = func(prop float64) bool {
		rollCount++
		return rollCount%2 == 1 // alternating pass/fail: recipient 1 pass, 2 fail, 3 pass
	}

	var calls []cancelCall
	origCancel := cancelByTypesFunc
	t.Cleanup(func() { cancelByTypesFunc = origCancel })
	cancelByTypesFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, types []string) error {
		calls = append(calls, cancelCall{characterId: characterId, types: types})
		return nil
	}

	err := Apply(tl())(context.Background())(nil, field.NewBuilder(0, 0, 1).Build(), 1, testInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	if rollCount != 3 {
		t.Fatalf("expected propRollFunc called once per recipient (3), got %d", rollCount)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 emitted cures (recipients 1 and 3), got %d: %v", len(calls), calls)
	}
	if calls[0].characterId != 1 || calls[1].characterId != 3 {
		t.Errorf("expected cures for [1,3], got %v", calls)
	}
}

func TestDispelEmitErrorContinues(t *testing.T) {
	origSelect, origProp := selectPartyMembersFunc, propRollFunc
	t.Cleanup(func() {
		selectPartyMembersFunc = origSelect
		propRollFunc = origProp
	})
	selectPartyMembersFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, bitmap byte) []channelhandler.PartyRecipient {
		return []channelhandler.PartyRecipient{recip(2), recip(3)}
	}
	propRollFunc = func(prop float64) bool { return true }

	var calls []cancelCall
	origCancel := cancelByTypesFunc
	t.Cleanup(func() { cancelByTypesFunc = origCancel })
	cancelByTypesFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, types []string) error {
		if characterId == 2 {
			return errors.New("simulated cancel failure for recipient 2")
		}
		calls = append(calls, cancelCall{characterId: characterId, types: types})
		return nil
	}

	err := Apply(tl())(context.Background())(nil, field.NewBuilder(0, 0, 1).Build(), 1, testInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected recipients before and after the failing one still emitted (2), got %d: %v", len(calls), calls)
	}
	if calls[0].characterId != 1 || calls[1].characterId != 3 {
		t.Errorf("expected cures for [1,3] (recipient 2 failed), got %v", calls)
	}
}

func TestDispelZeroPropCuresNobody(t *testing.T) {
	origSelect := selectPartyMembersFunc
	t.Cleanup(func() { selectPartyMembersFunc = origSelect })
	selectPartyMembersFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, bitmap byte) []channelhandler.PartyRecipient {
		return []channelhandler.PartyRecipient{recip(2), recip(3)}
	}

	var calls []cancelCall
	origCancel := cancelByTypesFunc
	t.Cleanup(func() { cancelByTypesFunc = origCancel })
	cancelByTypesFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, types []string) error {
		calls = append(calls, cancelCall{characterId: characterId, types: types})
		return nil
	}

	// Real propRollFunc (not overridden) with the zero-value effect.Model:
	// Prop() is 0 -> the real roll always fails.
	err := Apply(tl())(context.Background())(nil, field.NewBuilder(0, 0, 1).Build(), 1, testInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("expected zero cures with zero prop, got %d: %v", len(calls), calls)
	}
}

func TestPropRollBoundaries(t *testing.T) {
	if propRollFunc(0) {
		t.Errorf("propRollFunc(0) = true, want false")
	}
	if propRollFunc(-0.5) {
		t.Errorf("propRollFunc(-0.5) = true, want false")
	}
	if !propRollFunc(1.0) {
		t.Errorf("propRollFunc(1.0) = false, want true")
	}
	if !propRollFunc(1.5) {
		t.Errorf("propRollFunc(1.5) = false, want true")
	}
}

func TestDispelSummaryLogFields(t *testing.T) {
	origSelect, origProp := selectPartyMembersFunc, propRollFunc
	t.Cleanup(func() {
		selectPartyMembersFunc = origSelect
		propRollFunc = origProp
	})
	selectPartyMembersFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, bitmap byte) []channelhandler.PartyRecipient {
		return []channelhandler.PartyRecipient{recip(2), recip(3)}
	}

	var rollCount int
	propRollFunc = func(prop float64) bool {
		rollCount++
		return rollCount%2 == 1 // recipient 1 pass, 2 fail, 3 pass
	}

	origCancel := cancelByTypesFunc
	t.Cleanup(func() { cancelByTypesFunc = origCancel })
	cancelByTypesFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, types []string) error {
		return nil
	}

	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	err := Apply(logger)(context.Background())(nil, field.NewBuilder(0, 0, 1).Build(), 9, testInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	var entry *logrus.Entry
	for _, e := range hook.AllEntries() {
		if e.Message == "dispel_party_cure_summary" {
			entry = e
			break
		}
	}
	if entry == nil {
		t.Fatalf("expected a dispel_party_cure_summary log entry")
	}
	if entry.Level != logrus.DebugLevel {
		t.Errorf("summary log level = %v, want Debug", entry.Level)
	}

	if got := entry.Data["caster"]; got != uint32(9) {
		t.Errorf("caster = %v, want 9", got)
	}
	if got := entry.Data["bitmap"]; got != byte(0x30) {
		t.Errorf("bitmap = %v, want 0x30", got)
	}
	if got := entry.Data["recipients_selected"]; got != 3 {
		t.Errorf("recipients_selected = %v, want 3", got)
	}
	if got := entry.Data["cures_emitted"]; got != 2 {
		t.Errorf("cures_emitted = %v, want 2", got)
	}
	if got := entry.Data["prop_skipped"]; got != 1 {
		t.Errorf("prop_skipped = %v, want 1", got)
	}
}
