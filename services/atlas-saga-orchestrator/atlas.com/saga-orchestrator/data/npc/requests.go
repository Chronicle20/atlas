package npc

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	npcById = "data/npcs/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(npcId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+npcById, npcId))
}
