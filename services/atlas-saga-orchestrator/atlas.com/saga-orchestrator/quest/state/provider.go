package state

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func startedQuestsProvider(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestStartedQuests(characterId), Extract, model.Filters[Model]())
		}
	}
}

func GetStartedQuestIds(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (map[uint32]bool, error) {
	return func(ctx context.Context) func(characterId uint32) (map[uint32]bool, error) {
		return func(characterId uint32) (map[uint32]bool, error) {
			quests, err := startedQuestsProvider(l)(ctx)(characterId)()
			if err != nil {
				return nil, err
			}

			result := make(map[uint32]bool)
			for _, q := range quests {
				result[q.QuestId()] = true
			}
			return result, nil
		}
	}
}
