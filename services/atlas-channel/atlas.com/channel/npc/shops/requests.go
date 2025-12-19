package shops

import (
	"atlas-channel/rest"
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
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+npcShop, templateId))
}
