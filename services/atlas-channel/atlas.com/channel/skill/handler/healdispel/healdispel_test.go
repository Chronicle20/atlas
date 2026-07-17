package healdispel

import (
	"io"
	"testing"

	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	channelhandler "atlas-channel/skill/handler"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/sirupsen/logrus"
)

func tl() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func superGm(id uint32) character.Model {
	return character.NewModelBuilder().SetId(id).SetLevel(200).SetJobId(job.SuperGmId).MustBuild()
}

func recip(id uint32, hp, maxHp, mp, maxMp uint16) channelhandler.PartyRecipient {
	return channelhandler.NewPartyRecipientBuilder().
		SetId(id).SetHp(hp).SetMaxHp(maxHp).SetMp(mp).SetMaxMp(maxMp).Build()
}

type capture struct {
	hp        map[uint32]int16
	mp        map[uint32]int16
	dispelled map[uint32][]string
	selfCount int
	fgnCount  int
}

func newDeps(caster character.Model, casterErr error, hidden bool, recips []channelhandler.PartyRecipient, c *capture) healDispelDeps {
	c.hp = map[uint32]int16{}
	c.mp = map[uint32]int16{}
	c.dispelled = map[uint32][]string{}
	return healDispelDeps{
		loadCaster:  func(uint32) (character.Model, error) { return caster, casterErr },
		isGmHidden:  func(uint32) (bool, error) { return hidden, nil },
		selectInMap: func(field.Model) []channelhandler.PartyRecipient { return recips },
		// Effective max mirrors base for the test (no gear bonus).
		effectiveMax: func(_ field.Model, id uint32) (uint32, uint32, error) {
			for _, r := range recips {
				if r.Id() == id {
					return uint32(r.MaxHp()), uint32(r.MaxMp()), nil
				}
			}
			return 0, 0, nil
		},
		changeHP:        func(_ field.Model, id uint32, amt int16) error { c.hp[id] = amt; return nil },
		changeMP:        func(_ field.Model, id uint32, amt int16) error { c.mp[id] = amt; return nil },
		dispel:          func(_ field.Model, id uint32, types []string) error { c.dispelled[id] = types; return nil },
		announceSelf:    func(byte) error { c.selfCount++; return nil },
		announceForeign: func(byte) error { c.fgnCount++; return nil },
	}
}

func TestNonSuperGmRejected(t *testing.T) {
	nonGm := character.NewModelBuilder().SetId(1).SetJobId(job.Id(100)).MustBuild() // Warrior
	var cap capture
	d := newDeps(nonGm, nil, false, []channelhandler.PartyRecipient{recip(1, 1, 100, 1, 100)}, &cap)
	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, mustEffect(t, effect.RestModel{Hp: 10}), d)
	if len(cap.hp) != 0 || len(cap.mp) != 0 || len(cap.dispelled) != 0 {
		t.Errorf("non-SuperGM caster produced effects: hp=%v mp=%v dispel=%v", cap.hp, cap.mp, cap.dispelled)
	}
}

func TestHealDispelAllRecipients(t *testing.T) {
	// hp=100 flat + 0.5*maxHp(1000)=500 -> 600 restore, clamped to headroom 900 -> 600.
	e := mustEffect(t, effect.RestModel{Hp: 100, Mp: 50, HPR: 0.5, MPR: 0.5})
	recips := []channelhandler.PartyRecipient{
		recip(1, 100, 1000, 100, 1000), // caster
		recip(2, 950, 1000, 990, 1000), // near-full: HP headroom 50, MP headroom 10
	}
	var cap capture
	d := newDeps(superGm(1), nil, false, recips, &cap)
	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, e, d)

	if cap.hp[1] != 600 {
		t.Errorf("recipient 1 HP delta = %d, want 600", cap.hp[1])
	}
	if cap.hp[2] != 50 { // clamped to headroom
		t.Errorf("recipient 2 HP delta = %d, want 50 (clamped)", cap.hp[2])
	}
	if cap.mp[2] != 10 { // 50 flat + 500 ratio -> clamped to headroom 10
		t.Errorf("recipient 2 MP delta = %d, want 10 (clamped)", cap.mp[2])
	}
	if len(cap.dispelled[1]) != 11 || len(cap.dispelled[2]) != 11 {
		t.Errorf("dispel types = %d/%d, want 11 each", len(cap.dispelled[1]), len(cap.dispelled[2]))
	}
	if cap.selfCount != 1 || cap.fgnCount != 1 {
		t.Errorf("announce self=%d foreign=%d, want 1/1 (visible caster)", cap.selfCount, cap.fgnCount)
	}
}

func TestForeignSuppressedWhenHidden(t *testing.T) {
	e := mustEffect(t, effect.RestModel{Hp: 10})
	var cap capture
	d := newDeps(superGm(1), nil, true, []channelhandler.PartyRecipient{recip(1, 1, 100, 1, 100)}, &cap)
	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, e, d)
	if cap.selfCount != 1 || cap.fgnCount != 0 {
		t.Errorf("announce self=%d foreign=%d, want 1/0 (hidden caster)", cap.selfCount, cap.fgnCount)
	}
}

func mustEffect(t *testing.T, rm effect.RestModel) effect.Model {
	t.Helper()
	m, err := effect.Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return m
}
