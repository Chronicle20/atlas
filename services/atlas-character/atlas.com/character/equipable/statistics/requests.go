package statistics

import (
	"atlas-character/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	equipmentResource = "equipables"
	equipResource     = equipmentResource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("EQUIPABLES")
}

func requestCreate(itemId uint32) requests.Request[RestModel] {
	input := &RestModel{
		ItemId: itemId,
	}
	return rest.MakePostRequest[RestModel](getBaseRequest()+equipmentResource, input)
}

func requestById(equipmentId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+equipResource, equipmentId))
}

func updateById(equipmentId uint32, i RestModel) requests.Request[RestModel] {
	return rest.MakePatchRequest[RestModel](fmt.Sprintf(getBaseRequest()+equipResource, equipmentId), i)
}

func deleteById(equipmentId uint32) requests.EmptyBodyRequest {
	return rest.MakeDeleteRequest(fmt.Sprintf(getBaseRequest()+equipResource, equipmentId))
}
