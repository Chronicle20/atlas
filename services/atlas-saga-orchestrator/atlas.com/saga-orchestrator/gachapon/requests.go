package gachapon

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const BaseUrl = "GACHAPONS_URL"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, BaseUrl)
}

func requestSelectReward(ctx context.Context, gachaponId string) requests.Request[RewardRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RewardRestModel](err)
	}
	return requests.PostRequest[RewardRestModel](
		fmt.Sprintf("%sgachapons/%s/rewards/select", root, gachaponId), nil)
}

func requestGetGachapon(ctx context.Context, gachaponId string) requests.Request[GachaponRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[GachaponRestModel](err)
	}
	return requests.GetRequest[GachaponRestModel](
		fmt.Sprintf("%sgachapons/%s", root, gachaponId))
}

func SelectReward(l logrus.FieldLogger, ctx context.Context) func(gachaponId string) (RewardRestModel, error) {
	return func(gachaponId string) (RewardRestModel, error) {
		return requestSelectReward(ctx, gachaponId)(l, ctx)
	}
}

func GetGachapon(l logrus.FieldLogger, ctx context.Context) func(gachaponId string) (GachaponRestModel, error) {
	return func(gachaponId string) (GachaponRestModel, error) {
		return requestGetGachapon(ctx, gachaponId)(l, ctx)
	}
}
