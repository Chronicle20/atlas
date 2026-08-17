package monsterbook

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	// Resource is the path template for fetching a character's monster book
	// collection summary from atlas-monster-book.
	Resource = "characters/%d/monster-book"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MONSTER_BOOK")
}

func requestByCharacterId(ctx context.Context, characterId character.Id) requests.Request[CollectionRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[CollectionRestModel](err)
	}
	return requests.GetRequest[CollectionRestModel](fmt.Sprintf(root+Resource, characterId))
}
