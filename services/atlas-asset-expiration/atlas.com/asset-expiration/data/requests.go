package data

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Equipment   = "data/equipment/%d"
	Consumables = "data/consumables/%d"
	Setup       = "data/setups/%d"
	Etc         = "data/etcs/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestEquipment(templateId uint32) requests.Request[EquipmentRestModel] {
	return requests.GetRequest[EquipmentRestModel](fmt.Sprintf(getBaseRequest()+Equipment, templateId))
}

func requestConsumable(templateId uint32) requests.Request[ConsumableRestModel] {
	return requests.GetRequest[ConsumableRestModel](fmt.Sprintf(getBaseRequest()+Consumables, templateId))
}

func requestSetup(templateId uint32) requests.Request[SetupRestModel] {
	return requests.GetRequest[SetupRestModel](fmt.Sprintf(getBaseRequest()+Setup, templateId))
}

func requestEtc(templateId uint32) requests.Request[EtcRestModel] {
	return requests.GetRequest[EtcRestModel](fmt.Sprintf(getBaseRequest()+Etc, templateId))
}
