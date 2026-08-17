package buffs

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	CharacterBuffs = "characters/%d/buffs"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "BUFFS")
}

// characterBuffsUrl returns the list URL for a character's buffs.
func characterBuffsUrl(ctx context.Context, characterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+CharacterBuffs, characterId), nil
}

// identity is the no-op transformer for requests.DrainProvider, since
// BuffRestModel is already the target type for this consumer.
func identity(m BuffRestModel) (BuffRestModel, error) {
	return m, nil
}

// RequestCharacterBuffs fetches ALL active buffs for a character. The
// upstream atlas-buffs list is now paginated (task-117); fetchBuffBonuses
// (the sole caller) must see every buff to compute stat bonuses, so this
// drains every page rather than fetching just the first. The
// requests.Request[[]BuffRestModel] return type is preserved so call sites
// are unchanged.
func RequestCharacterBuffs(characterId uint32) requests.Request[[]BuffRestModel] {
	return func(l logrus.FieldLogger, ctx context.Context) ([]BuffRestModel, error) {
		url, err := characterBuffsUrl(ctx, characterId)
		if err != nil {
			return nil, err
		}
		return requests.DrainProvider[BuffRestModel, BuffRestModel](l, ctx)(url, 250, identity, model.Filters[BuffRestModel]())()
	}
}
