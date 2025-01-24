package character

import (
	"atlas-npc-conversations/job"
	"context"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(characterId), Extract)()
		}
	}
}

type AttributeCriteria func(Model) bool

func MeetsCriteria(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, criteria ...AttributeCriteria) bool {
	return func(ctx context.Context) func(characterId uint32, criteria ...AttributeCriteria) bool {
		return func(characterId uint32, criteria ...AttributeCriteria) bool {
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve character %d for criteria check.", characterId)
				return false
			}
			for _, check := range criteria {
				if ok := check(c); !ok {
					return false
				}
			}
			return true
		}
	}
}

func IsBeginnerTree(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) bool {
	return func(ctx context.Context) func(characterId uint32) bool {
		return func(characterId uint32) bool {
			return MeetsCriteria(l)(ctx)(characterId, IsBeginnerTreeCriteria())
		}
	}
}

func IsBeginnerTreeCriteria() AttributeCriteria {
	return func(c Model) bool {
		return job.IsA(c.JobId(), job.Beginner, job.Noblesse, job.Legend)
	}
}

func HasMeso(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, amount uint32) bool {
	return func(ctx context.Context) func(characterId uint32, amount uint32) bool {
		return func(characterId uint32, amount uint32) bool {
			return MeetsCriteria(l)(ctx)(characterId, HasMesoCriteria(amount))
		}
	}
}

func HasMesoCriteria(amount uint32) AttributeCriteria {
	return func(c Model) bool {
		return c.Meso() >= amount
	}
}

func RequestChangeMeso(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, worldId byte, amount int32) error {
	return func(ctx context.Context) func(characterId uint32, worldId byte, amount int32) error {
		return func(characterId uint32, worldId byte, amount int32) error {
			return nil
		}
	}
}
