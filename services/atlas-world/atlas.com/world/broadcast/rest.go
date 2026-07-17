package broadcast

import (
	"math"
	"time"
)

// RestModel is the JSON:API resource for a GET on one (worldId, family)
// broadcast queue. Id is the family (TV|AVATAR): together with the
// world-scoped route this uniquely identifies the resource.
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

// Transform maps a domain QueueModel snapshot to its REST representation at
// instant now. ActiveRemainingSeconds is the time left on the Active entry
// (rounded up to the next whole second, floored at 0 for an already-expired
// but not-yet-swept entry); PendingCount is the queue depth behind Active;
// WaitSeconds is QueueModel.WaitSeconds(now) - the same estimate a
// newly-enqueued entry would be given.
func Transform(family string, q QueueModel, now time.Time) (RestModel, error) {
	var activeRemainingSeconds uint32
	if q.Active != nil {
		remaining := q.Active.ExpiresAt.Sub(now)
		if remaining > 0 {
			activeRemainingSeconds = uint32(math.Ceil(remaining.Seconds()))
		}
	}

	return RestModel{
		Id:                     family,
		Family:                 family,
		ActiveRemainingSeconds: activeRemainingSeconds,
		PendingCount:           len(q.Pending),
		WaitSeconds:            q.WaitSeconds(now),
	}, nil
}
