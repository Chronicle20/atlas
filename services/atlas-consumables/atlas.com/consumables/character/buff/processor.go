package buff

import (
	"atlas-consumables/character/buff/stat"
	"atlas-consumables/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func Apply(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, fromId uint32, sourceId int32, duration int32, statups []stat.Model) model.Operator[uint32] {
	return func(ctx context.Context) func(m _map.Model, fromId uint32, sourceId int32, duration int32, statups []stat.Model) model.Operator[uint32] {
		return func(m _map.Model, fromId uint32, sourceId int32, duration int32, statups []stat.Model) model.Operator[uint32] {
			return func(characterId uint32) error {
				return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(applyCommandProvider(m, characterId, fromId, sourceId, duration, statups))
			}
		}
	}
}
