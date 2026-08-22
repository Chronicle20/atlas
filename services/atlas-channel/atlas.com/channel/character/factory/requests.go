package factory

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/seed"

	// MapleLifeResource is the factory's Maple Life character-creation
	// route, registered in
	// services/atlas-character-factory/atlas.com/character-factory/factory/resource.go.
	MapleLifeResource = "factory/characters/maple-life"
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

func requestCreateMapleLife(ctx context.Context, accountId uint32, worldId world.Id, name string, classOrdinal uint32, gender byte, face uint32, hair uint32, hairColor uint32, skinColor byte, sp byte) requests.Request[CreateCharacterResponse] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[CreateCharacterResponse](err)
	}
	i := MapleLifeCreateRestModel{
		AccountId:    accountId,
		WorldId:      byte(worldId),
		Name:         name,
		ClassOrdinal: classOrdinal,
		Gender:       gender,
		Face:         face,
		Hair:         hair,
		HairColor:    hairColor,
		SkinColor:    skinColor,
		SP:           sp,
	}
	return requests.PostRequest[CreateCharacterResponse](root+MapleLifeResource, i)
}
