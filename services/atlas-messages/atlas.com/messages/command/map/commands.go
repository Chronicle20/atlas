package _map

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/map"
	"atlas-messages/message"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"regexp"
	"strconv"
)

func WarpCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
		return func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`@warp\s+(\w+)\s+(\d+)`)
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
				idProvider = _map.NewProcessor(l, ctx).CharacterIdsInMapStringProvider(worldId, channelId, match[2])
			} else {
				idProvider = model.ToSliceProvider(character.NewProcessor(l, ctx).IdByNameProvider(match[1]))
			}

			return warpCommandProducer(worldId, channelId, c.Id(), idProvider, match[2])

		}
	}
}

func warpCommandProducer(worldId byte, channelId byte, actorId uint32, idProvider model.Provider[[]uint32], mapStr string) (command.Executor, bool) {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			mp := _map.NewProcessor(l, ctx)
			requestedMapId, err := strconv.ParseUint(mapStr, 10, 32)
			if err != nil {
				return errors.New("map does not exist")
			}

			exists := mp.Exists(uint32(requestedMapId))
			if !exists {
				l.Debugf("Ignoring character [%d] command [%d], because they did not input a valid map.", actorId, requestedMapId)
				return errors.New("map does not exist")
			}

			ids, err := idProvider()
			if err != nil {
				return err
			}
			for _, id := range ids {
				err = mp.WarpRandom(worldId)(channelId)(id)(uint32(requestedMapId))
				if err != nil {
					l.WithError(err).Errorf("Unable to warp character [%d] via warp map command.", id)
				}
			}
			return err
		}
	}, true
}

func WhereAmICommandProducer(_ logrus.FieldLogger) func(_ context.Context) func(worldId byte, channelId byte, character character.Model, m string) (command.Executor, bool) {
	return func(_ context.Context) func(worldId byte, channelId byte, character character.Model, m string) (command.Executor, bool) {
		return func(worldId byte, channelId byte, character character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`@query map`)
			match := re.FindStringSubmatch(m)
			if len(match) != 1 {
				return nil, false
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					return message.NewProcessor(l, ctx).IssuePinkText(worldId, channelId, character.MapId(), 0, "You are in map "+strconv.Itoa(int(character.MapId())), []uint32{character.Id()})
				}
			}, true
		}
	}
}
