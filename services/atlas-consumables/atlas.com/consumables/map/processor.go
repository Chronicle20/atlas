package _map

import (
	"atlas-consumables/character"
	"atlas-consumables/portal"
	"context"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func WarpRandom(l logrus.FieldLogger) func(ctx context.Context) func(m _map2.Model) func(characterId uint32) error {
	return func(ctx context.Context) func(m _map2.Model) func(characterId uint32) error {
		return func(m _map2.Model) func(characterId uint32) error {
			return func(characterId uint32) error {
				return WarpToPortal(l)(ctx)(m, characterId, portal.RandomSpawnPointIdProvider(l)(ctx)(m.MapId()))
			}
		}
	}
}

func WarpToPortal(l logrus.FieldLogger) func(ctx context.Context) func(m _map2.Model, characterId uint32, p model.Provider[uint32]) error {
	return func(ctx context.Context) func(m _map2.Model, characterId uint32, p model.Provider[uint32]) error {
		return func(m _map2.Model, characterId uint32, p model.Provider[uint32]) error {
			id, err := p()
			if err != nil {
				return err
			}
			return character.ChangeMap(l)(ctx)(m, characterId, id)
		}
	}
}
