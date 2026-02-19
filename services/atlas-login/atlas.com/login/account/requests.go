package account

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	AccountsResource   = "accounts"
	AccountsByName     = AccountsResource + "?name=%s"
	AccountsById       = AccountsResource + "/%d"
	Update             = AccountsResource + "/%d"
	PinAttempts        = AccountsResource + "/%d/pin-attempts"
	PicAttempts        = AccountsResource + "/%d/pic-attempts"
)

func getBaseRequest() string {
	return requests.RootUrl("ACCOUNTS")
}

func requestAccounts() requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](getBaseRequest() + AccountsResource)
}

func requestAccountByName(name string) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+AccountsByName, name))
}

func requestAccountById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+AccountsById, id))
}

func requestUpdate(m Model) requests.Request[RestModel] {
	im, _ := Transform(m)
	return requests.PatchRequest[RestModel](fmt.Sprintf(getBaseRequest()+Update, m.id), im)
}

func requestRecordPinAttempt(accountId uint32, success bool, ipAddress string, hwid string) requests.Request[PinAttemptOutputRestModel] {
	input := PinAttemptInputRestModel{Success: success, IpAddress: ipAddress, HWID: hwid}
	return requests.PostRequest[PinAttemptOutputRestModel](fmt.Sprintf(getBaseRequest()+PinAttempts, accountId), input)
}

func requestRecordPicAttempt(accountId uint32, success bool, ipAddress string, hwid string) requests.Request[PicAttemptOutputRestModel] {
	input := PicAttemptInputRestModel{Success: success, IpAddress: ipAddress, HWID: hwid}
	return requests.PostRequest[PicAttemptOutputRestModel](fmt.Sprintf(getBaseRequest()+PicAttempts, accountId), input)
}
