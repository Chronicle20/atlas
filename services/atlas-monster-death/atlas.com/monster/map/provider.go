package _map

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// CharacterIdsInFieldModelProvider fetches every character currently in one
// map instance, used to distribute mob-kill drops/quest-progress to
// everyone present when a monster dies. The upstream atlas-maps list is now
// paginated (task-117), so this drains every page rather than fetching just
// the first -- a truncated list here means some players in the map silently
// miss their drop/quest credit.
func CharacterIdsInFieldModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model) model.Provider[[]uint32] {
	return func(ctx context.Context) func(f field.Model) model.Provider[[]uint32] {
		return func(f field.Model) model.Provider[[]uint32] {
			return requests.DrainProvider[RestModel, uint32](l, ctx)(charactersInFieldUrl(f), 250, Extract, model.Filters[uint32]())
		}
	}
}
