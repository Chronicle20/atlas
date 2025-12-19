package monster

type DamageEntryModel struct {
	characterId uint32
	damage      uint32
}

func NewDamageEntryModel(characterId uint32, damage uint32) DamageEntryModel {
	return DamageEntryModel{
		characterId: characterId,
		damage:      damage,
	}
}

type DamageDistributionModel struct {
	solo                   map[uint32]uint32
	party                  map[uint32]map[uint32]uint32
	personalRatio          map[uint32]float64
	experiencePerDamage    float64
	standardDeviationRatio float64
}

func (d DamageDistributionModel) Solo() map[uint32]uint32 {
	return d.solo
}

func (d DamageDistributionModel) ExperiencePerDamage() float64 {
	return d.experiencePerDamage
}

func (d DamageDistributionModel) PersonalRatio() map[uint32]float64 {
	return d.personalRatio
}

func (d DamageDistributionModel) StandardDeviationRatio() float64 {
	return d.standardDeviationRatio
}
