package data

import (
	"atlas-npc-conversations/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

// RequestFaceById returns a request to get face data by ID
// Calls GET /data/cosmetics/faces/{faceId}
func RequestFaceById(faceId uint32) requests.Request[FaceRestModel] {
	return rest.MakeGetRequest[FaceRestModel](fmt.Sprintf(getBaseRequest()+"/data/cosmetics/faces/%d", faceId))
}

// RequestHairById returns a request to get hair data by ID
// Calls GET /data/cosmetics/hairs/{hairId}
func RequestHairById(hairId uint32) requests.Request[HairRestModel] {
	return rest.MakeGetRequest[HairRestModel](fmt.Sprintf(getBaseRequest()+"/data/cosmetics/hairs/%d", hairId))
}
