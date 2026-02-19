package item

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA") + "/data"
}

func requestConsumable(itemId uint32) requests.Request[ConsumableRestModel] {
	return requests.GetRequest[ConsumableRestModel](
		fmt.Sprintf(getBaseRequest()+"/consumables/%d", itemId),
	)
}

func requestSetup(itemId uint32) requests.Request[SetupRestModel] {
	return requests.GetRequest[SetupRestModel](
		fmt.Sprintf(getBaseRequest()+"/setups/%d", itemId),
	)
}

func requestEtc(itemId uint32) requests.Request[EtcRestModel] {
	return requests.GetRequest[EtcRestModel](
		fmt.Sprintf(getBaseRequest()+"/etcs/%d", itemId),
	)
}

func requestEquipable(itemId uint32) requests.Request[EquipableRestModel] {
	return requests.GetRequest[EquipableRestModel](
		fmt.Sprintf(getBaseRequest()+"/equipables/%d", itemId),
	)
}
