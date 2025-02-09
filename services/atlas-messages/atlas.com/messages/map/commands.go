package _map

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
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
				idProvider := model.ToSliceProvider(model.FixedProvider(c.Id()))
				return warpCommandProducer(worldId, channelId, c.Id(), idProvider, match[1])
			}

			re = regexp.MustCompile("@warp map (\\d*)")
			match = re.FindStringSubmatch(m)
			if len(match) == 2 {
				idProvider := CharacterIdsInMapStringProvider(l)(ctx)(worldId, channelId, match[1])
				return warpCommandProducer(worldId, channelId, c.Id(), idProvider, match[1])
			}

			re = regexp.MustCompile(`@warp\s+(\w+)\s+(\d+)`)
			match = re.FindStringSubmatch(m)
			if len(match) == 3 {
				idProvider := model.ToSliceProvider(character.IdByNameProvider(l)(ctx)(match[1]))
				return warpCommandProducer(worldId, channelId, c.Id(), idProvider, match[2])
			}

			return nil, false
		}
	}
}

func warpCommandProducer(worldId byte, channelId byte, actorId uint32, idProvider model.Provider[[]uint32], mapStr string) (command.Executor, bool) {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			requestedMapId, err := strconv.ParseUint(mapStr, 10, 32)
			if err != nil {
				return errors.New("map does not exist")
			}

			exists := Exists(l)(ctx)(uint32(requestedMapId))
			if !exists {
				l.Debugf("Ignoring character [%d] command [%d], because they did not input a valid map.", actorId, requestedMapId)
				return errors.New("map does not exist")
			}

			ids, err := idProvider()
			if err != nil {
				return err
			}
			for _, id := range ids {
				err = WarpRandom(l)(ctx)(worldId)(channelId)(id)(uint32(requestedMapId))
				if err != nil {
					l.WithError(err).Errorf("Unable to warp character [%d] via warp map command.", id)
				}
			}
			return err
		}
	}, true
}
