package chalkboard

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	ById = "chalkboards/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("CHALKBOARDS")
}

func requestById(characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, characterId))
}
