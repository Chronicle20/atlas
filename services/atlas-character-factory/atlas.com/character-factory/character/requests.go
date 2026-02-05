package character

import (
	"atlas-character-factory/rest"

	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	resource = "characters"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestCreate(accountId uint32, worldId world.Id, name string, gender byte, mapId _map.Id, jobId job.Id, face uint32, hair uint32, hairColor uint32, skinColor byte) requests.Request[RestModel] {
	i := RestModel{
		AccountId:    accountId,
		WorldId:      worldId,
		Name:         name,
		Gender:       gender,
		MapId:        mapId,
		JobId:        jobId,
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
