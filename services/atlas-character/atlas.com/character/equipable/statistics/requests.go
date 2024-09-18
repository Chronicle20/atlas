package statistics

import (
	"atlas-character/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	equipmentResource = "equipment"
	equipResource     = equipmentResource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("EQUIPABLE_SERVICE_URL")
}

func requestCreate(itemId uint32) requests.Request[RestModel] {
	input := &RestModel{
		ItemId: itemId,
	}
	return rest.MakePostRequest[RestModel](fmt.Sprintf(getBaseRequest()+equipmentResource), input)
}

func requestById(equipmentId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+equipResource, equipmentId))
}

func deleteById(equipmentId uint32) requests.EmptyBodyRequest {
	return rest.MakeDeleteRequest(fmt.Sprintf(getBaseRequest()+equipResource, equipmentId))
}
