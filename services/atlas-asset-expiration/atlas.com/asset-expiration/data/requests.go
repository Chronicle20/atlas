package data

import (
	"atlas-asset-expiration/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Equipment   = "equipment/%d"
	Consumables = "consumables/%d"
	Setup       = "setup/%d"
	Etc         = "etc/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestEquipment(templateId uint32) requests.Request[EquipmentRestModel] {
	return rest.MakeGetRequest[EquipmentRestModel](fmt.Sprintf(getBaseRequest()+Equipment, templateId))
}

func requestConsumable(templateId uint32) requests.Request[ConsumableRestModel] {
	return rest.MakeGetRequest[ConsumableRestModel](fmt.Sprintf(getBaseRequest()+Consumables, templateId))
}

func requestSetup(templateId uint32) requests.Request[SetupRestModel] {
	return rest.MakeGetRequest[SetupRestModel](fmt.Sprintf(getBaseRequest()+Setup, templateId))
}

func requestEtc(templateId uint32) requests.Request[EtcRestModel] {
	return rest.MakeGetRequest[EtcRestModel](fmt.Sprintf(getBaseRequest()+Etc, templateId))
}
