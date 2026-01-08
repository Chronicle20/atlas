package npc

import (
	"atlas-saga-orchestrator/rest"
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
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+npcById, npcId))
}
