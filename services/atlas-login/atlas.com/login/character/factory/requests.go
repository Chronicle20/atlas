package factory

import (
	"atlas-login/rest"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "characters/seed"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTER_FACTORY")
}

func requestCreate(accountId uint32, worldId byte, name string, jobIndex uint32, subJobIndex uint16, face uint32, hair uint32, color uint32, skinColor uint32, gender byte, top uint32, bottom uint32, shoes uint32, weapon uint32,
	strength byte, dexterity byte, intelligence byte, luck byte) requests.Request[CreateCharacterResponse] {
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
	return rest.MakePostRequest[CreateCharacterResponse](getBaseRequest()+Resource, i)
}
