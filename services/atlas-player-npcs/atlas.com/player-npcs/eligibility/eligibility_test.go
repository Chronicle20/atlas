package eligibility

import (
	"atlas-player-npcs/character"
	"atlas-player-npcs/configuration"
	"testing"
)

func buildCharacter(t *testing.T, level byte, gm bool) character.Model {
	t.Helper()
	gmAttr := 0
	if gm {
		gmAttr = 1
	}
	// jobId 100 (a stable branch category) -- job.MaxLevelFor(100) is 200
	// today for every job line (libs/atlas-constants/job/max_level.go).
	m, err := character.Extract(character.RestModel{JobId: 100, Level: level, Gm: gmAttr})
	if err != nil {
		t.Fatalf("character.Extract returned error: %v", err)
	}
	return m
}

func buildConfig(t *testing.T, autoDeployEnabled bool) configuration.Model {
	t.Helper()
	m, err := configuration.Extract(configuration.RestModel{AutoDeployEnabled: autoDeployEnabled})
	if err != nil {
		t.Fatalf("configuration.Extract returned error: %v", err)
	}
	return m
}

func TestEligible(t *testing.T) {
	tests := []struct {
		name             string
		autoDeploy       bool
		level            byte
		gm               bool
		existing         int
		conversationPath bool
		wantOk           bool
		wantReason       string
	}{
		{
			name:       "eligible for auto-deploy",
			autoDeploy: true, level: 200, gm: false, existing: 0,
			conversationPath: false,
			wantOk:           true, wantReason: "",
		},
		{
			name:       "below max level",
			autoDeploy: true, level: 199, gm: false, existing: 0,
			conversationPath: false,
			wantOk:           false, wantReason: ReasonIneligible,
		},
		{
			name:       "is a GM",
			autoDeploy: true, level: 200, gm: true, existing: 0,
			conversationPath: false,
			wantOk:           false, wantReason: ReasonIneligible,
		},
		{
			name:       "already deployed on the map",
			autoDeploy: true, level: 200, gm: false, existing: 1,
			conversationPath: false,
			wantOk:           false, wantReason: ReasonDuplicate,
		},
		{
			name:       "conversation predicate needs auto-deploy off",
			autoDeploy: true, level: 200, gm: false, existing: 0,
			conversationPath: true,
			wantOk:           false, wantReason: ReasonIneligible,
		},
		{
			name:       "conversation predicate satisfied",
			autoDeploy: false, level: 200, gm: false, existing: 0,
			conversationPath: true,
			wantOk:           true, wantReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := buildConfig(t, tt.autoDeploy)
			c := buildCharacter(t, tt.level, tt.gm)

			ok, reason := Evaluate(cfg, c, tt.existing, tt.conversationPath)
			if ok != tt.wantOk {
				t.Errorf("Evaluate() ok = %v, want %v", ok, tt.wantOk)
			}
			if reason != tt.wantReason {
				t.Errorf("Evaluate() reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}
