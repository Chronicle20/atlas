package quest

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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

// byCharacterUrl returns the list URL for a character's quests. It is a
// bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func byCharacterUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+ByCharacter, characterId)
}

// startedByCharacterUrl and completedByCharacterUrl are bare URLs (not
// requests.Request) because both lists are now paginated server-side
// (task-117) and consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func startedByCharacterUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+StartedQuests, characterId)
}

func completedByCharacterUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+CompletedQuests, characterId)
}