package account

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	AccountsResource = "accounts"
	AccountsById     = AccountsResource + "/%d"
	PicAttempts      = AccountsResource + "/%d/pic-attempts"
)

func getBaseRequest() string {
	return requests.RootUrl("ACCOUNTS")
}

func requestAccountById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+AccountsById, id))
}

// requestRecordPicAttempt mirrors atlas-login's account/requests.go — it is
// the lockout counter behind the account PIC (the credential these two
// cash-shop check ops validate). Record on both outcomes: success resets the
// counter, failure increments it.
func requestRecordPicAttempt(accountId uint32, success bool, ipAddress string, hwid string) requests.Request[PicAttemptOutputRestModel] {
	input := PicAttemptInputRestModel{Success: success, IpAddress: ipAddress, HWID: hwid}
	return requests.PostRequest[PicAttemptOutputRestModel](fmt.Sprintf(getBaseRequest()+PicAttempts, accountId), input)
}
