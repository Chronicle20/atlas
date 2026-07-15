package quest

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// startedQuestsProvider fetches every started quest for a character. The
// upstream atlas-quest list is now paginated (task-117); GetStartedQuestIds
// below builds a complete questId->started map for mob-kill quest matching,
// so this drains every page rather than fetching just the first.
func startedQuestsProvider(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			return requests.DrainProvider[RestModel, Model](l, ctx)(startedQuestsUrl(characterId), 250, Extract, model.Filters[Model]())
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
