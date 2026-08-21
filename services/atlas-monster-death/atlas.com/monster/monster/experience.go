package monster

import (
	"fmt"
	"math"
	"sort"
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

// planDistribution is the pure party-EXP planner. It takes no logger, no
// context, no clock, and no collaborator: every rule from FR-5/FR-6 lives
// here and is exercised by a mock-free table test.
//
// Gate ordering is fixed: the level gate selects recipients only.
// partyDamage, participationExp, the contributor list, and the interval set
// itself are all computed BEFORE the gate, from the ungated contributor
// list. A gated-out member who dealt damage still widens the interval for
// everyone else and still contributes to a pool they do not share in --
// that is Cosmic's behaviour (Monster.java:549-600) and is what makes the
// interval union meaningful (D14).
func planDistribution(in ExperienceInput, cfg ExperienceConfig) ExperiencePlan {
	damages := make([]DamageInput, len(in.Damages))
	copy(damages, in.Damages)
	sort.Slice(damages, func(i, j int) bool { return damages[i].CharacterId < damages[j].CharacterId })

	solos := make([]SoloInput, len(in.Solos))
	copy(solos, in.Solos)
	sort.Slice(solos, func(i, j int) bool { return solos[i].CharacterId < solos[j].CharacterId })

	parties := make([]PartyInput, len(in.Parties))
	copy(parties, in.Parties)
	for i := range parties {
		members := make([]MemberInput, len(parties[i].Members))
		copy(members, parties[i].Members)
		sort.Slice(members, func(a, b int) bool { return members[a].CharacterId < members[b].CharacterId })
		parties[i].Members = members
	}
	sort.Slice(parties, func(i, j int) bool { return parties[i].PartyId < parties[j].PartyId })

	damageOf := make(map[uint32]uint32, len(damages))
	var totalDamage uint32
	for _, d := range damages {
		damageOf[d.CharacterId] = d.Damage
		totalDamage += d.Damage
	}

	if totalDamage == 0 {
		return ExperiencePlan{
			TotalDamage:  0,
			TotalEntries: len(damages),
			Recipients:   nil,
			Exclusions:   nil,
		}
	}

	epd := float64(in.MonsterExperience) / float64(totalDamage)

	inFieldDamagers := 0
	for _, s := range solos {
		if _, ok := damageOf[s.CharacterId]; ok {
			inFieldDamagers++
		}
	}
	for _, p := range parties {
		for _, m := range p.Members {
			if d, ok := damageOf[m.CharacterId]; ok && d > 0 {
				inFieldDamagers++
			}
		}
	}
	outOfField := len(damages) - inFieldDamagers
	totalEntries := len(solos) + len(parties) + outOfField

	personalRatio := make(map[uint32]float64)
	entryRatios := make([]float64, 0, len(solos)+len(parties))
	for _, s := range solos {
		ratio := float64(damageOf[s.CharacterId]) / float64(totalDamage)
		personalRatio[s.CharacterId] = ratio
		entryRatios = append(entryRatios, ratio)
	}
	for _, p := range parties {
		partyRatio := 0.0
		for _, m := range p.Members {
			if d, ok := damageOf[m.CharacterId]; ok && d > 0 {
				ratio := float64(d) / float64(totalDamage)
				personalRatio[m.CharacterId] += ratio
				partyRatio += ratio
			}
		}
		entryRatios = append(entryRatios, partyRatio)
	}

	sort.Float64s(entryRatios)
	stdr := calculateExperienceStandardDeviationThreshold(entryRatios, totalEntries)

	var recipients []Recipient
	var exclusions []Exclusion

	for _, s := range solos {
		if s.Level == 0 {
			continue
		}
		recipients = append(recipients, Recipient{
			CharacterId:     s.CharacterId,
			Level:           s.Level,
			PartyId:         0,
			PooledExp:       float64(damageOf[s.CharacterId]) * epd,
			TotalPartyLevel: uint32(s.Level),
			PartyBonusMod:   0,
			IsMvp:           true,
			White:           isWhiteExperienceGain(s.CharacterId, personalRatio, stdr),
		})
	}

	for _, p := range parties {
		var contributors []MemberInput
		var partyDamage uint32
		for _, m := range p.Members {
			if d, ok := damageOf[m.CharacterId]; ok && d > 0 {
				contributors = append(contributors, m)
				partyDamage += d
			}
		}
		participationExp := float64(partyDamage) * epd

		var expMembers []MemberInput
		var excluded []MemberInput
		if cfg.EnforceMobLevelRange {
			var s intervalSet
			s.add(int(in.MonsterLevel)-int(cfg.LevelInterval), int(in.MonsterLevel)+int(cfg.LevelInterval))
			for _, c := range contributors {
				s.add(int(c.Level)-int(cfg.LeachInterval), int(c.Level)+int(cfg.LeachInterval))
			}
			built := s.build()
			for _, m := range p.Members {
				if built.contains(int(m.Level)) {
					expMembers = append(expMembers, m)
				} else {
					excluded = append(excluded, m)
				}
			}
		} else {
			expMembers = p.Members
		}

		for _, m := range excluded {
			exclusions = append(exclusions, Exclusion{CharacterId: m.CharacterId})
		}

		if len(expMembers) == 0 {
			continue
		}

		var totalPartyLevel uint32
		for _, m := range expMembers {
			totalPartyLevel += uint32(m.Level)
		}
		if totalPartyLevel == 0 {
			continue
		}

		var mvpId uint32
		var mvpDamage uint32
		first := true
		for _, m := range expMembers {
			d := damageOf[m.CharacterId]
			if first || d > mvpDamage {
				mvpId = m.CharacterId
				mvpDamage = d
				first = false
			}
		}

		hasPartySharers := len(expMembers) > 1
		partyBonusMod := 0.0
		if hasPartySharers {
			partyBonusMod = cfg.PartyBonusPerMember * float64(len(expMembers))
		}

		for _, m := range expMembers {
			recipients = append(recipients, Recipient{
				CharacterId:     m.CharacterId,
				Level:           m.Level,
				PartyId:         p.PartyId,
				PooledExp:       participationExp,
				TotalPartyLevel: totalPartyLevel,
				PartyBonusMod:   partyBonusMod,
				IsMvp:           m.CharacterId == mvpId,
				White:           isWhiteExperienceGain(m.CharacterId, personalRatio, stdr),
			})
		}
	}

	sort.Slice(recipients, func(i, j int) bool { return recipients[i].CharacterId < recipients[j].CharacterId })
	sort.Slice(exclusions, func(i, j int) bool { return exclusions[i].CharacterId < exclusions[j].CharacterId })

	return ExperiencePlan{
		Recipients:             recipients,
		Exclusions:             exclusions,
		TotalDamage:            totalDamage,
		TotalEntries:           totalEntries,
		ExperiencePerDamage:    epd,
		StandardDeviationRatio: stdr,
	}
}
