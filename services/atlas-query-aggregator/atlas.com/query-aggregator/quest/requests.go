package quest

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	CharactersResource = "characters"
	QuestsResource     = "quests"
	ByCharacterAndId   = CharactersResource + "/%d/" + QuestsResource + "/%d"
	ByCharacter        = CharactersResource + "/%d/" + QuestsResource
	StartedQuests      = CharactersResource + "/%d/" + QuestsResource + "/started"
	CompletedQuests    = CharactersResource + "/%d/" + QuestsResource + "/completed"
)

func getBaseRequest() string {
	return requests.RootUrl("QUESTS")
}

func requestById(characterId uint32, questId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacterAndId, characterId, questId))
}

func requestByCharacter(characterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByCharacter, characterId))
}

func requestStartedByCharacter(characterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+StartedQuests, characterId))
}

func requestCompletedByCharacter(characterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+CompletedQuests, characterId))
}