package equipment

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource      = "data/equipment"
	EquipmentById = Resource + "/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

// RequestById returns a request to fetch equipment data by template ID from
// the atlas-data service. Tenant header propagation is handled by the request
// decorator chain.
func RequestById(ctx context.Context, templateId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+"/"+EquipmentById, templateId))
}
