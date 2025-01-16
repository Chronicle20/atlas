package buddylist

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "characters/%d/buddy-list"
)

func getBaseRequest() string {
	return "http://127.0.0.1:8080/api/"
}

func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, id))
}
