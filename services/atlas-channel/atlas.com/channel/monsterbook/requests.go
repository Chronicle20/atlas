package monsterbook

import (
	"context"
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

var baseURLProvider = func(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MONSTER_BOOK")
}

func getBaseRequest(ctx context.Context) (string, error) {
	return baseURLProvider(ctx)
}

func requestByCharacterId(ctx context.Context, characterId character.Id) requests.Request[CollectionRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[CollectionRestModel](err)
	}
	return requests.GetRequest[CollectionRestModel](fmt.Sprintf(root+Resource, characterId))
}

// cardsByCharacterIdUrl returns the list URL for a character's owned
// monster-book cards. It is a bare URL (not a requests.Request) because
// the list is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func cardsByCharacterIdUrl(ctx context.Context, characterId character.Id) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+CardsResource, characterId), nil
}

// SetBaseURLForTest swaps the base URL for tests using httptest. Only
// call from a test; production code uses the env-driven default. The
// injected closure ignores ctx -- tests always exercise the fixed httptest
// URL regardless of any environment on the context.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func(_ context.Context) (string, error) { return url + "/api/", nil }
	return func() { baseURLProvider = prev }
}
