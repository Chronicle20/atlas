package consumable

const (
	EnvEventTopic   = "EVENT_TOPIC_CONSUMABLE_STATUS"
	EventTypeError  = "ERROR"
	EventTypeScroll = "SCROLL"

	ErrorTypePetCannotConsume = "PET_CANNOT_CONSUME"
)

type Event[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type ErrorBody struct {
	Error string `json:"error"`
}

type ScrollBody struct {
	Success         bool `json:"success"`
	Cursed          bool `json:"cursed"`
	LegendarySpirit bool `json:"legendarySpirit"`
	WhiteScroll     bool `json:"whiteScroll"`
}
