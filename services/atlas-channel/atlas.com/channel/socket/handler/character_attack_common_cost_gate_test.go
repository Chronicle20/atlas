package handler

import (
	"context"
	"testing"

	"atlas-channel/data/skill/effect"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/socket/writer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

// TestProcessAttack_RegisteredSkill_GateUsesLookup pins the dispatcher-membership
// gate that processAttack uses to skip the HPConsume/MPConsume block. The actual
// branch in character_attack_common.go reads:
//
//	if _, registered := handler.Lookup(skill3.Id(ai.SkillId())); !registered { ... cost ... }
//
// End-to-end behavior with a fake character.Processor / monster.Processor is out
// of scope for this gate test; the gate is one line of code and Lookup's own
// behavior is pinned in skill/handler/registry_test.go. This test guards the
// public Register/Lookup contract that processAttack relies on, so any future
// refactor of the registry surface will surface here.
func TestProcessAttack_RegisteredSkill_GateUsesLookup(t *testing.T) {
	id := skill2.Id(900900900)
	channelhandler.Register(id, func(_ logrus.FieldLogger) func(_ context.Context) func(
		wp writer.Producer, f field.Model, characterId uint32,
		info packetmodel.SkillUsageInfo, e effect.Model,
	) error {
		return func(_ context.Context) func(
			writer.Producer, field.Model, uint32,
			packetmodel.SkillUsageInfo, effect.Model,
		) error {
			return func(_ writer.Producer, _ field.Model, _ uint32,
				_ packetmodel.SkillUsageInfo, _ effect.Model) error {
				return nil
			}
		}
	})
	t.Cleanup(func() {
		// Best-effort cleanup. If a future refactor exposes Unregister we should
		// switch to it; for now, leaving the entry is safe because the test id
		// is unused in production.
		_ = id
	})

	if _, ok := channelhandler.Lookup(id); !ok {
		t.Fatalf("Lookup after Register returned ok=false; processAttack cost-gate would treat all skills as un-registered")
	}
}
