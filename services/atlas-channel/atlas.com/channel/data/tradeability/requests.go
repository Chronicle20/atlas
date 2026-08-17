package tradeability

import (
	"context"
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

func getBaseRequest(ctx context.Context) (string, error) { return requests.RootUrlFor(ctx, "DATA") }

func requestEquipment(ctx context.Context, id item.Id) requests.Request[EquipmentRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[EquipmentRestModel](err)
	}
	return requests.GetRequest[EquipmentRestModel](fmt.Sprintf(root+EquipmentById, id))
}

func requestConsumable(ctx context.Context, id item.Id) requests.Request[ConsumableRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[ConsumableRestModel](err)
	}
	return requests.GetRequest[ConsumableRestModel](fmt.Sprintf(root+ConsumableById, id))
}

func requestSetup(ctx context.Context, id item.Id) requests.Request[SetupRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[SetupRestModel](err)
	}
	return requests.GetRequest[SetupRestModel](fmt.Sprintf(root+SetupById, id))
}

func requestEtc(ctx context.Context, id item.Id) requests.Request[EtcRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[EtcRestModel](err)
	}
	return requests.GetRequest[EtcRestModel](fmt.Sprintf(root+EtcById, id))
}

func requestCash(ctx context.Context, id item.Id) requests.Request[CashRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[CashRestModel](err)
	}
	return requests.GetRequest[CashRestModel](fmt.Sprintf(root+CashById, id))
}
