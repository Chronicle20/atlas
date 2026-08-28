package monster

import (
	"atlas-monsters/monster/information"
	"atlas-monsters/monster/mobskill"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
)

// TestExecuteDiseaseCaller verifies executeDispel and executeBanish inherit
// box (and, for banish, banish-map) scoping purely by sharing
// getDiseaseTargets.
func TestExecuteDiseaseCaller(t *testing.T) {
	tests := []struct {
		name           string
		inField        []uint32
		positions      map[uint32][2]int16
		count          uint32
		infoModel      func() information.Model
		execute        func(p *ProcessorImpl, m Model, sd mobskill.Model)
		wantEventCount int
		wantTopic      string
	}{
		{
			name:    "dispel targets only in-box characters",
			inField: []uint32{1, 2, 3},
			positions: map[uint32][2]int16{
				1: {110, 205},
				2: {400, 205},
				3: {112, 205},
			},
			count: 2,
			execute: func(p *ProcessorImpl, m Model, sd mobskill.Model) {
				p.executeDispel(m, sd, byte(monster2.SkillTypeDispel))
			},
			wantEventCount: 2,
			wantTopic:      EnvCommandTopicCharacterBuff,
		},
		{
			name:    "dispel has no cap for non-seduce",
			inField: []uint32{1, 2, 3, 4},
			positions: map[uint32][2]int16{
				1: {110, 205},
				2: {111, 205},
				3: {112, 205},
				4: {113, 205},
			},
			count: 2,
			execute: func(p *ProcessorImpl, m Model, sd mobskill.Model) {
				p.executeDispel(m, sd, byte(monster2.SkillTypeDispel))
			},
			wantEventCount: 4,
			wantTopic:      EnvCommandTopicCharacterBuff,
		},
		{
			name:    "banish targets only in-box characters",
			inField: []uint32{1, 2, 3},
			positions: map[uint32][2]int16{
				1: {110, 205},
				2: {400, 205},
				3: {112, 205},
			},
			count: 2,
			infoModel: func() information.Model {
				return information.NewBuilder().SetBanish(information.Banish{MapId: 104000000}).Build()
			},
			execute: func(p *ProcessorImpl, m Model, sd mobskill.Model) {
				p.executeBanish(m, sd, byte(monster2.SkillTypeBanish))
			},
			wantEventCount: 2,
			wantTopic:      EnvCommandTopicPortal,
		},
		{
			name:    "banish with no banish map emits nothing",
			inField: []uint32{1, 2, 3},
			positions: map[uint32][2]int16{
				1: {110, 205},
				2: {400, 205},
				3: {112, 205},
			},
			count: 2,
			infoModel: func() information.Model {
				return information.NewBuilder().Build()
			},
			execute: func(p *ProcessorImpl, m Model, sd mobskill.Model) {
				p.executeBanish(m, sd, byte(monster2.SkillTypeBanish))
			},
			wantEventCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.infoModel != nil {
				prevHook := testInformationLookup
				testInformationLookup = func(_ uint32) (information.Model, error) {
					return tt.infoModel(), nil
				}
				defer func() { testInformationLookup = prevHook }()
			}

			p, events := newRecordingProcessor(t, newTestTenant(t))
			p.inFieldFn = func(_ field.Model) ([]uint32, error) {
				return tt.inField, nil
			}
			p.positionFn = func(id uint32) (int16, int16, error) {
				pos := tt.positions[id]
				return pos[0], pos[1], nil
			}

			m := Clone(Model{}).SetX(100).SetY(200).SetControlCharacterId(7).Build()
			sd := mobskill.NewBuilder().SetBoundingBox(-50, -30, 50, 30).SetCount(tt.count).Build()

			tt.execute(p, m, sd)

			if len(*events) != tt.wantEventCount {
				t.Fatalf("expected %d events; got %d (%v)", tt.wantEventCount, len(*events), *events)
			}
			if tt.wantTopic == "" {
				return
			}
			for _, e := range *events {
				if e.Topic != tt.wantTopic {
					t.Fatalf("expected topic %s; got %s", tt.wantTopic, e.Topic)
				}
			}
		})
	}
}
