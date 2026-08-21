package monster

import (
	"math"
	"testing"
)

func TestComputeAward(t *testing.T) {
	tests := []struct {
		name            string
		level           byte
		pooledExp       float64
		totalPartyLevel uint32
		partyBonusMod   float64
		isMvp           bool
		expRate         float64
		wantPersonal    uint32
		wantBonus       uint32
		wantGuarded     bool
	}{
		{
			name:            "solo identity",
			level:           50,
			pooledExp:       1000,
			totalPartyLevel: 50,
			partyBonusMod:   0.0,
			isMvp:           true,
			expRate:         1.0,
			wantPersonal:    1000,
			wantBonus:       0,
			wantGuarded:     false,
		},
		{
			name:            "solo with rate",
			level:           50,
			pooledExp:       1000,
			totalPartyLevel: 50,
			partyBonusMod:   0.0,
			isMvp:           true,
			expRate:         2.0,
			wantPersonal:    2000,
			wantBonus:       0,
			wantGuarded:     false,
		},
		{
			name:            "one-member party equals solo (FR-5.11)",
			level:           50,
			pooledExp:       1000,
			totalPartyLevel: 50,
			partyBonusMod:   0.0,
			isMvp:           true,
			expRate:         1.0,
			wantPersonal:    1000,
			wantBonus:       0,
			wantGuarded:     false,
		},
		{
			name:            "two-member party, MVP",
			level:           50,
			pooledExp:       1000,
			totalPartyLevel: 100,
			partyBonusMod:   0.10,
			isMvp:           true,
			expRate:         1.0,
			wantPersonal:    600,
			wantBonus:       60,
			wantGuarded:     false,
		},
		{
			name:            "two-member party, non-MVP",
			level:           50,
			pooledExp:       1000,
			totalPartyLevel: 100,
			partyBonusMod:   0.10,
			isMvp:           false,
			expRate:         1.0,
			wantPersonal:    400,
			wantBonus:       40,
			wantGuarded:     false,
		},
		{
			name:            "unequal levels, MVP",
			level:           100,
			pooledExp:       1000,
			totalPartyLevel: 150,
			partyBonusMod:   0.10,
			isMvp:           true,
			expRate:         1.0,
			wantPersonal:    733,
			wantBonus:       73,
			wantGuarded:     false,
		},
		{
			name:            "unequal levels, non-MVP",
			level:           50,
			pooledExp:       1000,
			totalPartyLevel: 150,
			partyBonusMod:   0.10,
			isMvp:           false,
			expRate:         1.0,
			wantPersonal:    266,
			wantBonus:       26,
			wantGuarded:     false,
		},
		{
			name:            "bonus is rate-multiplied (FR-8.2)",
			level:           50,
			pooledExp:       1000,
			totalPartyLevel: 100,
			partyBonusMod:   0.10,
			isMvp:           true,
			expRate:         3.0,
			wantPersonal:    1800,
			wantBonus:       180,
			wantGuarded:     false,
		},
		{
			name:            "four-member party bonus",
			level:           50,
			pooledExp:       1000,
			totalPartyLevel: 200,
			partyBonusMod:   0.20,
			isMvp:           false,
			expRate:         1.0,
			wantPersonal:    200,
			wantBonus:       40,
			wantGuarded:     false,
		},
		{
			name:            "zero pooled exp",
			level:           50,
			pooledExp:       0,
			totalPartyLevel: 100,
			partyBonusMod:   0.10,
			isMvp:           true,
			expRate:         1.0,
			wantPersonal:    0,
			wantBonus:       0,
			wantGuarded:     false,
		},
		{
			name:            "zero rate",
			level:           50,
			pooledExp:       1000,
			totalPartyLevel: 100,
			partyBonusMod:   0.10,
			isMvp:           true,
			expRate:         0.0,
			wantPersonal:    0,
			wantBonus:       0,
			wantGuarded:     false,
		},
		{
			name:            "zero totalPartyLevel is guarded (FR-8.6)",
			level:           50,
			pooledExp:       1000,
			totalPartyLevel: 0,
			partyBonusMod:   0.10,
			isMvp:           true,
			expRate:         1.0,
			wantPersonal:    0,
			wantBonus:       0,
			wantGuarded:     true,
		},
		{
			name:            "zero level, non-zero party level",
			level:           0,
			pooledExp:       1000,
			totalPartyLevel: 100,
			partyBonusMod:   0.0,
			isMvp:           false,
			expRate:         1.0,
			wantPersonal:    0,
			wantBonus:       0,
			wantGuarded:     false,
		},
		{
			name:            "overflow clamps",
			level:           50,
			pooledExp:       1e18,
			totalPartyLevel: 50,
			partyBonusMod:   0.0,
			isMvp:           true,
			expRate:         1.0,
			wantPersonal:    math.MaxUint32,
			wantBonus:       0,
			wantGuarded:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultExperienceConfig()
			r := Recipient{
				Level:           tt.level,
				PooledExp:       tt.pooledExp,
				TotalPartyLevel: tt.totalPartyLevel,
				PartyBonusMod:   tt.partyBonusMod,
				IsMvp:           tt.isMvp,
			}

			personal, bonus, guarded := computeAward(r, tt.expRate, cfg)

			if personal != tt.wantPersonal {
				t.Errorf("personal = %d, want %d", personal, tt.wantPersonal)
			}
			if bonus != tt.wantBonus {
				t.Errorf("bonus = %d, want %d", bonus, tt.wantBonus)
			}
			if guarded != tt.wantGuarded {
				t.Errorf("guarded = %v, want %v", guarded, tt.wantGuarded)
			}
		})
	}
}

func TestComputeAward_NonFiniteIsGuarded(t *testing.T) {
	cfg := DefaultExperienceConfig()

	tests := []struct {
		name    string
		r       Recipient
		expRate float64
	}{
		{
			name: "PooledExp NaN",
			r: Recipient{
				Level:           50,
				PooledExp:       math.NaN(),
				TotalPartyLevel: 50,
				IsMvp:           true,
			},
			expRate: 1.0,
		},
		{
			name: "PooledExp Inf",
			r: Recipient{
				Level:           50,
				PooledExp:       math.Inf(1),
				TotalPartyLevel: 50,
				IsMvp:           true,
			},
			expRate: 1.0,
		},
		{
			name: "expRate NaN",
			r: Recipient{
				Level:           50,
				PooledExp:       1000,
				TotalPartyLevel: 50,
				IsMvp:           true,
			},
			expRate: math.NaN(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			personal, bonus, guarded := computeAward(tt.r, tt.expRate, cfg)
			if personal != 0 || bonus != 0 || !guarded {
				t.Errorf("computeAward() = (%d, %d, %v), want (0, 0, true)", personal, bonus, guarded)
			}
		})
	}
}

func TestLevelGateHintText(t *testing.T) {
	got := levelGateHintText("Blue Snail", 2)
	want := "You have gained #rno experience#k from defeating #e#bBlue Snail#k#n (lv. #b2#k)! Take note you must have around the same level as the mob to start earning EXP from it."
	if got != want {
		t.Errorf("levelGateHintText() = %q, want %q", got, want)
	}
}
