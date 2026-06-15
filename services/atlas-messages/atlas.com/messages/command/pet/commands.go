package pet

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/kafka/message/pet"
	"atlas-messages/kafka/producer"
	_map "atlas-messages/map"
	petlookup "atlas-messages/pet"
	"context"
	"regexp"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

// AwardTamenessCommandProducer handles the GM command
// "@award <target> tameness <amount>", raising the tameness/closeness of each
// target character's spawned pet(s) by reusing atlas-pets' additive
// AWARD_CLOSENESS command. Used to exercise the pet-evolution tameness gate.
func AwardTamenessCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		mp := _map.NewProcessor(l, ctx)
		pp := petlookup.NewProcessor(l, ctx)
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			ch := f.Channel()

			re := regexp.MustCompile(`^@award\s+(\w+)\s+tameness\s+(\d+)$`)
			match := re.FindStringSubmatch(m)
			if len(match) != 3 {
				return nil, false
			}

			if !c.Gm() {
				l.Debugf("Ignoring character [%d] command [%s], because they are not a gm.", c.Id(), m)
				return nil, false
			}

			var idProvider model.Provider[[]uint32]
			if match[1] == "me" {
				idProvider = model.ToSliceProvider(model.FixedProvider(c.Id()))
			} else if match[1] == "map" {
				f := field.NewBuilder(ch.WorldId(), ch.Id(), f.MapId()).Build()
				idProvider = mp.CharacterIdsInFieldProvider(f)
			} else {
				idProvider = model.ToSliceProvider(cp.IdByNameProvider(match[1]))
			}

			tAmount, err := strconv.ParseUint(match[2], 10, 16)
			if err != nil {
				return nil, false
			}
			amount := uint16(tAmount)

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					cids, err := idProvider()
					if err != nil {
						return err
					}
					for _, id := range cids {
						petIds, perr := pp.GetSpawnedPetIds(id)
						if perr != nil {
							l.WithError(perr).Errorf("Unable to resolve spawned pets for character [%d].", id)
							continue
						}
						for _, petId := range petIds {
							emitErr := producer.ProviderImpl(l)(ctx)(pet.EnvCommandTopic)(pet.AwardClosenessCommandProvider(petId, amount))
							if emitErr != nil {
								l.WithError(emitErr).Errorf("Unable to award [%d] tameness to pet [%d] of character [%d].", amount, petId, id)
							}
						}
					}
					return nil
				}
			}, true
		}
	}
}
