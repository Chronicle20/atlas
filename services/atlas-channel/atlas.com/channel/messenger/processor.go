package messenger

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func Create(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) error {
	return func(ctx context.Context) func(characterId uint32) error {
		return func(characterId uint32) error {
			l.Debugf("Character [%d] attempting to create a messenger.", characterId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(createCommandProvider(characterId))
		}
	}
}

func Leave(l logrus.FieldLogger) func(ctx context.Context) func(messengerId uint32, characterId uint32) error {
	return func(ctx context.Context) func(messengerId uint32, characterId uint32) error {
		return func(messengerId uint32, characterId uint32) error {
			l.Debugf("Character [%d] attempting to leave messenger [%d].", characterId, messengerId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(leaveCommandProvider(characterId, messengerId))
		}
	}
}

func RequestInvite(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, targetCharacterId uint32) error {
	return func(ctx context.Context) func(characterId uint32, targetCharacterId uint32) error {
		return func(characterId uint32, targetCharacterId uint32) error {
			l.Debugf("Character [%d] attempting to invite [%d] to a messenger.", characterId, targetCharacterId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestInviteCommandProvider(characterId, targetCharacterId))
		}
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(messengerId uint32) (Model, error) {
	return func(ctx context.Context) func(messengerId uint32) (Model, error) {
		return func(messengerId uint32) (Model, error) {
			return byIdProvider(l)(ctx)(messengerId)()
		}
	}
}

func byIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(messengerId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(messengerId uint32) model.Provider[Model] {
		return func(messengerId uint32) model.Provider[Model] {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(messengerId), Extract)
		}
	}
}

func GetByMemberId(l logrus.FieldLogger) func(ctx context.Context) func(memberId uint32) (Model, error) {
	return func(ctx context.Context) func(memberId uint32) (Model, error) {
		return func(memberId uint32) (Model, error) {
			return ByMemberIdProvider(l)(ctx)(memberId)()
		}
	}
}

func ByMemberIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(memberId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(memberId uint32) model.Provider[Model] {
		return func(memberId uint32) model.Provider[Model] {
			rp := requests.SliceProvider[RestModel, Model](l, ctx)(requestByMemberId(memberId), Extract, model.Filters[Model]())
			return model.FirstProvider(rp, model.Filters[Model]())
		}
	}
}
