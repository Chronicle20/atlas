package monsterbook

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	// Resource is the path template for fetching a character's monster book
	// collection from atlas-monster-book.
	Resource = "characters/%d/monster-book"
	// CardsResource is the path template for a character's owned card list.
	CardsResource = "characters/%d/monster-book/cards"
)

var baseURLProvider = func() string {
	return requests.RootUrl("MONSTER_BOOK")
}

func getBaseRequest() string {
	return baseURLProvider()
}

func requestByCharacterId(characterId character.Id) requests.Request[CollectionRestModel] {
	return requests.GetRequest[CollectionRestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}

// cardsByCharacterIdUrl returns the list URL for a character's owned
// monster-book cards. It is a bare URL (not a requests.Request) because
// the list is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func cardsByCharacterIdUrl(characterId character.Id) string {
	return fmt.Sprintf(getBaseRequest()+CardsResource, characterId)
}

// SetBaseURLForTest swaps the base URL for tests using httptest. Only
// call from a test; production code uses the env-driven default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string { return url + "/api/" }
	return func() { baseURLProvider = prev }
}
