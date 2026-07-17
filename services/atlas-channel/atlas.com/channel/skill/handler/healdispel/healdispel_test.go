package healdispel

import (
	"errors"
	"io"
	"testing"

	"atlas-channel/character"
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
	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, d)
	if len(cap.hp) != 0 || len(cap.mp) != 0 || len(cap.dispelled) != 0 {
		t.Errorf("non-SuperGM caster produced effects: hp=%v mp=%v dispel=%v", cap.hp, cap.mp, cap.dispelled)
	}
	if cap.selfCount != 0 || cap.fgnCount != 0 {
		t.Errorf("non-SuperGM caster produced announces: self=%d foreign=%d, want 0/0", cap.selfCount, cap.fgnCount)
	}
}

func TestHealDispelAllRecipients(t *testing.T) {
	recips := []channelhandler.PartyRecipient{
		recip(1, 100, 1000, 100, 1000), // caster: full-restore delta 900/900
		recip(2, 950, 1000, 990, 1000), // near-full: full-restore delta 50/10
	}
	var cap capture
	d := newDeps(superGm(1), nil, false, recips, &cap)
	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, d)

	if cap.hp[1] != 900 {
		t.Errorf("recipient 1 HP delta = %d, want 900 (full restore)", cap.hp[1])
	}
	if cap.mp[1] != 900 {
		t.Errorf("recipient 1 MP delta = %d, want 900 (full restore)", cap.mp[1])
	}
	if cap.hp[2] != 50 {
		t.Errorf("recipient 2 HP delta = %d, want 50 (full restore to max)", cap.hp[2])
	}
	if cap.mp[2] != 10 {
		t.Errorf("recipient 2 MP delta = %d, want 10 (full restore to max)", cap.mp[2])
	}
	if len(cap.dispelled[1]) != 11 || len(cap.dispelled[2]) != 11 {
		t.Errorf("dispel types = %d/%d, want 11 each", len(cap.dispelled[1]), len(cap.dispelled[2]))
	}
	if cap.selfCount != 1 || cap.fgnCount != 1 {
		t.Errorf("announce self=%d foreign=%d, want 1/1 (visible caster)", cap.selfCount, cap.fgnCount)
	}
}

func TestForeignSuppressedWhenHidden(t *testing.T) {
	var cap capture
	d := newDeps(superGm(1), nil, true, []channelhandler.PartyRecipient{recip(1, 1, 100, 1, 100)}, &cap)
	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, d)
	if cap.selfCount != 1 || cap.fgnCount != 0 {
		t.Errorf("announce self=%d foreign=%d, want 1/0 (hidden caster)", cap.selfCount, cap.fgnCount)
	}
}

// TestForeignSuppressedWhenHiddenStateUnknown guards the fail-safe default: if
// isGmHidden errors (hidden state unknown), the handler must treat the caster
// as HIDDEN rather than visible — better to skip one cosmetic foreign
// animation than to leak a hidden GM's position (design.md OQ-3 / FR-17). The
// self-announce and the heal/dispel effects must still occur normally.
func TestForeignSuppressedWhenHiddenStateUnknown(t *testing.T) {
	recips := []channelhandler.PartyRecipient{
		recip(1, 100, 1000, 100, 1000),
	}
	var cap capture
	d := newDeps(superGm(1), nil, false, recips, &cap)
	d.isGmHidden = func(uint32) (bool, error) { return false, errors.New("simulated buff-lookup failure") }

	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, d)

	if cap.selfCount != 1 || cap.fgnCount != 0 {
		t.Errorf("announce self=%d foreign=%d, want 1/0 (hidden state unknown must fail safe to hidden)", cap.selfCount, cap.fgnCount)
	}
	if cap.hp[1] != 900 {
		t.Errorf("recipient 1 HP delta = %d, want 900 (heal must still occur despite hidden-state lookup error)", cap.hp[1])
	}
	if cap.mp[1] != 900 {
		t.Errorf("recipient 1 MP delta = %d, want 900 (heal must still occur despite hidden-state lookup error)", cap.mp[1])
	}
	if len(cap.dispelled[1]) != 11 {
		t.Errorf("dispel types = %d, want 11 (dispel must still occur despite hidden-state lookup error)", len(cap.dispelled[1]))
	}
}

// TestPerRecipientIsolation guards the "log-and-continue, never abort"
// invariant: a ChangeHP failure for one recipient must not prevent the loop
// from processing the remaining recipients or from announcing self.
func TestPerRecipientIsolation(t *testing.T) {
	recips := []channelhandler.PartyRecipient{
		recip(1, 100, 1000, 100, 1000), // caster: will fail ChangeHP
		recip(2, 950, 1000, 990, 1000), // must still be fully processed
	}
	var cap capture
	d := newDeps(superGm(1), nil, false, recips, &cap)
	d.changeHP = func(_ field.Model, id uint32, amt int16) error {
		if id == 1 {
			return errors.New("simulated ChangeHP failure for recipient 1")
		}
		cap.hp[id] = amt
		return nil
	}

	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, d)

	if _, ok := cap.hp[1]; ok {
		t.Errorf("recipient 1 HP delta recorded despite simulated ChangeHP failure: %v", cap.hp[1])
	}
	if cap.hp[2] != 50 {
		t.Errorf("recipient 2 HP delta = %d, want 50 (loop must continue past recipient 1's failure)", cap.hp[2])
	}
	if cap.mp[2] != 10 {
		t.Errorf("recipient 2 MP delta = %d, want 10 (loop must continue past recipient 1's failure)", cap.mp[2])
	}
	if len(cap.dispelled[1]) != 11 || len(cap.dispelled[2]) != 11 {
		t.Errorf("dispel types = %d/%d, want 11 each (dispel must still run for both recipients)", len(cap.dispelled[1]), len(cap.dispelled[2]))
	}
	if cap.selfCount != 1 {
		t.Errorf("selfCount = %d, want 1 (self-announce must still fire despite the per-recipient failure)", cap.selfCount)
	}
}

// TestEffectiveMaxFallsBackToBase guards effectiveMaxOrBase: when the
// effective-stats seam returns (0, 0, nil) for a recipient, the recipient's
// BASE max must be used for the full-restore delta, not zero.
func TestEffectiveMaxFallsBackToBase(t *testing.T) {
	recips := []channelhandler.PartyRecipient{
		recip(1, 100, 1000, 100, 1000), // base hp 1000/mp 1000; effectiveMax stubbed to (0,0,nil)
	}
	var cap capture
	d := newDeps(superGm(1), nil, false, recips, &cap)
	d.effectiveMax = func(_ field.Model, _ uint32) (uint32, uint32, error) {
		return 0, 0, nil
	}

	_ = applyHealDispel(tl(), field.NewBuilder(0, 0, 1).Build(), 1, d)

	if cap.hp[1] != 900 {
		t.Errorf("recipient 1 HP delta = %d, want 900 (fall back to base maxHp 1000)", cap.hp[1])
	}
	if cap.mp[1] != 900 {
		t.Errorf("recipient 1 MP delta = %d, want 900 (fall back to base maxMp 1000)", cap.mp[1])
	}
}
