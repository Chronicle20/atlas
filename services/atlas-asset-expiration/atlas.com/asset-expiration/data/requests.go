package data

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Equipment   = "data/equipment/%d"
	Consumables = "data/consumables/%d"
	Setup       = "data/setups/%d"
	Etc         = "data/etcs/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestEquipment(ctx context.Context, templateId uint32) requests.Request[EquipmentRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[EquipmentRestModel](err)
	}
	return requests.GetRequest[EquipmentRestModel](fmt.Sprintf(root+Equipment, templateId))
}

func requestConsumable(ctx context.Context, templateId uint32) requests.Request[ConsumableRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[ConsumableRestModel](err)
	}
	return requests.GetRequest[ConsumableRestModel](fmt.Sprintf(root+Consumables, templateId))
}

func requestSetup(ctx context.Context, templateId uint32) requests.Request[SetupRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[SetupRestModel](err)
	}
	return requests.GetRequest[SetupRestModel](fmt.Sprintf(root+Setup, templateId))
}

func requestEtc(ctx context.Context, templateId uint32) requests.Request[EtcRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[EtcRestModel](err)
	}
	return requests.GetRequest[EtcRestModel](fmt.Sprintf(root+Etc, templateId))
}
