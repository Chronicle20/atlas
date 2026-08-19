package parcel

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	// parcelsByRecipientResource lists a recipient's pending mailbox in a
	// world (services/atlas-parcel/atlas.com/parcel/parcel/resource.go
	// handleGetParcels). filter[worldId] is REQUIRED alongside
	// filter[recipientId] — world 0 is an ordinary real world, not a
	// sentinel default, and atlas-parcel rejects a recipient filter with no
	// world filter outright.
	parcelsByRecipientResource = "parcels?filter[recipientId]=%d&filter[worldId]=%d&filter[status]=pending"
	// parcelResource fetches a single parcel by id.
	parcelResource = "parcels/%s"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "PARCEL")
}

// requestForRecipient is the mailbox-capacity pre-flight (design §6.2) — the
// only remote call besides recipient resolution the Duey send flow makes.
func requestForRecipient(ctx context.Context, recipientId uint32, worldId world.Id) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	url := fmt.Sprintf(root+parcelsByRecipientResource, recipientId, byte(worldId))
	return requests.GetRequest[[]RestModel](url)
}

// requestById retrieves a single parcel by id (Task 18's receive/discard
// arms).
func requestById(ctx context.Context, parcelId uuid.UUID) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	url := fmt.Sprintf(root+parcelResource, parcelId.String())
	return requests.GetRequest[RestModel](url)
}

// discardRestModel is the PATCH /parcels/{id} request body (design §4.4 /
// §7.3) — recipientId lets atlas-parcel reject a discard issued by anyone
// but the parcel's own recipient, mirroring the authorization
// atlas-parcel's custody Discard method already performs for the (unused
// here) Kafka path.
type discardRestModel struct {
	Id          string `json:"-"`
	RecipientId uint32 `json:"recipientId"`
}

func (r discardRestModel) GetName() string { return "parcels" }

func (r discardRestModel) GetID() string { return r.Id }

func (r *discardRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// Required JSON:API relationship stubs — see RestModel's identical comment.
func (r *discardRestModel) SetToOneReferenceID(_, _ string) error { return nil }

func (r *discardRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// discardRequest marks parcelId discarded on behalf of recipientId.
func discardRequest(ctx context.Context, parcelId uuid.UUID, recipientId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	url := fmt.Sprintf(root+parcelResource, parcelId.String())
	return requests.PatchRequest[RestModel](url, discardRestModel{Id: parcelId.String(), RecipientId: recipientId})
}
