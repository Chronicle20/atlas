package data

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

// RequestFaceById returns a request to get face data by ID
// Calls GET /data/cosmetics/faces/{faceId}
func RequestFaceById(ctx context.Context, faceId uint32) requests.Request[FaceRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[FaceRestModel](err)
	}
	return requests.GetRequest[FaceRestModel](fmt.Sprintf(root+"/data/cosmetics/faces/%d", faceId))
}

// RequestHairById returns a request to get hair data by ID
// Calls GET /data/cosmetics/hairs/{hairId}
func RequestHairById(ctx context.Context, hairId uint32) requests.Request[HairRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[HairRestModel](err)
	}
	return requests.GetRequest[HairRestModel](fmt.Sprintf(root+"/data/cosmetics/hairs/%d", hairId))
}
