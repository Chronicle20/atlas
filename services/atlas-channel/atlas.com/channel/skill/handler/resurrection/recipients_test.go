package resurrection

import (
	"context"
	"io"
	"testing"

	"atlas-channel/data/skill/effect"
	channelhandler "atlas-channel/skill/handler"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
}

func installSelectorSpies(t *testing.T) (partyCalled, mapCalled *bool) {
	t.Helper()
	pc, mc := false, false
	prevParty, prevMap := selectDeadParty, selectDeadMap
	selectDeadParty = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _, _ int16, _ effect.Model, _ byte) []channelhandler.PartyRecipient {
		pc = true
		return nil
	}
	selectDeadMap = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _, _ int16, _ effect.Model) []channelhandler.PartyRecipient {
		mc = true
		return nil
	}
	t.Cleanup(func() {
		selectDeadParty = prevParty
		selectDeadMap = prevMap
	})
	return &pc, &mc
}

func TestSelectByVariant_BishopUsesPartySelector(t *testing.T) {
	pc, mc := installSelectorSpies(t)
	selectByVariant(testLogger(), context.Background(), testField(), 1, 0, 0, effect.Model{}, 0x7E, skill2.BishopResurrectionId)
	if !*pc || *mc {
		t.Fatalf("Bishop: partyCalled=%v mapCalled=%v, want party only", *pc, *mc)
	}
}

func TestSelectByVariant_GmUsesMapSelector(t *testing.T) {
	pc, mc := installSelectorSpies(t)
	selectByVariant(testLogger(), context.Background(), testField(), 1, 0, 0, effect.Model{}, 0x7E, skill2.GmResurrectionId)
	if *pc || !*mc {
		t.Fatalf("GM: partyCalled=%v mapCalled=%v, want map only", *pc, *mc)
	}
}

func TestSelectByVariant_SuperGmUsesMapSelector(t *testing.T) {
	pc, mc := installSelectorSpies(t)
	selectByVariant(testLogger(), context.Background(), testField(), 1, 0, 0, effect.Model{}, 0x7E, skill2.SuperGmResurrectionId)
	if *pc || !*mc {
		t.Fatalf("SuperGM: partyCalled=%v mapCalled=%v, want map only", *pc, *mc)
	}
}
