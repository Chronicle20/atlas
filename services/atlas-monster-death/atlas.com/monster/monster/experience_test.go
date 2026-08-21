package monster

import (
	"math"
	"reflect"
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

// partyBonusMod computes cfg.PartyBonusPerMember * n at runtime rather than
// as a constant expression, so the expected value has the same float64
// rounding as the implementation's runtime multiplication (e.g. 0.05*3 !=
// the constant-folded 0.15).
func partyBonusMod(n float64) float64 {
	return DefaultExperienceConfig().PartyBonusPerMember * n
}

func TestPlanDistribution(t *testing.T) {
	tests := []struct {
		name           string
		input          ExperienceInput
		cfg            func() ExperienceConfig
		wantTotalDmg   uint32
		wantEntries    int
		wantRecipients []Recipient
		wantExclusions []Exclusion
	}{
		{
			name: "solo single damager",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      100,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 500}},
				Solos:             []SoloInput{{CharacterId: 1, Level: 50}},
			},
			wantTotalDmg: 500,
			wantEntries:  1,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 50, PartyId: 0, PooledExp: 1000, TotalPartyLevel: 50, PartyBonusMod: 0, IsMvp: true, White: true},
			},
		},
		{
			name: "two solo damagers",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      100,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 750}, {CharacterId: 2, Damage: 250}},
				Solos:             []SoloInput{{CharacterId: 1, Level: 50}, {CharacterId: 2, Level: 50}},
			},
			wantTotalDmg: 1000,
			wantEntries:  2,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 50, PartyId: 0, PooledExp: 750, TotalPartyLevel: 50, PartyBonusMod: 0, IsMvp: true, White: true},
				{CharacterId: 2, Level: 50, PartyId: 0, PooledExp: 250, TotalPartyLevel: 50, PartyBonusMod: 0, IsMvp: true, White: false},
			},
		},
		{
			name: "zero total damage",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      100,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 0}},
				Solos:             []SoloInput{{CharacterId: 1, Level: 50}},
			},
			wantTotalDmg: 0,
			wantEntries:  1,
		},
		{
			name: "empty input",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      100,
			},
			wantTotalDmg: 0,
			wantEntries:  0,
		},
		{
			name: "two-member party, one damager (FR-5.3)",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      100,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 1000}},
				Parties:           []PartyInput{{PartyId: 9, Members: []MemberInput{{CharacterId: 1, Level: 50}, {CharacterId: 2, Level: 50}}}},
			},
			wantTotalDmg: 1000,
			wantEntries:  1,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 50, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 100, PartyBonusMod: 0.10, IsMvp: true, White: true},
				{CharacterId: 2, Level: 50, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 100, PartyBonusMod: 0.10, IsMvp: false, White: false},
			},
		},
		{
			name: "one-member party equals solo (FR-5.11)",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      100,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 1000}},
				Parties:           []PartyInput{{PartyId: 9, Members: []MemberInput{{CharacterId: 1, Level: 50}}}},
			},
			wantTotalDmg: 1000,
			wantEntries:  1,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 50, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 50, PartyBonusMod: 0.0, IsMvp: true, White: true},
			},
		},
		{
			name: "four-member party bonus (FR-5.8)",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      100,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 1000}},
				Parties: []PartyInput{{PartyId: 9, Members: []MemberInput{
					{CharacterId: 1, Level: 50}, {CharacterId: 2, Level: 50}, {CharacterId: 3, Level: 50}, {CharacterId: 4, Level: 50},
				}}},
			},
			wantTotalDmg: 1000,
			wantEntries:  1,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 50, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 200, PartyBonusMod: 0.20, IsMvp: true, White: true},
				{CharacterId: 2, Level: 50, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 200, PartyBonusMod: 0.20, IsMvp: false, White: false},
				{CharacterId: 3, Level: 50, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 200, PartyBonusMod: 0.20, IsMvp: false, White: false},
				{CharacterId: 4, Level: 50, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 200, PartyBonusMod: 0.20, IsMvp: false, White: false},
			},
		},
		{
			name: "MVP is highest damager in expMembers",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      100,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 100}, {CharacterId: 2, Damage: 900}},
				Parties:           []PartyInput{{PartyId: 9, Members: []MemberInput{{CharacterId: 1, Level: 50}, {CharacterId: 2, Level: 50}}}},
			},
			wantTotalDmg: 1000,
			wantEntries:  1,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 50, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 100, PartyBonusMod: 0.10, IsMvp: false, White: false},
				{CharacterId: 2, Level: 50, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 100, PartyBonusMod: 0.10, IsMvp: true, White: false},
			},
		},
		{
			name: "MVP tie breaks to lowest characterId (D13)",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      100,
				Damages:           []DamageInput{{CharacterId: 2, Damage: 500}, {CharacterId: 1, Damage: 500}},
				Parties:           []PartyInput{{PartyId: 9, Members: []MemberInput{{CharacterId: 1, Level: 50}, {CharacterId: 2, Level: 50}}}},
			},
			wantTotalDmg: 1000,
			wantEntries:  1,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 50, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 100, PartyBonusMod: 0.10, IsMvp: true, White: false},
				{CharacterId: 2, Level: 50, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 100, PartyBonusMod: 0.10, IsMvp: false, White: false},
			},
		},
		{
			name: "MVP falls to a non-damager when no member damaged",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      50,
				Damages:           []DamageInput{{CharacterId: 7, Damage: 1000}},
				Parties:           []PartyInput{{PartyId: 9, Members: []MemberInput{{CharacterId: 1, Level: 50}, {CharacterId: 2, Level: 50}}}},
			},
			wantTotalDmg: 1000,
			wantEntries:  2,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 50, PartyId: 9, PooledExp: 0, TotalPartyLevel: 100, PartyBonusMod: 0.10, IsMvp: true, White: false},
				{CharacterId: 2, Level: 50, PartyId: 9, PooledExp: 0, TotalPartyLevel: 100, PartyBonusMod: 0.10, IsMvp: false, White: false},
			},
		},
		{
			name: "out-of-field damager counts but receives nothing (D12)",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      100,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 500}, {CharacterId: 7, Damage: 500}},
				Solos:             []SoloInput{{CharacterId: 1, Level: 50}},
			},
			wantTotalDmg: 1000,
			wantEntries:  2,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 50, PartyId: 0, PooledExp: 500, TotalPartyLevel: 50, PartyBonusMod: 0, IsMvp: true, White: true},
			},
		},
		{
			name: "level gate excludes and does not count (FR-6.1)",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      125,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 1000}},
				Parties: []PartyInput{{PartyId: 9, Members: []MemberInput{
					{CharacterId: 1, Level: 120}, {CharacterId: 2, Level: 32}, {CharacterId: 3, Level: 70},
				}}},
			},
			wantTotalDmg: 1000,
			wantEntries:  1,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 120, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 120, PartyBonusMod: 0.0, IsMvp: true, White: true},
			},
			wantExclusions: []Exclusion{{CharacterId: 2}, {CharacterId: 3}},
		},
		{
			name: "interval union admits and rejects (FR-6.2)",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      125,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 600}, {CharacterId: 2, Damage: 400}},
				Parties: []PartyInput{{PartyId: 9, Members: []MemberInput{
					{CharacterId: 1, Level: 120}, {CharacterId: 2, Level: 30}, {CharacterId: 3, Level: 32}, {CharacterId: 4, Level: 70},
				}}},
			},
			wantTotalDmg: 1000,
			wantEntries:  1,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 120, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 182, PartyBonusMod: partyBonusMod(3), IsMvp: true, White: false},
				{CharacterId: 2, Level: 30, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 182, PartyBonusMod: partyBonusMod(3), IsMvp: false, White: false},
				{CharacterId: 3, Level: 32, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 182, PartyBonusMod: partyBonusMod(3), IsMvp: false, White: false},
			},
			wantExclusions: []Exclusion{{CharacterId: 4}},
		},
		{
			name: "gate disabled admits everyone",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      125,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 600}, {CharacterId: 2, Damage: 400}},
				Parties: []PartyInput{{PartyId: 9, Members: []MemberInput{
					{CharacterId: 1, Level: 120}, {CharacterId: 2, Level: 30}, {CharacterId: 3, Level: 32}, {CharacterId: 4, Level: 70},
				}}},
			},
			cfg: func() ExperienceConfig {
				c := DefaultExperienceConfig()
				c.EnforceMobLevelRange = false
				return c
			},
			wantTotalDmg: 1000,
			wantEntries:  1,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 120, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 252, PartyBonusMod: 0.20, IsMvp: true, White: false},
				{CharacterId: 2, Level: 30, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 252, PartyBonusMod: 0.20, IsMvp: false, White: false},
				{CharacterId: 3, Level: 32, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 252, PartyBonusMod: 0.20, IsMvp: false, White: false},
				{CharacterId: 4, Level: 70, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 252, PartyBonusMod: 0.20, IsMvp: false, White: false},
			},
		},
		{
			name: "gate never applies to solo (FR-6.3)",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      125,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 1000}},
				Solos:             []SoloInput{{CharacterId: 1, Level: 5}},
			},
			wantTotalDmg: 1000,
			wantEntries:  1,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 5, PartyId: 0, PooledExp: 1000, TotalPartyLevel: 5, PartyBonusMod: 0, IsMvp: true, White: true},
			},
		},
		{
			name: "a contributor's band widens the set and their damage feeds the pool (D14)",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      200,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 500}, {CharacterId: 2, Damage: 500}},
				Parties: []PartyInput{{PartyId: 9, Members: []MemberInput{
					{CharacterId: 1, Level: 30}, {CharacterId: 2, Level: 199}, {CharacterId: 3, Level: 32}, {CharacterId: 4, Level: 100},
				}}},
			},
			wantTotalDmg: 1000,
			wantEntries:  1,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 30, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 261, PartyBonusMod: partyBonusMod(3), IsMvp: true, White: false},
				{CharacterId: 2, Level: 199, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 261, PartyBonusMod: partyBonusMod(3), IsMvp: false, White: false},
				{CharacterId: 3, Level: 32, PartyId: 9, PooledExp: 1000, TotalPartyLevel: 261, PartyBonusMod: partyBonusMod(3), IsMvp: false, White: false},
			},
			wantExclusions: []Exclusion{{CharacterId: 4}},
		},
		{
			name: "party with no eligible members is skipped (FR-5.10)",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      200,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 1000}},
				Parties:           []PartyInput{{PartyId: 9, Members: []MemberInput{{CharacterId: 5, Level: 10}}}},
			},
			wantTotalDmg:   1000,
			wantEntries:    2,
			wantExclusions: []Exclusion{{CharacterId: 5}},
		},
		{
			name: "zero totalPartyLevel yields no recipients (FR-5.6)",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      100,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 1000}},
				Parties:           []PartyInput{{PartyId: 9, Members: []MemberInput{{CharacterId: 1, Level: 0}}}},
			},
			cfg: func() ExperienceConfig {
				c := DefaultExperienceConfig()
				c.EnforceMobLevelRange = false
				return c
			},
			wantTotalDmg: 1000,
			wantEntries:  1,
		},
		{
			name: "mixed solo and party",
			input: ExperienceInput{
				MonsterExperience: 1000,
				MonsterLevel:      100,
				Damages:           []DamageInput{{CharacterId: 1, Damage: 600}, {CharacterId: 2, Damage: 400}},
				Solos:             []SoloInput{{CharacterId: 1, Level: 50}},
				Parties:           []PartyInput{{PartyId: 9, Members: []MemberInput{{CharacterId: 2, Level: 50}, {CharacterId: 3, Level: 50}}}},
			},
			wantTotalDmg: 1000,
			wantEntries:  2,
			wantRecipients: []Recipient{
				{CharacterId: 1, Level: 50, PartyId: 0, PooledExp: 600, TotalPartyLevel: 50, PartyBonusMod: 0, IsMvp: true, White: true},
				{CharacterId: 2, Level: 50, PartyId: 9, PooledExp: 400, TotalPartyLevel: 100, PartyBonusMod: 0.10, IsMvp: true, White: false},
				{CharacterId: 3, Level: 50, PartyId: 9, PooledExp: 400, TotalPartyLevel: 100, PartyBonusMod: 0.10, IsMvp: false, White: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultExperienceConfig()
			if tt.cfg != nil {
				cfg = tt.cfg()
			}

			got := planDistribution(tt.input, cfg)

			if got.TotalDamage != tt.wantTotalDmg {
				t.Errorf("TotalDamage = %d, want %d", got.TotalDamage, tt.wantTotalDmg)
			}
			if got.TotalEntries != tt.wantEntries {
				t.Errorf("TotalEntries = %d, want %d", got.TotalEntries, tt.wantEntries)
			}
			if !reflect.DeepEqual(got.Recipients, tt.wantRecipients) {
				t.Errorf("Recipients = %+v, want %+v", got.Recipients, tt.wantRecipients)
			}
			if !reflect.DeepEqual(got.Exclusions, tt.wantExclusions) {
				t.Errorf("Exclusions = %+v, want %+v", got.Exclusions, tt.wantExclusions)
			}
		})
	}
}

func TestPlanDistribution_PartiesOrderedByPartyIdMembersByCharacterId(t *testing.T) {
	in := ExperienceInput{
		MonsterExperience: 1000,
		MonsterLevel:      100,
		Damages:           []DamageInput{{CharacterId: 4, Damage: 100}, {CharacterId: 2, Damage: 100}},
		Parties: []PartyInput{
			{PartyId: 9, Members: []MemberInput{{CharacterId: 4, Level: 50}, {CharacterId: 3, Level: 50}}},
			{PartyId: 2, Members: []MemberInput{{CharacterId: 2, Level: 50}, {CharacterId: 1, Level: 50}}},
		},
	}

	got := planDistribution(in, DefaultExperienceConfig())

	var ids []uint32
	for _, r := range got.Recipients {
		ids = append(ids, r.CharacterId)
	}
	want := []uint32{1, 2, 3, 4}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("recipient CharacterId order = %v, want %v", ids, want)
	}
}

func TestPlanDistribution_IsDeterministicUnderShuffledInput(t *testing.T) {
	cfg := DefaultExperienceConfig()
	in := ExperienceInput{
		MonsterExperience: 1000,
		MonsterLevel:      100,
		Damages:           []DamageInput{{CharacterId: 1, Damage: 600}, {CharacterId: 2, Damage: 400}},
		Solos:             []SoloInput{{CharacterId: 1, Level: 50}},
		Parties:           []PartyInput{{PartyId: 9, Members: []MemberInput{{CharacterId: 2, Level: 50}, {CharacterId: 3, Level: 50}}}},
	}

	want := planDistribution(in, cfg)

	for i := 0; i < 20; i++ {
		shuffled := ExperienceInput{
			MonsterExperience: in.MonsterExperience,
			MonsterLevel:      in.MonsterLevel,
			Damages:           reverseDamages(in.Damages),
			Solos:             reverseSolos(in.Solos),
			Parties:           reverseParties(in.Parties),
		}

		got := planDistribution(shuffled, cfg)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("iteration %d: planDistribution(shuffled) = %+v, want %+v", i, got, want)
		}
	}
}

func reverseDamages(in []DamageInput) []DamageInput {
	out := make([]DamageInput, len(in))
	for i, v := range in {
		out[len(in)-1-i] = v
	}
	return out
}

func reverseSolos(in []SoloInput) []SoloInput {
	out := make([]SoloInput, len(in))
	for i, v := range in {
		out[len(in)-1-i] = v
	}
	return out
}

func reverseParties(in []PartyInput) []PartyInput {
	out := make([]PartyInput, len(in))
	for i, v := range in {
		out[len(in)-1-i] = PartyInput{
			PartyId: v.PartyId,
			Members: reverseMembers(v.Members),
		}
	}
	return out
}

func reverseMembers(in []MemberInput) []MemberInput {
	out := make([]MemberInput, len(in))
	for i, v := range in {
		out[len(in)-1-i] = v
	}
	return out
}

func TestPlanDistribution_TotalEntriesComposition(t *testing.T) {
	in := ExperienceInput{
		MonsterExperience: 1000,
		MonsterLevel:      100,
		Damages: []DamageInput{
			{CharacterId: 1, Damage: 100},
			{CharacterId: 2, Damage: 100},
			{CharacterId: 3, Damage: 100},
			{CharacterId: 4, Damage: 100},
			{CharacterId: 5, Damage: 100},
			{CharacterId: 6, Damage: 100},
			{CharacterId: 7, Damage: 100},
		},
		Solos: []SoloInput{{CharacterId: 1, Level: 50}, {CharacterId: 2, Level: 50}},
		Parties: []PartyInput{
			{PartyId: 9, Members: []MemberInput{{CharacterId: 3, Level: 50}}},
			{PartyId: 10, Members: []MemberInput{{CharacterId: 4, Level: 50}}},
		},
	}

	got := planDistribution(in, DefaultExperienceConfig())

	if got.TotalEntries != 7 {
		t.Errorf("TotalEntries = %d, want 7", got.TotalEntries)
	}
}
