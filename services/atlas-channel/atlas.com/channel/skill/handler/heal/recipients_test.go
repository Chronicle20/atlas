package heal

import (
	"testing"

	"atlas-channel/data/skill/effect"
	"atlas-channel/skill/handler"
)

func TestSelectRecipients_CasterAlwaysIncluded(t *testing.T) {
	caster := recipient{Id: 1, X: 0, Y: 0, Hp: 500, MaxHp: 1000, IsCaster: true}
	got := selectRecipients(caster, nil)
	if len(got) != 1 || got[0] != caster {
		t.Fatalf("caster-only result = %#v, want [caster]", got)
	}
}

func TestSelectRecipients_PrependsCasterToParty(t *testing.T) {
	caster := recipient{Id: 1, Hp: 500, MaxHp: 1000, IsCaster: true}
	party := []handler.PartyRecipient{
		{Id: 2, Hp: 100, MaxHp: 500},
		{Id: 3, Hp: 700, MaxHp: 700},
	}
	got := selectRecipients(caster, party)
	if len(got) != 3 {
		t.Fatalf("recipients len = %d, want 3", len(got))
	}
	if got[0].Id != 1 || !got[0].IsCaster {
		t.Fatalf("recipients[0] = %#v, want caster", got[0])
	}
	if got[1].Id != 2 || got[2].Id != 3 {
		t.Fatalf("recipients ids = %v, want [1,2,3]", []uint32{got[0].Id, got[1].Id, got[2].Id})
	}
}

func TestWarnIfMissingRectangle_OncePerTuple(t *testing.T) {
	defer resetWarnedRectangles()

	calls := 0
	logf := func() { calls++ }

	warnIfMissingRectangle(2301002, 1, effect.Model{}, logf)
	warnIfMissingRectangle(2301002, 1, effect.Model{}, logf)
	warnIfMissingRectangle(2301002, 2, effect.Model{}, logf)

	if calls != 2 {
		t.Fatalf("warn calls = %d, want 2 (one per (id, level))", calls)
	}
}
