package note

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource     = "characters/%d/notes"
	NoteResource = "notes/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("NOTES")
}

// characterNotesUrl returns the list URL for a character's notes. It is a
// bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func characterNotesUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+Resource, characterId)
}

func requestById(noteId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+NoteResource, noteId))
}
