package item

import (
	"atlas-query-aggregator/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA") + "/data"
}

func requestConsumable(itemId uint32) requests.Request[ConsumableRestModel] {
	return rest.MakeGetRequest[ConsumableRestModel](
		fmt.Sprintf(getBaseRequest()+"/consumables/%d", itemId),
	)
}

func requestSetup(itemId uint32) requests.Request[SetupRestModel] {
	return rest.MakeGetRequest[SetupRestModel](
		fmt.Sprintf(getBaseRequest()+"/setups/%d", itemId),
	)
}

func requestEtc(itemId uint32) requests.Request[EtcRestModel] {
	return rest.MakeGetRequest[EtcRestModel](
		fmt.Sprintf(getBaseRequest()+"/etcs/%d", itemId),
	)
}

func requestEquipable(itemId uint32) requests.Request[EquipableRestModel] {
	return rest.MakeGetRequest[EquipableRestModel](
		fmt.Sprintf(getBaseRequest()+"/equipables/%d", itemId),
	)
}
