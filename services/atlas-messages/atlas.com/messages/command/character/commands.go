package character

import (
	"atlas-messages/character"
	"atlas-messages/command"
	_map "atlas-messages/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"regexp"
	"strconv"
)

func AwardExperienceCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
		return func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
			var cn string
			var amountStr string

			re := regexp.MustCompile(`@award\s+(\w+)\s+experience\s+(\d+)`)
			match := re.FindStringSubmatch(m)
			if len(match) == 3 {
				cn = match[1]
				amountStr = match[2]
			}

			if len(cn) == 0 {
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
				idProvider = _map.CharacterIdsInMapProvider(l)(ctx)(worldId, channelId, c.MapId())
			} else {
				idProvider = model.ToSliceProvider(character.IdByNameProvider(l)(ctx)(match[1]))
			}

			tAmount, err := strconv.ParseUint(amountStr, 10, 32)
			if err != nil {
				return nil, false
			}
			amount := uint32(tAmount)

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					cids, err := idProvider()
					if err != nil {
						return err
					}
					for _, id := range cids {
						err = character.AwardExperience(l)(ctx)(worldId, channelId, id, amount)
						if err != nil {
							l.WithError(err).Errorf("Unable to award [%d] with [%d] experience.", id, amount)
						}
					}
					return err
				}
			}, true
		}
	}
}
