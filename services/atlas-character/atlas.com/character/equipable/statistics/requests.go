package statistics

import (
	"atlas-character/rest/requests"
	"fmt"
	"os"
)

const (
	equipmentResource = "equipment"
	equipResource     = equipmentResource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("EQUIPMENT_SERVICE_URL")
}

func requestCreate(itemId uint32) requests.PostRequest[RestModel] {
	input := &RestModel{
		ItemId: itemId,
	}
	return requests.MakePostRequest[RestModel](fmt.Sprintf(getBaseRequest()+equipmentResource), input)
}

func requestById(equipmentId uint32) requests.Request[RestModel] {
	return requests.MakeGetRequest[RestModel](fmt.Sprintf(equipResource, equipmentId))
}
