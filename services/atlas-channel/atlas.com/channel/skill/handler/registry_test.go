package handler

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func TestLookup_NotRegistered_ReturnsFalse(t *testing.T) {
	_, ok := Lookup(skill2.Identity(999999999))
	if ok {
		t.Fatalf("Lookup(unregistered) returned ok=true, want false")
	}
}

func TestRegisterLookup_RoundTrip(t *testing.T) {
	called := false
	id := skill2.Identity(777777777)
	Register(id, func(_ logrus.FieldLogger) func(_ context.Context) func(
		wp writer.Producer, f field.Model, characterId uint32,
		info packetmodel.SkillUsageInfo, e effect.Model,
	) error {
		return func(_ context.Context) func(
			writer.Producer, field.Model, uint32,
			packetmodel.SkillUsageInfo, effect.Model,
		) error {
			return func(_ writer.Producer, _ field.Model, _ uint32,
				_ packetmodel.SkillUsageInfo, _ effect.Model,
			) error {
				called = true
				return nil
			}
		}
	})
	defer delete(registry, id)

	h, ok := Lookup(id)
	if !ok {
		t.Fatalf("Lookup after Register returned ok=false, want true")
	}
	if h == nil {
		t.Fatalf("Lookup returned nil handler")
	}
	_ = called
}

// TestDispatch_v48HideNotCorkscrew is the v0.48 correctness proof this
// task-187 structural fix exists for: wire 5101004 means SuperGmHide at
// v0.48 (GM/SuperGM skill band) and BrawlerCorkscrewBlow at v0.62+ (Brawler
// keydown attack). A registry keyed on the raw wire id cannot distinguish
// these -- dispatch MUST resolve the wire id through the tenant's version
// set to an Identity BEFORE calling Lookup. Register a sentinel handler
// under skill2.SuperGmHide only, then confirm the v48-resolved identity
// finds it and the v72-resolved identity (BrawlerCorkscrewBlow, never
// registered here) does not.
func TestDispatch_v48HideNotCorkscrew(t *testing.T) {
	Register(skill2.SuperGmHide, func(_ logrus.FieldLogger) func(_ context.Context) func(
		wp writer.Producer, f field.Model, characterId uint32,
		info packetmodel.SkillUsageInfo, e effect.Model,
	) error {
		return func(_ context.Context) func(
			writer.Producer, field.Model, uint32,
			packetmodel.SkillUsageInfo, effect.Model,
		) error {
			return func(_ writer.Producer, _ field.Model, _ uint32,
				_ packetmodel.SkillUsageInfo, _ effect.Model,
			) error {
				return nil
			}
		}
	})
	t.Cleanup(func() { Unregister(skill2.SuperGmHide) })

	id48, ok48 := constants.For("GMS", 48, 1).Skill.Resolve(skill2.Id(5101004))
	if !ok48 {
		t.Fatalf("v48 wire 5101004 failed to resolve to any identity")
	}
	if id48 != skill2.SuperGmHide {
		t.Fatalf("v48 wire 5101004 resolved to %v, want SuperGmHide", id48)
	}
	if _, ok := Lookup(id48); !ok {
		t.Fatal("v48 wire 5101004 (SuperGmHide) must dispatch the Hide handler")
	}

	id72, ok72 := constants.For("GMS", 72, 1).Skill.Resolve(skill2.Id(5101004))
	if !ok72 {
		t.Fatalf("v72 wire 5101004 failed to resolve to any identity")
	}
	if id72 != skill2.BrawlerCorkscrewBlow {
		t.Fatalf("v72 wire 5101004 resolved to %v, want BrawlerCorkscrewBlow", id72)
	}
	if _, ok := Lookup(id72); ok {
		t.Fatal("v72 wire 5101004 (BrawlerCorkscrewBlow) must NOT dispatch the Hide handler")
	}
}
