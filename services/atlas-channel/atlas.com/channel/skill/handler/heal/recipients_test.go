package heal

import (
	"atlas-channel/skill/handler"
	"testing"
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
		handler.NewPartyRecipientBuilder().SetId(2).SetHp(100).SetMaxHp(500).Build(),
		handler.NewPartyRecipientBuilder().SetId(3).SetHp(700).SetMaxHp(700).Build(),
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
