package monster

import (
	"atlas-maps/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func CountInMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) (int, error) {
	return func(worldId byte, channelId byte, mapId uint32) (int, error) {
		data, err := requestInMap(l, span, tenant)(worldId, channelId, mapId)(l)
		if err != nil {
			return 0, err
		}
		return len(data), nil
	}
}

func CreateMonster(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, fh uint16, team int32) {
	return func(worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, fh uint16, team int32) {
		_, err := requestCreate(l, span, tenant)(worldId, channelId, mapId, monsterId, x, y, fh, team)(l)
		if err != nil {
			l.WithError(err).Errorf("Creating monster for map %d", mapId)
		}
	}
}
