package food

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// EnvEventTopic is the topic carrying taming-mob food (feed) events produced by
// atlas-consumables (Task 33). The producer MUST populate worldId, itemId, and
// tirednessHeal so atlas-mounts can apply the feed math and emit the FEED event.
const EnvEventTopic = "EVENT_TOPIC_TAMING_MOB_FOOD"

// Event is the taming-mob food event consumed by atlas-mounts. worldId is
// required: ApplyFeedAndEmit needs it to emit the resulting mount status event,
// and this event is the only source of it. This struct is the cross-service
// contract — the consumables producer must match its field names and json tags.
type Event struct {
	WorldId       world.Id `json:"worldId"`
	CharacterId   uint32   `json:"characterId"`
	ItemId        uint32   `json:"itemId"`
	TirednessHeal int32    `json:"tirednessHeal"`
}
