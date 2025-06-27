package skill

import (
	"atlas-character/data/skill/effect"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(uniqueId uint32) (Model, error) {
	return func(ctx context.Context) func(uniqueId uint32) (Model, error) {
		return func(uniqueId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(uniqueId), Extract)()
		}
	}
}

func GetEffect(l logrus.FieldLogger) func(ctx context.Context) func(uniqueId uint32, level byte) (effect.Model, error) {
	return func(ctx context.Context) func(uniqueId uint32, level byte) (effect.Model, error) {
		return func(uniqueId uint32, level byte) (effect.Model, error) {
			s, err := GetById(l)(ctx)(uniqueId)
			if err != nil {
				return effect.Model{}, err
			}
			if level == 0 {
				return effect.Model{}, nil
			}
			if len(s.Effects()) < int(level-1) {
				return effect.Model{}, errors.New("level out of bounds")
			}
			return s.Effects()[level-1], nil
		}
	}
}
