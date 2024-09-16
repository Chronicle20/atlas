package factory

import (
	"atlas-login/character"
	"atlas-login/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	Resource = "characters"
)

func getBaseRequest() string {
	return os.Getenv("CHARACTER_FACTORY_SERVICE_URL")
}

func requestCreate(accountId uint32, worldId byte, name string, jobIndex uint32, subJobIndex uint16, face uint32, hair uint32, color uint32, skinColor uint32, gender byte, top uint32, bottom uint32, shoes uint32, weapon uint32,
	strength byte, dexterity byte, intelligence byte, luck byte) requests.Request[character.RestModel] {
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
		Strength:     strength,
		Dexterity:    dexterity,
		Intelligence: intelligence,
		Luck:         luck,
	}
	return rest.MakePostRequest[character.RestModel](fmt.Sprintf(getBaseRequest()+Resource), i)
}
