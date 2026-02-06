package note

import (
	"atlas-channel/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource     = "characters/%d/notes"
	NoteResource = "notes/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("NOTES")
}

func requestByCharacterId(characterId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}

func requestById(noteId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+NoteResource, noteId))
}
