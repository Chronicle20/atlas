package monster

import (
	"fmt"
	"math"
)

// DamageInput is one aggregated damage entry: exactly one per damaging
// character, in-field or not.
type DamageInput struct {
	CharacterId uint32
	Damage      uint32
}

// SoloInput is an in-field damager with no party.
type SoloInput struct {
	CharacterId uint32
	Level       byte
}

// MemberInput is one co-located party member. Membership here already means
// "in the field where the kill happened" -- co-location comes from the
// atlas-maps character list, never from the party service's member location
// fields, which are only eventually consistent (D11).
type MemberInput struct {
	CharacterId uint32
	Level       byte
}

// PartyInput is one participating party and its co-located members, whether or
// not they dealt damage.
type PartyInput struct {
	PartyId uint32
	Members []MemberInput
}

// ExperienceInput is everything the planner needs. Damages carries EVERY
// damager including those who left the field, because out-of-field damagers
// still count toward totalDamage and totalEntries even though they are never
// party-resolved and receive nothing (D12).
type ExperienceInput struct {
	MonsterExperience uint32
	MonsterLevel      uint32
	Damages           []DamageInput
	Solos             []SoloInput
	Parties           []PartyInput
}

// Recipient is one character who will receive an award. PooledExp is the
// party's whole participation EXP, not this member's share -- the split is
// applied by computeAward.
type Recipient struct {
	CharacterId     uint32
	Level           byte
	PartyId         uint32
	PooledExp       float64
	TotalPartyLevel uint32
	PartyBonusMod   float64
	IsMvp           bool
	White           bool
}

// Exclusion is a co-located party member the level gate kept out.
type Exclusion struct {
	CharacterId uint32
}

type ExperiencePlan struct {
	Recipients             []Recipient
	Exclusions             []Exclusion
	TotalDamage            uint32
	TotalEntries           int
	ExperiencePerDamage    float64
	StandardDeviationRatio float64
}

// computeAward applies the character's EXP rate to the pooled figure BEFORE
// the split, which is the existing ordering and is what makes FR-8.2 (party
// bonus is rate-multiplied too) hold without a second multiplication.
//
// guarded reports that a value was not representable and was replaced: a
// non-finite intermediate becomes 0, and a value at or above MaxUint32 is
// clamped. uint32(NaN) is implementation-defined in Go and must never reach
// the wire (FR-8.6).
func computeAward(r Recipient, expRate float64, cfg ExperienceConfig) (uint32, uint32, bool) {
	if r.TotalPartyLevel == 0 {
		return 0, 0, true
	}
	exp := r.PooledExp * expRate
	share := cfg.SplitCommonMod * float64(r.Level) / float64(r.TotalPartyLevel)
	if r.IsMvp {
		share += cfg.MvpMod
	}
	personalF := share * exp
	bonusF := r.PartyBonusMod * personalF

	personal, pg := toAwardAmount(personalF)
	bonus, bg := toAwardAmount(bonusF)
	return personal, bonus, pg || bg
}

// toAwardAmount converts a computed EXP figure to the uint32 the award command
// carries, reporting whether the value had to be replaced.
func toAwardAmount(v float64) (uint32, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, true
	}
	if v < 0 {
		return 0, true
	}
	if v >= math.MaxUint32 {
		return math.MaxUint32, true
	}
	return uint32(v), false
}

// levelGateHintText renders Cosmic's level-gate notice (Character.java:9246).
func levelGateHintText(name string, level uint32) string {
	return fmt.Sprintf("You have gained #rno experience#k from defeating #e#b%s#k#n (lv. #b%d#k)! Take note you must have around the same level as the mob to start earning EXP from it.", name, level)
}
