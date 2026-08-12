package handler

import (
	"atlas-channel/character/combo"
	"context"
	"testing"
	"time"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestAranComboClearOnCancel(t *testing.T) {
	tests := []struct {
		name      string
		cancelled skill3.Id
		want      bool
	}{
		{"combo ability cancel clears", skill3.AranStage1ComboAbilityId, true},
		{"legend variant cancel clears", skill3.LegendComboAbilityId, true},
		{"unrelated buff cancel does not clear", skill3.AranStage4ComboBarrierId, false},
	}
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tn := comboTestTenant(t)
			ctx := tenant.WithContext(context.Background(), tn)
			characterId := uint32(30 + i)

			combo.GetMirror().SetEligibility(tn, characterId, comboTestField(),
				combo.NewEligibility(skill3.AranStage1ComboAbilityId, 5, 5), time.Now())
			combo.GetMirror().Increment(tn, characterId, comboTestField(), combo.DefaultIdleWindow, time.Now())

			if got := aranComboClearOnCancel(ctx, characterId, tc.cancelled); got != tc.want {
				t.Fatalf("cleared: want %v, got %v", tc.want, got)
			}

			_, present := combo.GetMirror().Eligibility(tn, characterId, time.Now(), time.Minute)
			if tc.want && present {
				t.Error("entry survived a Combo Ability cancel")
			}
			if !tc.want && !present {
				t.Error("entry was dropped by an unrelated cancel")
			}
		})
	}
}
