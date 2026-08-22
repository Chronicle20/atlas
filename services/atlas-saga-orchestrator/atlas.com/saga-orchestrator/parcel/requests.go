package parcel

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	// parcelResource fetches a single parcel by id. atlas-parcel exposes
	// GET /parcels/{parcelId} (services/atlas-parcel/atlas.com/parcel/parcel/resource.go).
	parcelResource = "parcels/%s"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "PARCEL")
}

// RequestParcel retrieves a single parcel by id from atlas-parcel so
// expandWithdrawFromParcel can capture the item snapshot for the
// accept_to_character step. A single GET (not a paginated drain), mirroring
// compartment.RequestCompartment.
func RequestParcel(l logrus.FieldLogger, ctx context.Context) func(parcelId uuid.UUID) (RestModel, error) {
	return func(parcelId uuid.UUID) (RestModel, error) {
		root, err := getBaseRequest(ctx)
		if err != nil {
			return RestModel{}, err
		}
		url := fmt.Sprintf(root+parcelResource, parcelId.String())
		return requests.GetRequest[RestModel](url)(l, ctx)
	}
}
