package status

import (
	"atlas-npc-conversations/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("QUEST")
}

// RequestByCharacterAndQuest returns a request to get quest status for a character
// Calls GET /characters/{characterId}/quests/{questId}
func RequestByCharacterAndQuest(characterId uint32, questId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+"characters/%d/quests/%d", characterId, questId))
}
