package door

import (
	"context"
	"errors"

	mapdata "atlas-doors/data/map"
	skilldata "atlas-doors/data/skill"
	"atlas-doors/party"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

// doorPortalType is the atlas-data portal type for town-door portals (Type==6).
const doorPortalType uint8 = 6

var (
	errNoValidReturn = errors.New("map has no valid return destination")
	errTownMap       = errors.New("mystic door cannot be cast in a town")
	errFieldLimit    = errors.New("map field limit forbids mystic door")
)

// restResolver wires the data/map, data/skill, and party REST processors to
// produce a spawnPlan. It is the production implementation of the resolver seam.
type restResolver struct {
	l     logrus.FieldLogger
	ctx   context.Context
	maps  mapdata.Processor
	skill skilldata.Processor
	party party.Processor
}

func newRestResolver(l logrus.FieldLogger, ctx context.Context) resolver {
	return &restResolver{
		l:     l,
		ctx:   ctx,
		maps:  mapdata.NewProcessor(l, ctx),
		skill: skilldata.NewProcessor(l, ctx),
		party: party.NewProcessor(l, ctx),
	}
}

// PartyIdFor returns the caster's party id, or 0 when not in a party.
func (r *restResolver) PartyIdFor(_ context.Context, ownerCharacterId uint32) (uint32, error) {
	pm, err := r.party.GetByMemberId(ownerCharacterId)
	if err != nil {
		return 0, nil
	}
	return pm.Id(), nil
}

// ResolveSpawn fetches the source map metadata, defensively re-checks the spawn
// is permitted (the channel pre-checks too), resolves the town map and its door
// portals, computes the caster's party slot, and reads the skill effect duration.
func (r *restResolver) ResolveSpawn(_ context.Context, f field.Model, ownerCharacterId, partyId, skillId uint32, level byte) (spawnPlan, error) {
	srcMap, err := r.maps.GetById(f.MapId())
	if err != nil {
		return spawnPlan{}, err
	}

	// Defensive re-checks (the channel pre-checks these too).
	if srcMap.Town() {
		return spawnPlan{}, errTownMap
	}
	if srcMap.FieldLimit()&_map.FieldLimitNoMysticDoor != 0 {
		return spawnPlan{}, errFieldLimit
	}
	if !HasValidReturn(srcMap.ReturnMapId(), srcMap.ForcedReturnMapId()) {
		return spawnPlan{}, errNoValidReturn
	}

	townMapId := ResolveTownMap(srcMap.ReturnMapId(), srcMap.ForcedReturnMapId())

	// Fetch the town map's door-type portals (Type==6) in load order.
	townMap, err := r.maps.GetById(townMapId)
	if err != nil {
		return spawnPlan{}, err
	}
	doorPortals := make([]TownPortal, 0)
	for _, pt := range townMap.Portals() {
		if pt.Type() == doorPortalType {
			doorPortals = append(doorPortals, TownPortal{X: pt.X(), Y: pt.Y()})
		}
	}

	// Party members (ordered) for slot computation; solo casters have none.
	var members []uint32
	if partyId != 0 {
		if pm, perr := r.party.GetByMemberId(ownerCharacterId); perr == nil {
			members = pm.Members()
		}
	}
	slot := ComputeSlot(partyId, members, ownerCharacterId)
	wireId, tx, ty, _ := ResolveTownPortal(doorPortals, slot, defaultTownX, defaultTownY)

	// Skill effect duration; -1 / <=0 means "no expiry" → durationMs 0.
	var durationMs int32
	if eff, eerr := r.skill.GetEffect(skillId, level); eerr == nil {
		if d := eff.Duration(); d > 0 {
			durationMs = d
		}
	}

	return spawnPlan{
		townMapId:    townMapId,
		slot:         slot,
		townPortalId: wireId,
		townX:        tx,
		townY:        ty,
		durationMs:   durationMs,
	}, nil
}
