package factory

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/seed"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CHARACTER_FACTORY")
}

func requestCreate(ctx context.Context, accountId uint32, worldId world.Id, name string, jobIndex uint32, subJobIndex uint16, face uint32, hair uint32, color uint32, skinColor uint32, gender byte, top uint32, bottom uint32, shoes uint32, weapon uint32,
	strength byte, dexterity byte, intelligence byte, luck byte,
) requests.Request[CreateCharacterResponse] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[CreateCharacterResponse](err)
	}
	i := RestModel{
		AccountId:    accountId,
		WorldId:      worldId,
		Name:         name,
		Gender:       gender,
		JobIndex:     jobIndex,
		SubJobIndex:  uint32(subJobIndex),
		Face:         face,
		Hair:         hair,
		HairColor:    color,
		SkinColor:    byte(skinColor),
		Top:          top,
		Bottom:       bottom,
		Shoes:        shoes,
		Weapon:       weapon,
		Level:        1,
		Strength:     uint16(strength),
		Dexterity:    uint16(dexterity),
		Intelligence: uint16(intelligence),
		Luck:         uint16(luck),
		Hp:           50,
		Mp:           5,
		MapId:        0,
	}
	return requests.PostRequest[CreateCharacterResponse](root+Resource, i)
}
