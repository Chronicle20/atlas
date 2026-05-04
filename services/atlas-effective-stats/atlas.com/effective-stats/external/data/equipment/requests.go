package equipment

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource      = "data/equipment"
	EquipmentById = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

// RequestById returns a request to fetch equipment data by template ID from
// the atlas-data service. Tenant header propagation is handled by the request
// decorator chain.
func RequestById(templateId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+"/"+EquipmentById, templateId))
}
