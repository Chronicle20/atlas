package _map

import (
	"atlas-channel/session"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func CharacterIdsInMapModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model) model.Provider[[]uint32] {
	return func(ctx context.Context) func(m _map.Model) model.Provider[[]uint32] {
		return func(m _map.Model) model.Provider[[]uint32] {
			return requests.SliceProvider[RestModel, uint32](l, ctx)(requestCharactersInMap(m), Extract, model.Filters[uint32]())
		}
	}
}

func GetCharacterIdsInMap(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model) ([]uint32, error) {
	return func(ctx context.Context) func(m _map.Model) ([]uint32, error) {
		return func(m _map.Model) ([]uint32, error) {
			return CharacterIdsInMapModelProvider(l)(ctx)(m)()
		}
	}
}

func ForSessionsInSessionsMap(l logrus.FieldLogger) func(ctx context.Context) func(f func(oid uint32) model.Operator[session.Model]) model.Operator[session.Model] {
	return func(ctx context.Context) func(f func(oid uint32) model.Operator[session.Model]) model.Operator[session.Model] {
		return func(f func(oid uint32) model.Operator[session.Model]) model.Operator[session.Model] {
			return func(s session.Model) error {
				return session.ForEachByCharacterId(tenant.MustFromContext(ctx))(CharacterIdsInMapModelProvider(l)(ctx)(s.Map()), f(s.CharacterId()))
			}
		}
	}
}

func ForSessionsInMap(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, o model.Operator[session.Model]) error {
	return func(ctx context.Context) func(m _map.Model, o model.Operator[session.Model]) error {
		return func(m _map.Model, o model.Operator[session.Model]) error {
			return session.ForEachByCharacterId(tenant.MustFromContext(ctx))(CharacterIdsInMapModelProvider(l)(ctx)(m), o)
		}
	}
}

func NotCharacterIdFilter(referenceCharacterId uint32) func(characterId uint32) bool {
	return func(characterId uint32) bool {
		return referenceCharacterId != characterId
	}
}

func OtherCharacterIdsInMapModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, referenceCharacterId uint32) model.Provider[[]uint32] {
	return func(ctx context.Context) func(m _map.Model, referenceCharacterId uint32) model.Provider[[]uint32] {
		return func(m _map.Model, referenceCharacterId uint32) model.Provider[[]uint32] {
			return model.FilteredProvider(CharacterIdsInMapModelProvider(l)(ctx)(m), model.Filters(NotCharacterIdFilter(referenceCharacterId)))
		}
	}
}

func ForOtherSessionsInMap(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, referenceCharacterId uint32, o model.Operator[session.Model]) error {
	return func(ctx context.Context) func(m _map.Model, referenceCharacterId uint32, o model.Operator[session.Model]) error {
		return func(m _map.Model, referenceCharacterId uint32, o model.Operator[session.Model]) error {
			p := OtherCharacterIdsInMapModelProvider(l)(ctx)(m, referenceCharacterId)
			return session.ForEachByCharacterId(tenant.MustFromContext(ctx))(p, o)
		}
	}
}
