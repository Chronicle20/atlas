package shops

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	npcShop = "npcs/%d/shop?include=commodities"
)

func getBaseRequest() string {
	return requests.RootUrl("NPC_SHOP")
}

func requestNPCShop(templateId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+npcShop, templateId))
}
