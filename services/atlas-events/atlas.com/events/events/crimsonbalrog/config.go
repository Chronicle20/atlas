// Package crimsonbalrog implements the Crimson Balrog boat-attack event
// (design §15.5, PRD F6): a chance for a Crimson Balrog to spawn during an
// Orbis/Ellinia boat crossing. Every gameplay value is configuration (FR-B1) —
// this package contains no magic numbers outside the seed JSON.
package crimsonbalrog

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TypeName is the definition type this handler serves, and the registry key.
const TypeName = "CRIMSON_BALROG"

// Position is a single spawn location within an attack map.
type Position struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

// AttackMap is one deck (attack map) an occurrence may spawn on, along with
// the spawn positions available on it. Modelled per map rather than as one
// flat position because two decks (To Orbis vs To Ellinia) have different
// geometry (FR-B1).
type AttackMap struct {
	MapId          _map.Id    `json:"mapId"`
	SpawnPositions []Position `json:"spawnPositions"`
}

// VisualConfig names the show/hide state pair the client uses to render the
// Balrog's boat-shake visual effect.
type VisualConfig struct {
	Name         string `json:"name"`
	ShowState    byte   `json:"showState"`
	ShowSubState byte   `json:"showSubState"`
	HideState    byte   `json:"hideState"`
	HideSubState byte   `json:"hideSubState"`
}

// Config is the CRIMSON_BALROG definition's configuration.
//
// ApplicableRouteIds holds route SLUGS, not uuids. A route's uuid is
// tenant-derived (tenant.DerivedId(t.Id(), "routes", slug) —
// atlas-transports/transport/config/rest.go:79-88) so it differs per tenant
// and cannot be hard-coded in a static seed file. Route seeds are themselves
// identified by slug (deploy/seed/shared/all/routes/*.json "id"), and slug is
// the repo's established cross-reference idiom for this
// (deploy/seed/shared/all/vessels/*.json "routeAID"/"routeBID").
type Config struct {
	ApplicableRouteIds []string     `json:"applicableRouteIds"`
	TriggerDelay       Duration     `json:"triggerDelay"`
	TriggerDelayJitter Duration     `json:"triggerDelayJitter"`
	AttackProbability  float64      `json:"attackProbability"`
	MonsterId          uint32       `json:"monsterId"`
	MonsterCount       uint32       `json:"monsterCount"`
	AttackMaps         []AttackMap  `json:"attackMaps"`
	RelatedMapIds      []_map.Id    `json:"relatedMapIds"`
	BackgroundMusic    string       `json:"backgroundMusic"`
	Visual             VisualConfig `json:"visual"`
}

// Duration decodes a Go duration string ("3m", "60s") from JSON.
type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// AsDuration returns the underlying time.Duration.
func (d Duration) AsDuration() time.Duration { return time.Duration(d) }

// DecodeConfig unmarshals a raw configuration payload into Config.
func DecodeConfig(raw json.RawMessage) (Config, error) {
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("crimsonbalrog: decode configuration: %w", err)
	}
	return c, nil
}

// Validate rejects a configuration this handler cannot interpret (FR-D6). Each
// error names its field so the JSON:API error an administrator sees is
// actionable. Every bound here exists because violating it would fail LATER, at
// trigger time, where nobody is watching.
func (c Config) Validate() error {
	if len(c.ApplicableRouteIds) == 0 {
		return errors.New("applicableRouteIds: must contain at least one route")
	}
	for _, slug := range c.ApplicableRouteIds {
		if slug == "" {
			return errors.New("applicableRouteIds: entries must not be empty")
		}
	}
	if c.AttackProbability < 0 || c.AttackProbability > 1 {
		return fmt.Errorf("attackProbability: must be in [0,1], got %v", c.AttackProbability)
	}
	if c.MonsterCount == 0 {
		return errors.New("monsterCount: must be greater than zero")
	}
	if c.TriggerDelay < 0 || c.TriggerDelayJitter < 0 {
		return errors.New("triggerDelay/triggerDelayJitter: must not be negative")
	}
	if len(c.AttackMaps) == 0 {
		return errors.New("attackMaps: must contain at least one map")
	}
	for _, am := range c.AttackMaps {
		if uint32(len(am.SpawnPositions)) < c.MonsterCount {
			return fmt.Errorf("attackMaps[%d].spawnPositions: %d positions for monsterCount %d", am.MapId, len(am.SpawnPositions), c.MonsterCount)
		}
	}
	if c.Visual.Name == "" {
		return errors.New("visual.name: must be set")
	}
	return nil
}

// WorkContext is the per-occurrence context a Crimson Balrog work row carries:
// which voyage, which route, which world/channel, and the maps involved.
// RouteId stays uuid.UUID — that is what the transport StatusEvent Kafka
// message actually carries (atlas-transports/kafka/message/transport/kafka.go).
type WorkContext struct {
	VoyageId         uuid.UUID  `json:"voyageId"`
	RouteId          uuid.UUID  `json:"routeId"`
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	StagingMapId     _map.Id    `json:"stagingMapId"`
	DestinationMapId _map.Id    `json:"destinationMapId"`
	ObservationMapId _map.Id    `json:"observationMapId"`
	EnRouteMapIds    []_map.Id  `json:"enRouteMapIds"`
	DepartedAt       time.Time  `json:"departedAt"`
}
