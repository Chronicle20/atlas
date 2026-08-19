package crimsonbalrog

import (
	"atlas-events/event/registry"
	"atlas-events/external/transports"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// StageAttacking is the only stage this occurrence type reaches (Tasks
// 25/27 own its progression); Evaluate always seeds a new occurrence at this
// stage.
const StageAttacking = "ATTACKING"

// OccurrenceContext is the context payload of a CRIMSON_BALROG occurrence.
// It carries everything Start/Advance/Complete (Tasks 25, 27) need without a
// follow-up query: route/voyage identity, world/channel scope, the resolved
// attack and related maps (with their spawn positions), the monster to spawn
// and how many, and the visual/BGM the projection reads (FR-B9, FR-B10).
type OccurrenceContext struct {
	RouteId         uuid.UUID    `json:"routeId"`
	VoyageId        uuid.UUID    `json:"voyageId"`
	WorldId         world.Id     `json:"worldId"`
	ChannelId       channel.Id   `json:"channelId"`
	AttackMaps      []AttackMap  `json:"attackMaps"`
	RelatedMapIds   []_map.Id    `json:"relatedMapIds"`
	MonsterId       monster.Id   `json:"monsterId"`
	MonsterCount    uint32       `json:"monsterCount"`
	BackgroundMusic string       `json:"backgroundMusic"`
	Visual          VisualConfig `json:"visual"`
}

// EncodeOccurrenceContext marshals an OccurrenceContext for storage on
// registry.Seed.Context / occurrence.Model.Context.
func EncodeOccurrenceContext(oc OccurrenceContext) (json.RawMessage, error) {
	raw, err := json.Marshal(oc)
	if err != nil {
		return nil, fmt.Errorf("crimsonbalrog: encode occurrence context: %w", err)
	}
	return raw, nil
}

// DecodeOccurrenceContext unmarshals an occurrence's stored context. Used by
// Start/Advance/Complete (Tasks 25, 27).
func DecodeOccurrenceContext(raw json.RawMessage) (OccurrenceContext, error) {
	var oc OccurrenceContext
	if err := json.Unmarshal(raw, &oc); err != nil {
		return OccurrenceContext{}, fmt.Errorf("crimsonbalrog: decode occurrence context: %w", err)
	}
	return oc, nil
}

// Gate names carried on the `gate` field of every
// crimsonbalrog.evaluate_rejected log line. They name WHICH of Evaluate's
// silent (nil, nil) returns fired: without them a lost roll, an empty boat,
// and a disabled definition are indistinguishable in the
// scheduled_event_work table, which records only that the row COMPLETED.
const (
	gateNoVoyage     = "no_voyage"
	gateNotUnderway  = "voyage_not_underway"
	gateNotEnabled   = "definition_disabled"
	gateRoll         = "roll_failed"
	gateNobodyAboard = "nobody_aboard"
)

// evalLog tags an evaluation's log lines with the identity a reader needs to
// correlate a decision back to its scheduled_event_work row: the definition
// and the voyage are exactly the discriminating parts of that row's
// dedupe_key.
func (h *Handler) evalLog(d registry.Definition, wc WorkContext) logrus.FieldLogger {
	return h.l.WithFields(logrus.Fields{
		"definitionId": d.Id,
		"routeId":      wc.RouteId,
		"voyageId":     wc.VoyageId,
		"worldId":      wc.WorldId,
		"channelId":    wc.ChannelId,
	})
}

// Evaluate applies FR-B5's gates in order, cheapest first. Returning
// (nil, nil) is the ORDINARY "no occurrence" outcome (FR-B7, FR-B8), not an
// error: the work row completes and no history is written, which is what
// keeps the occurrence table a record of real events.
//
// Returning an ERROR is reserved for "we could not tell" — an unreachable
// atlas-transports or atlas-maps. The work row then retries. Reading an
// unreachable atlas-maps as "nobody aboard" would silently suppress attacks
// during any maps outage.
func (h *Handler) Evaluate(ctx context.Context, d registry.Definition, w registry.Work) (*registry.Seed, error) {
	c, err := DecodeConfig(d.Configuration)
	if err != nil {
		return nil, err
	}
	var wc WorkContext
	if err := json.Unmarshal(w.Context, &wc); err != nil {
		return nil, err
	}

	// Enabling a definition schedules one generic TRIGGER_EVALUATION with an
	// empty work context (event/orchestration.SetEnabled). CRIMSON_BALROG is
	// externally triggered by VOYAGE_DEPARTED (trigger.go) — an
	// enable-triggered evaluation carries no voyage and means nothing here.
	// Not an error: the work row completes normally, same as any other
	// ordinary "no occurrence" outcome (FR-B7, FR-B8).
	if wc.VoyageId == uuid.Nil {
		h.evalLog(d, wc).WithField("gate", gateNoVoyage).Info("crimsonbalrog.evaluate_rejected")
		return nil, nil
	}

	l := h.evalLog(d, wc)

	// 1. Is the voyage still underway?
	route, err := h.transports(ctx).GetRoute(wc.RouteId)
	if err != nil {
		return nil, err
	}
	if !transports.VoyageUnderway(route, wc.VoyageId) {
		l.WithFields(logrus.Fields{
			"gate":        gateNotUnderway,
			"routeState":  route.State,
			"routeVoyage": route.VoyageID,
		}).Info("crimsonbalrog.evaluate_rejected")
		return nil, nil
	}

	// 2. Is the definition still enabled? (It may have been disabled between
	//    departure and now — FR-D4.)
	if !d.Enabled {
		l.WithField("gate", gateNotEnabled).Info("crimsonbalrog.evaluate_rejected")
		return nil, nil
	}

	// 3. The roll. The rolled value and the threshold both ride on the log
	//    line: "the roll missed" is only a satisfying answer if you can see
	//    what it missed by, and a definition patched to a lower probability
	//    than the operator remembers looks identical otherwise.
	rolled := h.roll()
	if rolled >= c.AttackProbability {
		l.WithFields(logrus.Fields{
			"gate":              gateRoll,
			"rolled":            rolled,
			"attackProbability": c.AttackProbability,
		}).Info("crimsonbalrog.evaluate_rejected")
		return nil, nil
	}

	// 4. Is anyone aboard? The UNION of attack and related maps: a character
	//    in the cabin counts (FR-B6).
	aboard, err := h.anyoneAboard(ctx, c, wc)
	if err != nil {
		return nil, err
	}
	if !aboard {
		l.WithFields(logrus.Fields{
			"gate":          gateNobodyAboard,
			"attackMaps":    mapIds(c.AttackMaps),
			"relatedMapIds": c.RelatedMapIds,
		}).Info("crimsonbalrog.evaluate_rejected")
		return nil, nil
	}

	l.WithField("rolled", rolled).Info("crimsonbalrog.evaluate_seeded")

	key, err := h.ConcurrencyKey(ctx, w.Context)
	if err != nil {
		return nil, err
	}
	return h.seed(c, wc, key)
}

// mapIds projects the attack maps down to the ids alone, for the
// nobody_aboard log line: the spawn positions are noise there, and what a
// reader wants is which maps were actually queried.
func mapIds(ams []AttackMap) []_map.Id {
	ids := make([]_map.Id, 0, len(ams))
	for _, am := range ams {
		ids = append(ids, am.MapId)
	}
	return ids
}

// anyoneAboard reports whether any character is present in the UNION of the
// attack maps and the related maps (FR-B6: a character in the cabin counts).
// It short-circuits as soon as any map is non-empty, so a positive answer
// does not pay for every map query.
func (h *Handler) anyoneAboard(ctx context.Context, c Config, wc WorkContext) (bool, error) {
	mp := h.maps(ctx)

	for _, am := range c.AttackMaps {
		ids, err := mp.CharacterIdsInMap(field.NewBuilder(wc.WorldId, wc.ChannelId, am.MapId).Build())
		if err != nil {
			return false, err
		}
		if len(ids) > 0 {
			return true, nil
		}
	}
	for _, mapId := range c.RelatedMapIds {
		ids, err := mp.CharacterIdsInMap(field.NewBuilder(wc.WorldId, wc.ChannelId, mapId).Build())
		if err != nil {
			return false, err
		}
		if len(ids) > 0 {
			return true, nil
		}
	}
	return false, nil
}

// seed builds the registry.Seed for a new occurrence: the occurrence context
// (route/voyage/world/channel/attack maps/related maps/monsterId/
// monsterCount/resolved spawn positions plus the visual and BGM the
// projection reads — FR-B10), and the map scope rows: attack maps
// Visual: true, related maps (cabins) Visual: false (FR-B9, FR-B13).
func (h *Handler) seed(c Config, wc WorkContext, concurrencyKey string) (*registry.Seed, error) {
	oc := OccurrenceContext{
		RouteId:         wc.RouteId,
		VoyageId:        wc.VoyageId,
		WorldId:         wc.WorldId,
		ChannelId:       wc.ChannelId,
		AttackMaps:      c.AttackMaps,
		RelatedMapIds:   c.RelatedMapIds,
		MonsterId:       c.MonsterId,
		MonsterCount:    c.MonsterCount,
		BackgroundMusic: c.BackgroundMusic,
		Visual:          c.Visual,
	}
	raw, err := EncodeOccurrenceContext(oc)
	if err != nil {
		return nil, err
	}

	scope := make([]registry.MapScope, 0, len(c.AttackMaps)+len(c.RelatedMapIds))
	for _, am := range c.AttackMaps {
		scope = append(scope, registry.MapScope{MapId: am.MapId, Visual: true})
	}
	for _, mapId := range c.RelatedMapIds {
		scope = append(scope, registry.MapScope{MapId: mapId, Visual: false})
	}

	return &registry.Seed{
		Stage:          StageAttacking,
		Context:        raw,
		WorldId:        wc.WorldId,
		ChannelId:      wc.ChannelId,
		VoyageId:       wc.VoyageId,
		ConcurrencyKey: concurrencyKey,
		Maps:           scope,
	}, nil
}
