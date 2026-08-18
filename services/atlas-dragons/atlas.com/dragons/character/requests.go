package character

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters"
	// Sparse fieldset: atlas-dragons needs only the job and position. Fetching
	// the full character (inventory, skills, buffs) on every field entry would
	// be wasteful on a path that runs for every logging-in character.
	ById = Resource + "/%d?fields[characters]=jobId,x,y,stance"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}
