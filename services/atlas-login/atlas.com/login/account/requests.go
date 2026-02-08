package account

import (
	"atlas-login/rest"
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
	return rest.MakeGetRequest[[]RestModel](getBaseRequest() + AccountsResource)
}

func requestAccountByName(name string) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+AccountsByName, name))
}

func requestAccountById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+AccountsById, id))
}

func requestUpdate(m Model) requests.Request[RestModel] {
	im, _ := Transform(m)
	return rest.MakePatchRequest[RestModel](fmt.Sprintf(getBaseRequest()+Update, m.id), im)
}

func requestRecordPinAttempt(accountId uint32, success bool) requests.Request[PinAttemptOutputRestModel] {
	input := PinAttemptInputRestModel{Success: success}
	return rest.MakePostRequest[PinAttemptOutputRestModel](fmt.Sprintf(getBaseRequest()+PinAttempts, accountId), input)
}

func requestRecordPicAttempt(accountId uint32, success bool) requests.Request[PicAttemptOutputRestModel] {
	input := PicAttemptInputRestModel{Success: success}
	return rest.MakePostRequest[PicAttemptOutputRestModel](fmt.Sprintf(getBaseRequest()+PicAttempts, accountId), input)
}
