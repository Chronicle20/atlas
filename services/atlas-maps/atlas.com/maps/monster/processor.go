package monster

import (
	"context"
	"github.com/sirupsen/logrus"
)

func CountInMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) (int, error) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) (int, error) {
		return func(worldId byte, channelId byte, mapId uint32) (int, error) {
			data, err := requestInMap(worldId, channelId, mapId)(l, ctx)
			if err != nil {
				return 0, err
			}
			return len(data), nil
		}
	}
}

func CreateMonster(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, fh uint16, team int32) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, fh uint16, team int32) {
		return func(worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, fh uint16, team int32) {
			_, err := requestCreate(worldId, channelId, mapId, monsterId, x, y, fh, team)(l, ctx)
			if err != nil {
				l.WithError(err).Errorf("Creating monster for map %d", mapId)
			}
		}
	}
}
