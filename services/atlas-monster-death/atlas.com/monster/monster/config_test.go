package monster

import "testing"

func TestDefaultExperienceConfig(t *testing.T) {
	c := DefaultExperienceConfig()

	if c.EnforceMobLevelRange != true {
		t.Errorf("EnforceMobLevelRange = %v, want true", c.EnforceMobLevelRange)
	}
	if c.LevelInterval != 5 {
		t.Errorf("LevelInterval = %v, want 5", c.LevelInterval)
	}
	if c.LeachInterval != 5 {
		t.Errorf("LeachInterval = %v, want 5", c.LeachInterval)
	}
	if c.SplitCommonMod != 0.8 {
		t.Errorf("SplitCommonMod = %v, want 0.8", c.SplitCommonMod)
	}
	if c.MvpMod != 0.2 {
		t.Errorf("MvpMod = %v, want 0.2", c.MvpMod)
	}
	if c.PartyBonusPerMember != 0.05 {
		t.Errorf("PartyBonusPerMember = %v, want 0.05", c.PartyBonusPerMember)
	}
}

func TestLoadExperienceConfig(t *testing.T) {
	tests := []struct {
		name    string
		setEnv  func(t *testing.T)
		wantErr func(t *testing.T, c ExperienceConfig)
	}{
		{
			name: "no env is defaults",
			setEnv: func(t *testing.T) {
				t.Setenv(EnvEnforceMobLevelRange, "")
				t.Setenv(EnvLevelInterval, "")
				t.Setenv(EnvLeachInterval, "")
			},
			wantErr: func(t *testing.T, c ExperienceConfig) {
				if c != DefaultExperienceConfig() {
					t.Errorf("got %+v, want defaults %+v", c, DefaultExperienceConfig())
				}
			},
		},
		{
			name: "gate disabled",
			setEnv: func(t *testing.T) {
				t.Setenv(EnvEnforceMobLevelRange, "false")
			},
			wantErr: func(t *testing.T, c ExperienceConfig) {
				if c.EnforceMobLevelRange != false {
					t.Errorf("EnforceMobLevelRange = %v, want false", c.EnforceMobLevelRange)
				}
				if c.LevelInterval != 5 {
					t.Errorf("LevelInterval = %v, want 5", c.LevelInterval)
				}
				if c.LeachInterval != 5 {
					t.Errorf("LeachInterval = %v, want 5", c.LeachInterval)
				}
				if c.SplitCommonMod != 0.8 {
					t.Errorf("SplitCommonMod = %v, want 0.8", c.SplitCommonMod)
				}
				if c.MvpMod != 0.2 {
					t.Errorf("MvpMod = %v, want 0.2", c.MvpMod)
				}
				if c.PartyBonusPerMember != 0.05 {
					t.Errorf("PartyBonusPerMember = %v, want 0.05", c.PartyBonusPerMember)
				}
			},
		},
		{
			name: "gate explicitly enabled",
			setEnv: func(t *testing.T) {
				t.Setenv(EnvEnforceMobLevelRange, "true")
			},
			wantErr: func(t *testing.T, c ExperienceConfig) {
				if c.EnforceMobLevelRange != true {
					t.Errorf("EnforceMobLevelRange = %v, want true", c.EnforceMobLevelRange)
				}
			},
		},
		{
			name: "intervals overridden",
			setEnv: func(t *testing.T) {
				t.Setenv(EnvLevelInterval, "10")
				t.Setenv(EnvLeachInterval, "3")
			},
			wantErr: func(t *testing.T, c ExperienceConfig) {
				if c.LevelInterval != 10 {
					t.Errorf("LevelInterval = %v, want 10", c.LevelInterval)
				}
				if c.LeachInterval != 3 {
					t.Errorf("LeachInterval = %v, want 3", c.LeachInterval)
				}
			},
		},
		{
			name: "unparseable bool falls back",
			setEnv: func(t *testing.T) {
				t.Setenv(EnvEnforceMobLevelRange, "maybe")
			},
			wantErr: func(t *testing.T, c ExperienceConfig) {
				if c.EnforceMobLevelRange != true {
					t.Errorf("EnforceMobLevelRange = %v, want true (fallback)", c.EnforceMobLevelRange)
				}
			},
		},
		{
			name: "unparseable interval falls back",
			setEnv: func(t *testing.T) {
				t.Setenv(EnvLevelInterval, "abc")
			},
			wantErr: func(t *testing.T, c ExperienceConfig) {
				if c.LevelInterval != 5 {
					t.Errorf("LevelInterval = %v, want 5 (fallback)", c.LevelInterval)
				}
			},
		},
		{
			name: "zero interval is honoured",
			setEnv: func(t *testing.T) {
				t.Setenv(EnvLevelInterval, "0")
			},
			wantErr: func(t *testing.T, c ExperienceConfig) {
				if c.LevelInterval != 0 {
					t.Errorf("LevelInterval = %v, want 0", c.LevelInterval)
				}
			},
		},
		{
			name: "balance constants are never env-driven",
			setEnv: func(t *testing.T) {
				t.Setenv("SPLIT_COMMON_MOD", "9")
				t.Setenv("MVP_MOD", "9")
				t.Setenv("PARTY_BONUS_PER_MEMBER", "9")
			},
			wantErr: func(t *testing.T, c ExperienceConfig) {
				if c.SplitCommonMod != 0.8 {
					t.Errorf("SplitCommonMod = %v, want 0.8", c.SplitCommonMod)
				}
				if c.MvpMod != 0.2 {
					t.Errorf("MvpMod = %v, want 0.2", c.MvpMod)
				}
				if c.PartyBonusPerMember != 0.05 {
					t.Errorf("PartyBonusPerMember = %v, want 0.05", c.PartyBonusPerMember)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setEnv(t)
			c := LoadExperienceConfig()
			tt.wantErr(t, c)
		})
	}
}
