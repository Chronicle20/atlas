package pending_change

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const BaseUrl = "CHARACTER_URL"

func getBaseRequest() string {
	return requests.RootUrl(BaseUrl)
}

func changeWorldRequest(characterId uint32, body WorldChangeInputRestModel) requests.Request[struct{}] {
	url := fmt.Sprintf("%scharacters/%d/world-change", getBaseRequest(), characterId)
	return requests.PostRequest[struct{}](url, body)
}

func resolveRequest(characterId uint32, id string, body ResolveInputRestModel) requests.Request[struct{}] {
	url := fmt.Sprintf("%scharacters/%d/pending-changes/%s/resolve", getBaseRequest(), characterId, id)
	return requests.PostRequest[struct{}](url, body)
}

func transferEligibilityRequest(characterId uint32, destinationWorldId world.Id) requests.Request[EligibilityRestModel] {
	url := fmt.Sprintf("%scharacters/%d/transfer-eligibility?destinationWorldId=%d", getBaseRequest(), characterId, destinationWorldId)
	return requests.GetRequest[EligibilityRestModel](url)
}
