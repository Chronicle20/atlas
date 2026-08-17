package cosmetic

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	CharacterResource = "/characters/%d"
)

func getCharacterServiceUrl(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CHARACTERS")
}

func requestCharacterById(ctx context.Context, characterId uint32) requests.Request[RestCharacterModel] {
	root, err := getCharacterServiceUrl(ctx)
	if err != nil {
		return requests.ErrorRequest[RestCharacterModel](err)
	}
	return requests.GetRequest[RestCharacterModel](fmt.Sprintf(root+CharacterResource, characterId))
}

func requestUpdateCharacter(ctx context.Context, characterId uint32, updateRequest CharacterUpdateRequest) requests.Request[RestCharacterModel] {
	root, err := getCharacterServiceUrl(ctx)
	if err != nil {
		return requests.ErrorRequest[RestCharacterModel](err)
	}
	return requests.PatchRequest[RestCharacterModel](
		fmt.Sprintf(root+CharacterResource, characterId),
		updateRequest,
	)
}
