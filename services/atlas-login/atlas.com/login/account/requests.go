package account

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	AccountsResource = "accounts"
	AccountsByName   = AccountsResource + "?name=%s"
	AccountsById     = AccountsResource + "/%d"
	Update           = AccountsResource + "/%d"
	PinAttempts      = AccountsResource + "/%d/pin-attempts"
	PicAttempts      = AccountsResource + "/%d/pic-attempts"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "ACCOUNTS")
}

func requestAccountByName(ctx context.Context, name string) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+AccountsByName, name))
}

func requestAccountById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+AccountsById, id))
}

func requestUpdate(ctx context.Context, m Model) requests.Request[RestModel] {
	im, _ := Transform(m)
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.PatchRequest[RestModel](fmt.Sprintf(root+Update, m.id), im)
}

func requestRecordPinAttempt(ctx context.Context, accountId uint32, success bool, ipAddress string, hwid string) requests.Request[PinAttemptOutputRestModel] {
	input := PinAttemptInputRestModel{Success: success, IpAddress: ipAddress, HWID: hwid}
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[PinAttemptOutputRestModel](err)
	}
	return requests.PostRequest[PinAttemptOutputRestModel](fmt.Sprintf(root+PinAttempts, accountId), input)
}

func requestRecordPicAttempt(ctx context.Context, accountId uint32, success bool, ipAddress string, hwid string) requests.Request[PicAttemptOutputRestModel] {
	input := PicAttemptInputRestModel{Success: success, IpAddress: ipAddress, HWID: hwid}
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[PicAttemptOutputRestModel](err)
	}
	return requests.PostRequest[PicAttemptOutputRestModel](fmt.Sprintf(root+PicAttempts, accountId), input)
}
