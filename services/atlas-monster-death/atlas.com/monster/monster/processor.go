package monster

import (
	"atlas-monster-death/character"
	_map "atlas-monster-death/map"
	"atlas-monster-death/monster/drop"
	"atlas-monster-death/monster/information"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"math"
	"math/rand"
)

func CreateDrops(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, id uint32, monsterId uint32, x int16, y int16, killerId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, id uint32, monsterId uint32, x int16, y int16, killerId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, id uint32, monsterId uint32, x int16, y int16, killerId uint32) error {
			// TODO determine type of drop
			dropType := byte(0)

			ds, err := drop.GetByMonsterId(l)(ctx)(monsterId)
			if err != nil {
				return err
			}
			l.Debugf("Monster [%d] has [%d] drops to evaluate.", monsterId, len(ds))

			ds = getSuccessfulDrops(ds, killerId)

			for i, d := range ds {
				_ = drop.Create(l)(ctx)(worldId, channelId, mapId, i+1, id, x, y, killerId, dropType, d)
			}
			return nil
		}
	}
}

func getSuccessfulDrops(options []drop.Model, killerId uint32) []drop.Model {
	res := make([]drop.Model, 0)
	for _, d := range options {
		if evaluateSuccess(d, killerId) {
			res = append(res, d)
		}
	}
	return res
}

func evaluateSuccess(d drop.Model, killerId uint32) bool {
	//TODO evaluate rates
	//TODO channel rate
	//TODO buff rate  (cards)
	//TODO evaluate card rate for killer, whether it's meso or drop.
	chance := int32(math.Min(float64(d.Chance()), math.MaxUint32))
	return rand.Int31n(999999) < chance
}

func DistributeExperience(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, monsterId uint32, damageEntries []DamageEntryModel) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, monsterId uint32, damageEntries []DamageEntryModel) error {
		return func(worldId byte, channelId byte, mapId uint32, monsterId uint32, damageEntries []DamageEntryModel) error {
			d, _ := produceDistribution(l)(ctx)(worldId, channelId, mapId, monsterId, damageEntries)()
			for k, v := range d.Solo() {
				exp := float64(v) * d.ExperiencePerDamage()
				c, err := character.GetById(l)(ctx)(k)
				if err != nil {
					l.WithError(err).Errorf("Unable to locate character %d whose for distributing experience from monster death.", k)
				} else {
					whiteExperienceGain := isWhiteExperienceGain(c.Id(), d.PersonalRatio(), d.StandardDeviationRatio())
					distributeCharacterExperience(l)(ctx)(worldId, channelId, c.Id(), c.Level(), exp, 0.0, c.Level(), true, whiteExperienceGain, false)
				}
			}
			return nil
		}
	}
}

func produceDistribution(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, monsterId uint32, damageEntries []DamageEntryModel) model.Provider[DamageDistributionModel] {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, monsterId uint32, damageEntries []DamageEntryModel) model.Provider[DamageDistributionModel] {
		return func(worldId byte, channelId byte, mapId uint32, monsterId uint32, damageEntries []DamageEntryModel) model.Provider[DamageDistributionModel] {
			mi, err := information.GetById(l)(ctx)(monsterId)
			if err != nil {
				return model.ErrorProvider[DamageDistributionModel](err)
			}

			cim, err := model.CollectToMap[uint32, uint32, bool](_map.CharacterIdsInMapModelProvider(l)(ctx)(worldId, channelId, mapId), func(m uint32) uint32 {
				return m
			}, func(m uint32) bool {
				return true
			})()
			if err != nil {
				return model.ErrorProvider[DamageDistributionModel](err)
			}

			totalEntries := 0
			// TODO parties
			partyDistribution := make(map[uint32]map[uint32]uint32)
			soloDistribution := make(map[uint32]uint32)

			for _, de := range damageEntries {
				if _, ok := cim[de.characterId]; ok {
					soloDistribution[de.characterId] = de.damage
				}
				totalEntries += 1
			}

			// TODO account for healing
			totalDamage := mi.HP()
			epd := float64(mi.Experience()*20) / float64(totalDamage)

			personalRatio := make(map[uint32]float64)
			entryExperienceRatio := make([]float64, 0)

			for k, v := range soloDistribution {
				ratio := float64(v) / float64(totalDamage)
				personalRatio[k] = ratio
				entryExperienceRatio = append(entryExperienceRatio, ratio)
			}

			for _, party := range partyDistribution {
				ratio := 0.0
				for k, v := range party {
					cr := float64(v) / float64(totalDamage)
					personalRatio[k] = cr
					ratio += cr
				}
				entryExperienceRatio = append(entryExperienceRatio, ratio)
			}

			stdr := calculateExperienceStandardDeviationThreshold(entryExperienceRatio, totalEntries)
			m := DamageDistributionModel{
				solo:                   soloDistribution,
				party:                  partyDistribution,
				personalRatio:          personalRatio,
				experiencePerDamage:    epd,
				standardDeviationRatio: stdr,
			}
			return model.FixedProvider(m)
		}
	}
}

func calculateExperienceStandardDeviationThreshold(entryExperienceRatio []float64, totalEntries int) float64 {
	averageExperienceReward := 0.0
	for _, v := range entryExperienceRatio {
		averageExperienceReward += v
	}
	averageExperienceReward /= float64(totalEntries)

	varExperienceReward := 0.0
	for _, v := range entryExperienceRatio {
		varExperienceReward += math.Pow(v-averageExperienceReward, 2)
	}
	varExperienceReward /= float64(len(entryExperienceRatio))

	return averageExperienceReward + math.Sqrt(varExperienceReward)
}

func isWhiteExperienceGain(characterId uint32, personalRatio map[uint32]float64, standardDeviationRatio float64) bool {
	if val, ok := personalRatio[characterId]; ok {
		return val >= standardDeviationRatio
	} else {
		return false
	}
}

func distributeCharacterExperience(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, level byte, experience float64, partyBonusMod float64, totalPartyLevel byte, hightestPartyDamage bool, whiteExperienceGain bool, hasPartySharers bool) {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, level byte, experience float64, partyBonusMod float64, totalPartyLevel byte, hightestPartyDamage bool, whiteExperienceGain bool, hasPartySharers bool) {
		return func(worldId byte, channelId byte, characterId uint32, level byte, experience float64, partyBonusMod float64, totalPartyLevel byte, hightestPartyDamage bool, whiteExperienceGain bool, hasPartySharers bool) {
			expSplitCommonMod := 0.8
			characterExperience := (float64(expSplitCommonMod) * float64(level)) / float64(totalPartyLevel)
			if hightestPartyDamage {
				characterExperience += float64(0.2)
			}
			characterExperience *= experience
			bonusExperience := partyBonusMod * characterExperience

			_ = character.AwardExperience(l)(ctx)(worldId, channelId, characterId, whiteExperienceGain, uint32(characterExperience), uint32(bonusExperience))
		}
	}
}
