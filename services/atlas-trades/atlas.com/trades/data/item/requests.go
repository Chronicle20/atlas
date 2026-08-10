package item

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	EquipmentById  = "data/equipment/%d"
	ConsumableById = "data/consumables/%d"
	SetupById      = "data/setups/%d"
	EtcById        = "data/etcs/%d"
	CashById       = "data/cash/items/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestEquipment(id item.Id) requests.Request[EquipmentRestModel] {
	return requests.GetRequest[EquipmentRestModel](fmt.Sprintf(getBaseRequest()+EquipmentById, id))
}

func requestConsumable(id item.Id) requests.Request[ConsumableRestModel] {
	return requests.GetRequest[ConsumableRestModel](fmt.Sprintf(getBaseRequest()+ConsumableById, id))
}

func requestSetup(id item.Id) requests.Request[SetupRestModel] {
	return requests.GetRequest[SetupRestModel](fmt.Sprintf(getBaseRequest()+SetupById, id))
}

func requestEtc(id item.Id) requests.Request[EtcRestModel] {
	return requests.GetRequest[EtcRestModel](fmt.Sprintf(getBaseRequest()+EtcById, id))
}

func requestCash(id item.Id) requests.Request[CashRestModel] {
	return requests.GetRequest[CashRestModel](fmt.Sprintf(getBaseRequest()+CashById, id))
}
