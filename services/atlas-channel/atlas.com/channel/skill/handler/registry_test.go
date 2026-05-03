package handler

import (
	"context"
	"testing"

	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

func TestLookup_NotRegistered_ReturnsFalse(t *testing.T) {
	_, ok := Lookup(skill2.Id(999999999))
	if ok {
		t.Fatalf("Lookup(unregistered) returned ok=true, want false")
	}
}

func TestRegisterLookup_RoundTrip(t *testing.T) {
	called := false
	id := skill2.Id(777777777)
	Register(id, func(_ logrus.FieldLogger) func(_ context.Context) func(
		wp writer.Producer, f field.Model, characterId uint32,
		info packetmodel.SkillUsageInfo, e effect.Model,
	) error {
		return func(_ context.Context) func(
			writer.Producer, field.Model, uint32,
			packetmodel.SkillUsageInfo, effect.Model,
		) error {
			return func(_ writer.Producer, _ field.Model, _ uint32,
				_ packetmodel.SkillUsageInfo, _ effect.Model) error {
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
