package worldbroadcast

const (
	FamilyTV     = "TV"
	FamilyAvatar = "AVATAR"
)

// RestModel is the JSON:API wire representation of a GET on one
// (worldId, family) broadcast queue, returned by atlas-world's
// broadcast-queues resource (broadcast/rest.go, task-123 Task 9). Id is
// the family (TV|AVATAR) - together with the world-scoped route this
// uniquely identifies the resource. Unlike the world-side RestModel
// (marshal-only, produced from a domain QueueModel), this side needs
// SetID because it is the response decoder for requests.GetRequest.
type RestModel struct {
	Id                     string `json:"-"`
	Family                 string `json:"family"`
	ActiveRemainingSeconds uint32 `json:"activeRemainingSeconds"`
	PendingCount           int    `json:"pendingCount"`
	WaitSeconds            uint32 `json:"waitSeconds"`
}

func (r RestModel) GetName() string {
	return "broadcast-queues"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}
