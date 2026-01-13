package character

import (
	"atlas-character-factory/rest"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-constants/job"
)

const (
	resource = "characters"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestCreate(accountId uint32, worldId byte, name string, gender byte, mapId uint32, jobId job.Id, face uint32, hair uint32, hairColor uint32, skinColor byte) requests.Request[RestModel] {
	i := RestModel{
		AccountId:    accountId,
		WorldId:      worldId,
		Name:         name,
		Gender:       gender,
		MapId:        mapId,
		JobId:        uint16(jobId),
		Face:         face,
		Hair:         hair + hairColor,
		SkinColor:    skinColor,
		Level:        1,
		Hp:           50,
		MaxHp:        50,
		Mp:           5,
		MaxMp:        5,
		Strength:     13,
		Dexterity:    4,
		Intelligence: 4,
		Luck:         4,
	}
	return rest.MakePostRequest[RestModel](getBaseRequest()+resource, i)
}
