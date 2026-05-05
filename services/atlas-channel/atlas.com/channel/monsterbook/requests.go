package monsterbook

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	// Resource is the path template for fetching a character's monster book
	// collection from atlas-monster-book.
	Resource = "characters/%d/monster-book"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTER_BOOK")
}

func requestByCharacterId(characterId uint32) requests.Request[CollectionRestModel] {
	return requests.GetRequest[CollectionRestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}
