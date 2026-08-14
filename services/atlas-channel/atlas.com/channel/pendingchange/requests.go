package pendingchange

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/pending-changes"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestCreateUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+Resource, characterId)
}
