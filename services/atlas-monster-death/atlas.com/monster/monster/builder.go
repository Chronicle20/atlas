package monster

import "errors"

type DamageDistributionBuilder struct {
	solo                   map[uint32]uint32
	party                  map[uint32]map[uint32]uint32
	personalRatio          map[uint32]float64
	experiencePerDamage    float64
	standardDeviationRatio float64
}

func NewDamageDistributionBuilder() *DamageDistributionBuilder {
	return &DamageDistributionBuilder{
		solo:          make(map[uint32]uint32),
		party:         make(map[uint32]map[uint32]uint32),
		personalRatio: make(map[uint32]float64),
	}
}

func (b *DamageDistributionBuilder) SetSolo(solo map[uint32]uint32) *DamageDistributionBuilder {
	b.solo = solo
	return b
}

func (b *DamageDistributionBuilder) SetParty(party map[uint32]map[uint32]uint32) *DamageDistributionBuilder {
	b.party = party
	return b
}

func (b *DamageDistributionBuilder) SetPersonalRatio(personalRatio map[uint32]float64) *DamageDistributionBuilder {
	b.personalRatio = personalRatio
	return b
}

func (b *DamageDistributionBuilder) SetExperiencePerDamage(experiencePerDamage float64) *DamageDistributionBuilder {
	b.experiencePerDamage = experiencePerDamage
	return b
}

func (b *DamageDistributionBuilder) SetStandardDeviationRatio(standardDeviationRatio float64) *DamageDistributionBuilder {
	b.standardDeviationRatio = standardDeviationRatio
	return b
}

func (b *DamageDistributionBuilder) Build() (DamageDistributionModel, error) {
	if b.solo == nil {
		return DamageDistributionModel{}, errors.New("solo map cannot be nil")
	}
	if b.party == nil {
		return DamageDistributionModel{}, errors.New("party map cannot be nil")
	}
	if b.personalRatio == nil {
		return DamageDistributionModel{}, errors.New("personalRatio map cannot be nil")
	}
	return DamageDistributionModel{
		solo:                   b.solo,
		party:                  b.party,
		personalRatio:          b.personalRatio,
		experiencePerDamage:    b.experiencePerDamage,
		standardDeviationRatio: b.standardDeviationRatio,
	}, nil
}
