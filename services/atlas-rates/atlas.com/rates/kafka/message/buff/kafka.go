package buff

import "time"

const (
	EnvEventStatusTopic        = "EVENT_TOPIC_CHARACTER_BUFF_STATUS"
	EventStatusTypeBuffApplied = "APPLIED"
	EventStatusTypeBuffExpired = "EXPIRED"
)

type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

type AppliedStatusEventBody struct {
	FromId    uint32       `json:"fromId"`
	SourceId  int32        `json:"sourceId"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}

type ExpiredStatusEventBody struct {
	SourceId  int32        `json:"sourceId"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}

// Game client stat types that affect rates
// These are TemporaryStatType constants from the game client, not rate-specific types
const (
	StatTypeHolySymbol = "HOLY_SYMBOL" // EXP rate buff (additive: amount is bonus percentage)
	StatTypeMesoUp     = "MESO_UP"     // Meso rate buff (direct: amount is total percentage)
)

// ConversionMethod defines how to convert a stat amount to a rate multiplier
type ConversionMethod int

const (
	// ConversionAdditive: multiplier = 1.0 + (amount / 100.0)
	// Example: amount=50 -> 1.50x (50% bonus)
	ConversionAdditive ConversionMethod = iota

	// ConversionDirect: multiplier = amount / 100.0
	// Example: amount=103 -> 1.03x (103% of base)
	ConversionDirect
)

// RateMapping defines how a buff stat type maps to a rate type
type RateMapping struct {
	RateType   string           // The rate type (e.g., "exp", "meso")
	Conversion ConversionMethod // How to convert amount to multiplier
}

// buffToRateMappings maps game client stat types to rate types with conversion methods
var buffToRateMappings = map[string]RateMapping{
	StatTypeHolySymbol: {RateType: "exp", Conversion: ConversionAdditive},
	StatTypeMesoUp:     {RateType: "meso", Conversion: ConversionDirect},
}

// IsRateStatType checks if a stat change type affects rates
func IsRateStatType(statType string) bool {
	_, exists := buffToRateMappings[statType]
	return exists
}

// GetRateMapping returns the rate mapping for a stat type, if it exists
func GetRateMapping(statType string) (RateMapping, bool) {
	mapping, exists := buffToRateMappings[statType]
	return mapping, exists
}

// CalculateMultiplier converts a stat amount to a rate multiplier using the specified conversion method
func CalculateMultiplier(amount int32, conversion ConversionMethod) float64 {
	switch conversion {
	case ConversionAdditive:
		return 1.0 + (float64(amount) / 100.0)
	case ConversionDirect:
		return float64(amount) / 100.0
	default:
		return 1.0
	}
}
