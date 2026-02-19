package gachapon

import (
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
	return requests.PostRequest[RewardRestModel](
		fmt.Sprintf("%sgachapons/%s/rewards/select", getBaseRequest(), gachaponId), nil)
}

func requestGetGachapon(gachaponId string) requests.Request[GachaponRestModel] {
	return requests.GetRequest[GachaponRestModel](
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
