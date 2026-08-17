package account

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	AccountsResource = "accounts"
	AccountsById     = AccountsResource + "/%d"
	PicAttempts      = AccountsResource + "/%d/pic-attempts"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "ACCOUNTS")
}

func requestAccountById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+AccountsById, id))
}

// requestRecordPicAttempt mirrors atlas-login's account/requests.go — it is
// the lockout counter behind the account PIC (the credential these two
// cash-shop check ops validate). Record on both outcomes: success resets the
// counter, failure increments it.
func requestRecordPicAttempt(ctx context.Context, accountId uint32, success bool, ipAddress string, hwid string) requests.Request[PicAttemptOutputRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[PicAttemptOutputRestModel](err)
	}
	input := PicAttemptInputRestModel{Success: success, IpAddress: ipAddress, HWID: hwid}
	return requests.PostRequest[PicAttemptOutputRestModel](fmt.Sprintf(root+PicAttempts, accountId), input)
}
