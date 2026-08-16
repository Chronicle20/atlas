package item

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func getBaseRequest(ctx context.Context) (string, error) {
	root, err := requests.RootUrlFor(ctx, "DATA")
	if err != nil {
		return "", err
	}
	return root + "/data", nil
}

func requestConsumable(ctx context.Context, itemId uint32) requests.Request[ConsumableRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[ConsumableRestModel](err)
	}
	return requests.GetRequest[ConsumableRestModel](
		fmt.Sprintf(root+"/consumables/%d", itemId),
	)
}

func requestSetup(ctx context.Context, itemId uint32) requests.Request[SetupRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[SetupRestModel](err)
	}
	return requests.GetRequest[SetupRestModel](
		fmt.Sprintf(root+"/setups/%d", itemId),
	)
}

func requestEtc(ctx context.Context, itemId uint32) requests.Request[EtcRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[EtcRestModel](err)
	}
	return requests.GetRequest[EtcRestModel](
		fmt.Sprintf(root+"/etcs/%d", itemId),
	)
}

func requestEquipable(ctx context.Context, itemId uint32) requests.Request[EquipableRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[EquipableRestModel](err)
	}
	return requests.GetRequest[EquipableRestModel](
		fmt.Sprintf(root+"/equipables/%d", itemId),
	)
}
