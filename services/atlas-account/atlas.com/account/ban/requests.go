package ban

import (
	"atlas-account/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	BansCheck = "bans/check?ip=%s&hwid=%s&accountId=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("BANS")
}

func requestCheckBan(ip string, hwid string, accountId uint32) requests.Request[CheckRestModel] {
	return rest.MakeGetRequest[CheckRestModel](fmt.Sprintf(getBaseRequest()+BansCheck, ip, hwid, accountId))
}
