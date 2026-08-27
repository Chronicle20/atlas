package monster

import (
	"os"
	"strconv"
)

const (
	EnvEnforceMobLevelRange = "USE_ENFORCE_MOB_LEVEL_RANGE"
	EnvLevelInterval        = "LEVEL_INTERVAL"
	EnvLeachInterval        = "LEACH_INTERVAL"
)

// ExperienceConfig carries the EXP-distribution tunables. The three gate
// settings mirror <cosmic>/config.yaml:243 and :293-294 and are overridable
// per deployment. The three split mods are game-balance constants with no env
// key on purpose: an env-tunable split would let two replicas of the same
// world award different EXP for the same kill.
type ExperienceConfig struct {
	EnforceMobLevelRange bool
	LevelInterval        uint32
	LeachInterval        uint32
	SplitCommonMod       float64
	MvpMod               float64
	PartyBonusPerMember  float64
}

func DefaultExperienceConfig() ExperienceConfig {
	return ExperienceConfig{
		EnforceMobLevelRange: true,
		LevelInterval:        5,
		LeachInterval:        5,
		SplitCommonMod:       0.8,
		MvpMod:               0.2,
		PartyBonusPerMember:  0.05,
	}
}

// LoadExperienceConfig layers env overrides over the defaults. An absent or
// unparseable key keeps the default rather than failing the service: EXP
// tuning must never be able to prevent a replica from starting.
func LoadExperienceConfig() ExperienceConfig {
	c := DefaultExperienceConfig()

	if v, ok := os.LookupEnv(EnvEnforceMobLevelRange); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			c.EnforceMobLevelRange = b
		}
	}

	if v, ok := os.LookupEnv(EnvLevelInterval); ok {
		if i, err := strconv.ParseUint(v, 10, 32); err == nil {
			c.LevelInterval = uint32(i)
		}
	}

	if v, ok := os.LookupEnv(EnvLeachInterval); ok {
		if i, err := strconv.ParseUint(v, 10, 32); err == nil {
			c.LeachInterval = uint32(i)
		}
	}

	return c
}
