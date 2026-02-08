package gachapon

import (
	"atlas-saga-orchestrator/rest"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const BaseUrl = "GACHAPONS_URL"

func getBaseRequest() string {
	return requests.RootUrl(BaseUrl)
}

func requestSelectReward(gachaponId string) requests.Request[RewardRestModel] {
	return rest.MakePostRequest[RewardRestModel](
		fmt.Sprintf("%sgachapons/%s/rewards/select", getBaseRequest(), gachaponId), nil)
}

func requestGetGachapon(gachaponId string) requests.Request[GachaponRestModel] {
	return rest.MakeGetRequest[GachaponRestModel](
		fmt.Sprintf("%sgachapons/%s", getBaseRequest(), gachaponId))
}

func SelectReward(l logrus.FieldLogger, ctx context.Context) func(gachaponId string) (RewardRestModel, error) {
	return func(gachaponId string) (RewardRestModel, error) {
		return requestSelectReward(gachaponId)(l, ctx)
	}
}

func GetGachapon(l logrus.FieldLogger, ctx context.Context) func(gachaponId string) (GachaponRestModel, error) {
	return func(gachaponId string) (GachaponRestModel, error) {
		return requestGetGachapon(gachaponId)(l, ctx)
	}
}
