package buddylist

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "characters/%d/buddy-list"
)

func getBaseRequest() string {
	return requests.RootUrl("BUDDIES")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, id))
}
