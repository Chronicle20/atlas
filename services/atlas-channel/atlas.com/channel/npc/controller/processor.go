package controller

import (
	"atlas-channel/character/buff"
	_map "atlas-channel/map"
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	TryClaim(f field.Model, npcObjectId uint32, characterId uint32) (bool, error)
	ReleaseFor(f field.Model, characterId uint32) ([]uint32, error)
	ElectFor(f field.Model, npcObjectIds []uint32, exclude ...uint32) (map[uint32]uint32, error)
	UncontrolledIn(f field.Model, npcObjectIds []uint32) ([]uint32, error)
}

// ProcessorImpl decides NPC-controller assignments. Constructed one per
// handler/sweep invocation, matching the codebase's per-invocation
// processor idiom — but a single instance IS safe to share across
// goroutines that call TryClaim/isHidden concurrently (task-176:
// data/npc.ForEachInMap fans the per-map NPC sweep out across one
// goroutine per NPC via model.ParallelExecute(), and spawnNPCForSession
// intentionally builds one Processor per session sweep so the hidden
// winner-check is memoized across all of them). hiddenCacheMu guards
// hiddenCache against the resulting concurrent reads/writes.
type ProcessorImpl struct {
	l             logrus.FieldLogger
	ctx           context.Context
	t             tenant.Model
	fieldIdsFn    func(f field.Model) ([]uint32, error)
	hiddenFn      func(characterId uint32) bool
	hiddenCacheMu sync.Mutex
	hiddenCache   map[uint32]bool
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:           l,
		ctx:         ctx,
		t:           tenant.MustFromContext(ctx),
		hiddenCache: make(map[uint32]bool),
	}
	p.fieldIdsFn = func(f field.Model) ([]uint32, error) {
		return _map.NewProcessor(l, ctx).GetCharacterIdsInMap(f)
	}
	// Winner-check (design §3.2): fetch ONE candidate's buffs from
	// atlas-buffs and test IsGmHidden. Fail-open: an unreachable buffs
	// service must not stall NPC control, so errors read as "not hidden".
	p.hiddenFn = func(characterId uint32) bool {
		bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
		if err != nil {
			l.WithError(err).Warnf("Unable to winner-check hide state of [%d]; treating as visible.", characterId)
			return false
		}
		return buff.IsGmHidden(bs)
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

// isHidden memoizes the hidden winner-check per characterId. Locked for the
// full check-then-fetch-then-store: the spawn-path sweep always calls this
// with the SAME characterId (the entering session's own character) across
// its parallel per-NPC goroutines, so serializing on that single key still
// yields exactly one atlas-buffs fetch — the lock adds no extra round trips,
// it just makes the memoization race-free (task-176 code review).
func (p *ProcessorImpl) isHidden(characterId uint32) bool {
	p.hiddenCacheMu.Lock()
	defer p.hiddenCacheMu.Unlock()
	if v, ok := p.hiddenCache[characterId]; ok {
		return v
	}
	v := p.hiddenFn(characterId)
	p.hiddenCache[characterId] = v
	return v
}

func contains(ids []uint32, id uint32) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// TryClaim is the map-enter (and reveal) claim path (FR-5.2). It returns
// true when the caller should send the controller grant to characterId:
// either this call won a fresh/stale-replacement claim, or characterId is
// already the recorded controller (grant re-issue — same rationale as the
// MonsterControl re-issue in spawnMonsterForSession).
func (p *ProcessorImpl) TryClaim(f field.Model, npcObjectId uint32, characterId uint32) (bool, error) {
	r := GetRegistry()
	if r == nil {
		return false, nil
	}
	cur, ok, err := r.ControllerOf(p.ctx, p.t, f, npcObjectId)
	if err != nil {
		return false, err
	}
	if ok {
		if cur == characterId {
			return true, nil
		}
		live, lerr := p.fieldIdsFn(f)
		if lerr != nil {
			return false, lerr
		}
		if contains(live, cur) {
			return false, nil
		}
		// Stale (controller no longer in the field — missed exit or crashed
		// pod): release, then race for it below. Concurrent enterers both
		// reach Claim; SetNX lets exactly one win.
		if derr := r.Release(p.ctx, p.t, f, npcObjectId); derr != nil {
			return false, derr
		}
	}
	if p.isHidden(characterId) {
		p.l.Debugf("Character [%d] is GM-hidden; not claiming NPC [%d] in field [%s].", characterId, npcObjectId, f.Id())
		return false, nil
	}
	return r.Claim(p.ctx, p.t, f, npcObjectId, characterId)
}

// ReleaseFor drops every controller entry held by characterId in f and
// returns the released NPC ids (FR-5.3 / FR-6.1).
func (p *ProcessorImpl) ReleaseFor(f field.Model, characterId uint32) ([]uint32, error) {
	r := GetRegistry()
	if r == nil {
		return nil, nil
	}
	ids, err := r.ControlledBy(p.ctx, p.t, f, characterId)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if err := r.Release(p.ctx, p.t, f, ids...); err != nil {
		return nil, err
	}
	p.l.Debugf("Released [%d] NPC controller entries held by character [%d] in field [%s].", len(ids), characterId, f.Id())
	return ids, nil
}

// ElectFor assigns a controller to each requested NPC using the same rule
// as monsters: least-loaded live session, hidden excluded (FR-5.2/FR-6.2),
// no forced transfer semantics — callers pass only NPCs known to need a
// controller. Returns npcId -> winner for announcement. NPCs that lose a
// SetNX race to a concurrent claim are simply omitted.
func (p *ProcessorImpl) ElectFor(f field.Model, npcObjectIds []uint32, exclude ...uint32) (map[uint32]uint32, error) {
	assignments := make(map[uint32]uint32)
	r := GetRegistry()
	if r == nil || len(npcObjectIds) == 0 {
		return assignments, nil
	}
	live, err := p.fieldIdsFn(f)
	if err != nil {
		return assignments, err
	}
	counts := make(map[uint32]int)
	for _, id := range live {
		if contains(exclude, id) {
			continue
		}
		counts[id] = 0
	}
	existing, err := r.GetAll(p.ctx, p.t, f)
	if err != nil {
		return assignments, err
	}
	for _, cid := range existing {
		if _, ok := counts[cid]; ok {
			counts[cid]++
		}
	}
	leastLoaded := func() (uint32, bool) {
		var best uint32
		bestCount := -1
		for id, c := range counts {
			if bestCount == -1 || c < bestCount {
				best = id
				bestCount = c
			}
		}
		return best, bestCount != -1
	}
	for _, npcId := range npcObjectIds {
		if cur, ok := existing[npcId]; ok && !contains(live, cur) {
			if derr := r.Release(p.ctx, p.t, f, npcId); derr != nil {
				p.l.WithError(derr).Warnf("Unable to release stale controller entry for NPC [%d]; skipping.", npcId)
				continue
			}
		}
		var winner uint32
		found := false
		for {
			cand, ok := leastLoaded()
			if !ok {
				break
			}
			if p.isHidden(cand) {
				delete(counts, cand)
				continue
			}
			winner = cand
			found = true
			break
		}
		if !found {
			p.l.Debugf("No eligible NPC controller candidate in field [%s]; NPC [%d] left uncontrolled.", f.Id(), npcId)
			continue
		}
		won, cerr := r.Claim(p.ctx, p.t, f, npcId, winner)
		if cerr != nil {
			p.l.WithError(cerr).Warnf("Unable to claim NPC [%d] for [%d]; skipping.", npcId, winner)
			continue
		}
		if won {
			assignments[npcId] = winner
			counts[winner]++
		}
	}
	return assignments, nil
}

// UncontrolledIn filters npcObjectIds to those with no live controller —
// absent entry, or an entry whose controller is no longer in the field.
func (p *ProcessorImpl) UncontrolledIn(f field.Model, npcObjectIds []uint32) ([]uint32, error) {
	r := GetRegistry()
	if r == nil {
		return nil, nil
	}
	existing, err := r.GetAll(p.ctx, p.t, f)
	if err != nil {
		return nil, err
	}
	live, err := p.fieldIdsFn(f)
	if err != nil {
		return nil, err
	}
	var out []uint32
	for _, npcId := range npcObjectIds {
		cur, ok := existing[npcId]
		if !ok || !contains(live, cur) {
			out = append(out, npcId)
		}
	}
	return out, nil
}

// IsController is the movement/animation guard (task-176 companion change
// 1). Fail-open TRUE on infrastructure failure or pre-init so NPC motion
// never freezes on a Redis outage; false for uncontrolled NPCs and
// non-controllers (spoof/stale-client suppression).
func IsController(ctx context.Context, t tenant.Model, f field.Model, characterId uint32, npcObjectId uint32) bool {
	r := GetRegistry()
	if r == nil {
		return true
	}
	cur, ok, err := r.ControllerOf(ctx, t, f, npcObjectId)
	if err != nil {
		return true
	}
	return ok && cur == characterId
}
