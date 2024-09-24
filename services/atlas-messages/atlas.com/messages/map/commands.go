package _map

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"context"
	"github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
)

func WarpCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
		return func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
			if !c.Gm() {
				l.Debugf("Ignoring character [%d] command [%s], because they are not a gm.", c.Id(), m)
				return nil, false
			}

			if !strings.HasPrefix(m, "@warp") {
				return nil, false
			}

			re := regexp.MustCompile("@warp me (\\d*)")
			match := re.FindStringSubmatch(m)
			if len(match) == 2 {
				return warpCharacterCommandProducer(l)(ctx)(worldId, channelId, c.Id(), match[1])
			}

			re = regexp.MustCompile("@warp map (\\d*)")
			match = re.FindStringSubmatch(m)
			if len(match) == 2 {
				return warpMapCommandProducer(l)(ctx)(worldId, channelId, match[1])
			}

			re = regexp.MustCompile(`@warp\s+(\w+)\s+(\d+)`)
			match = re.FindStringSubmatch(m)
			if len(match) == 3 {
				oc, err := character.GetByName(l)(ctx)(match[1])
				if err != nil {
					return nil, false
				}
				return warpCharacterCommandProducer(l)(ctx)(worldId, channelId, oc.Id(), match[2])
			}
			return nil, false
		}
	}
}

func warpMapCommandProducer(_ logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapString string) (command.Executor, bool) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapString string) (command.Executor, bool) {
		return func(worldId byte, channelId byte, mapString string) (command.Executor, bool) {
			mapId, err := strconv.ParseUint(mapString, 10, 32)
			if err != nil {
				return nil, false
			}
			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					var cids []uint32
					cids, err = GetCharacterIdsInMap(l)(ctx)(worldId, channelId, uint32(mapId))
					if err != nil {
						return err
					}
					for _, id := range cids {
						err = WarpRandom(l)(ctx)(worldId)(channelId)(id)(uint32(mapId))
						if err != nil {
							l.WithError(err).Errorf("Unable to warp character [%d] via warp map command.", id)
						}
					}
					return err
				}
			}, true
		}
	}
}

func warpCharacterCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapString string) (command.Executor, bool) {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapString string) (command.Executor, bool) {
		return func(worldId byte, channelId byte, characterId uint32, mapString string) (command.Executor, bool) {
			mapId, err := strconv.ParseUint(mapString, 10, 32)
			if err != nil {
				return nil, false
			}
			exists := Exists(l)(ctx)(uint32(mapId))
			if !exists {
				l.Debugf("Ignoring character [%d] command [%s], because they did not input a valid map.", characterId, mapString)
				return nil, false
			}
			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					return WarpRandom(l)(ctx)(worldId)(channelId)(characterId)(uint32(mapId))
				}
			}, true
		}
	}
}
