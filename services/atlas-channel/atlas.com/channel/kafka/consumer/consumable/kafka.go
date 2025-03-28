package consumable

const (
	EnvEventTopic   = "EVENT_TOPIC_CONSUMABLE_STATUS"
	EventTypeError  = "ERROR"
	EventTypeScroll = "SCROLL"
)

type Event[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type ErrorBody struct {
}
type ScrollBody struct {
	Success         bool `json:"success"`
	Cursed          bool `json:"cursed"`
	LegendarySpirit bool `json:"legendarySpirit"`
	WhiteScroll     bool `json:"whiteScroll"`
}
