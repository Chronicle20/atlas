package buff

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvEventStatusTopic        = "EVENT_TOPIC_CHARACTER_BUFF_STATUS"
	EventStatusTypeBuffApplied = "APPLIED"
	EventStatusTypeBuffExpired = "EXPIRED"
)

type StatusEvent[E any] struct {
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	CharacterId uint32     `json:"characterId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
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
	StatTypeCurse      = "CURSE"       // EXP rate debuff (fixed: amount ignored, canonical v83 multiplier 0.5)
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

	// ConversionFixed: multiplier = mapping.Multiplier (amount ignored)
	// Example: CURSE -> 0.5x flat
	ConversionFixed
)

// RateMapping defines how a buff stat type maps to a rate type
type RateMapping struct {
	RateType   string           // The rate type (e.g., "exp", "meso")
	Conversion ConversionMethod // How to convert amount to multiplier
	Multiplier float64          // Used only when Conversion == ConversionFixed; ignored otherwise
}

// buffToRateMappings maps game client stat types to rate types with conversion methods
var buffToRateMappings = map[string]RateMapping{
	StatTypeHolySymbol: {RateType: "exp", Conversion: ConversionAdditive},
	StatTypeMesoUp:     {RateType: "meso", Conversion: ConversionDirect},
	StatTypeCurse:      {RateType: "exp", Conversion: ConversionFixed, Multiplier: 0.5},
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

// CalculateMultiplier converts a stat amount to a rate multiplier using the mapping's conversion method.
// For Amount-derived modes (Additive, Direct), `amount` is consumed.
// For Fixed mode, `mapping.Multiplier` is returned verbatim and `amount` is ignored.
func CalculateMultiplier(amount int32, mapping RateMapping) float64 {
	switch mapping.Conversion {
	case ConversionAdditive:
		return 1.0 + (float64(amount) / 100.0)
	case ConversionDirect:
		return float64(amount) / 100.0
	case ConversionFixed:
		return mapping.Multiplier
	default:
		return 1.0
	}
}
