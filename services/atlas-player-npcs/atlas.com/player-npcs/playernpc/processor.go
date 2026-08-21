package playernpc

import (
	"atlas-player-npcs/allocation"
	"atlas-player-npcs/character"
	"atlas-player-npcs/configuration"
	"atlas-player-npcs/eligibility"
	"atlas-player-npcs/inventory"
	"atlas-player-npcs/position"
	"atlas-player-npcs/ranking"
	"atlas-player-npcs/routing"
	"atlas-player-npcs/snapshot"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	mapdata "atlas-player-npcs/data/map"
	npcdata "atlas-player-npcs/data/npc"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Deploy runs the design §8.1 sequence -- pg_advisory_xact_lock, read
// in-use script ids and placed rectangles, allocate a script id, resolve a
// position, compute rank ordinals, insert -- inside one transaction, then
// emits DEPLOYED (and, when a reorganize (design §5.4) was needed to fit
// the new NPC, one REPOSITIONED first). Never inside the transaction: a
// client must never be told about an NPC that then rolls back (design
// §8.1). A commit that succeeds followed by a failed emit leaves a
// persisted NPC that surfaces on the next map enter, which is the
// accepted direction.
//
// Position is an explicit deploy position (PRD §5's optional `position:
// {x, y}` on the GM path). Deploy still resolves the foothold beneath it
// via the map's ground-snap endpoint, but otherwise uses it verbatim --
// "that position is used, not the positioner" -- so it never triggers a
// reorganize.
type Position struct {
	X int16
	Y int16
}

// EventType names the four design §7/§8.3 domain events a state-changing
// operation can emit.
type EventType string

const (
	EventTypeDeployed     EventType = "DEPLOYED"
	EventTypeUpdated      EventType = "UPDATED"
	EventTypeRemoved      EventType = "REMOVED"
	EventTypeRepositioned EventType = "REPOSITIONED"
)

// Event is what a state-changing operation emits, always after the
// transaction that produced it has committed (design §8.1). Repositioned
// carries every repositioned NPC in Models (design §5.4's single event
// carrying the full list); every other type carries exactly one.
type Event struct {
	Type    EventType
	WorldId byte
	MapId   uint32
	Models  []Model
}

// EventEmitter is invoked once per Event. Task 17 supplies the
// Kafka-backed implementation (kafka/message/playernpc + producer.go) and
// wires it into NewProcessor from main.go; this package stays free of a
// Kafka dependency in the meantime. A nil emitter passed to NewProcessor
// is replaced with a no-op.
type EventEmitter func(Event)

// Processor is the deploy transaction and the other Player NPC
// state-changing operations Task 16 (REST), Task 17 (Kafka commands) and
// Task 21 (GM commands) share.
type Processor interface {
	// Deploy creates a new Player NPC for characterId on (worldId, mapId).
	// enforceEligibility true runs the full eligibility.Evaluate check
	// (FR-1.1's automatic-deploy path, FR-6.3's conversation-triggered
	// path); false is the GM path (FR-8.1), which still enforces the
	// per-map duplicate rule and script-id availability but bypasses
	// level/GM checks. explicit, when non-nil, bypasses the positioner
	// entirely (PRD §5).
	Deploy(characterId uint32, worldId world.Id, mapId _map.Id, enforceEligibility bool, explicit *Position) (Model, error)
	// Redeploy refreshes an existing Player NPC's appearance and
	// current-standing ranks in place (design §6.2). Script id, object id
	// and position are unchanged.
	Redeploy(id uuid.UUID) (Model, error)
	// RemoveById deletes a single Player NPC by id.
	RemoveById(id uuid.UUID) (Model, error)
	// Remove deletes every Player NPC belonging to characterId, optionally
	// scoped to mapId (nil removes every map).
	Remove(characterId uint32, mapId *_map.Id) ([]Model, error)
	// GetById fetches a single Player NPC.
	GetById(id uuid.UUID) (Model, error)
	// GetByMap fetches every Player NPC deployed on (worldId, mapId) --
	// the map-enter read path (design §6).
	GetByMap(worldId world.Id, mapId _map.Id, page model.Page) ([]Model, error)
}

type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	db   *gorm.DB
	cp   character.Processor
	ip   inventory.Processor
	rp   ranking.Processor
	cfgp configuration.Processor
	np   npcdata.Processor
	mp   mapdata.Processor
	emit EventEmitter
}

// NewProcessor constructs a Processor. The read clients (cp/ip/rp/cfgp/np/
// mp) are injected rather than constructed internally -- the same
// dependency-injection shape snapshot.Capture already uses (Task 13/14) --
// so processor_test.go can substitute stubs instead of making real HTTP
// calls, and so a caller that already holds one of these processors (e.g.
// a Kafka consumer handling several commands in one batch) is not forced
// to build a second one. Task 16/17/21's wiring constructs the real
// HTTP-backed processors with NewProcessor(l, ctx) from each package and
// passes them here. emit is invoked after every state-changing operation's
// transaction commits; pass nil for a no-op.
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, cp character.Processor, ip inventory.Processor, rp ranking.Processor, cfgp configuration.Processor, np npcdata.Processor, mp mapdata.Processor, emit EventEmitter) Processor {
	if emit == nil {
		emit = func(Event) {}
	}
	return &ProcessorImpl{l: l, ctx: ctx, db: db, cp: cp, ip: ip, rp: rp, cfgp: cfgp, np: np, mp: mp, emit: emit}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return getPlayerNpcModel(p.db.WithContext(p.ctx), id)
}

func (p *ProcessorImpl) GetByMap(worldId world.Id, mapId _map.Id, page model.Page) ([]Model, error) {
	return playerNpcsByMap(p.db.WithContext(p.ctx), byte(worldId), uint32(mapId), page)
}

func (p *ProcessorImpl) Deploy(characterId uint32, worldId world.Id, mapId _map.Id, enforceEligibility bool, explicit *Position) (Model, error) {
	te := tenant.MustFromContext(p.ctx)
	tenantId := te.Id()

	c, err := p.cp.GetById(characterId)
	if err != nil {
		return Model{}, err
	}

	snap, err := snapshot.Capture(characterId, worldId, p.cp, p.ip, p.rp)
	if err != nil {
		return Model{}, err
	}

	cfg := p.cfgp.GetByTenantId(tenantId)

	usable, err := allocation.UsablePoolFor(tenantId, func(id uint32) (bool, bool, error) {
		nm, err := p.np.GetById(id)
		if err != nil {
			if errors.Is(err, requests.ErrNotFound) {
				return false, false, nil
			}
			return false, false, err
		}
		return true, nm.Imitate(), nil
	})
	if err != nil {
		return Model{}, err
	}

	mapModel, err := p.mp.GetById(mapId)
	if err != nil {
		return Model{}, err
	}
	var bounds position.Rect
	if area := mapModel.MapArea(); area != nil {
		bounds = position.Rect{X: area.X(), Y: area.Y(), W: area.Width(), H: area.Height()}
	}
	snap2 := groundSnapFunc(p.mp, mapId)

	var created Model
	var repositioned []uuid.UUID

	err = database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		if err := advisoryLock(tx, tenantId, byte(worldId), uint32(mapId)); err != nil {
			return err
		}

		existingCount, err := countByName(tx, byte(worldId), uint32(mapId), c.Name())
		if err != nil {
			return err
		}
		if existingCount > 0 {
			return ErrDuplicate
		}
		if enforceEligibility {
			if ok, reason := eligibility.Evaluate(cfg, c, existingCount, false); !ok {
				if reason == eligibility.ReasonDuplicate {
					return ErrDuplicate
				}
				return ErrIneligible
			}
		}

		inUse, err := inUseScriptIds(tx, byte(worldId))
		if err != nil {
			return err
		}
		set := constants.For(te.Region(), te.MajorVersion(), te.MinorVersion())
		branch := allocation.BranchFor(set, c.JobId(), mapId)
		scriptId, allocErr := allocation.Allocate(usable, inUse, branch)
		if allocErr != nil {
			return ErrPoolExhausted
		}

		jobCategory := routing.JobCategory(c.JobId())
		worldJobRank, err := nextWorldJobRank(tx, byte(worldId), jobCategory)
		if err != nil {
			return err
		}
		overallJobRank, err := nextOverallJobRank(tx, jobCategory)
		if err != nil {
			return err
		}

		var point position.Point
		var step byte
		switch {
		case explicit != nil:
			results, err := p.mp.Ground(mapId, []mapdata.GroundPoint{mapdata.NewGroundPoint(explicit.X, explicit.Y)})
			if err != nil {
				return err
			}
			var fh uint32
			if len(results) > 0 {
				fh = results[0].Fh()
			}
			point = position.Point{X: explicit.X, CY: explicit.Y, Fh: fh}
			step, err = currentStepForMap(tx, byte(worldId), uint32(mapId))
			if err != nil {
				return err
			}
		case routing.IsPodiumMap(mapId):
			point, step, repositioned, err = p.resolvePodiumPosition(tx, byte(worldId), uint32(mapId), cfg, worldJobRank)
			if err != nil {
				return err
			}
		default:
			point, step, repositioned, err = p.resolveGridPosition(tx, byte(worldId), uint32(mapId), cfg, bounds, scriptId, snap2)
			if err != nil {
				return err
			}
		}

		objectId := objectid.PlayerNpcObjectIdFor(scriptId)

		equipment := make([]EquipmentModel, 0, len(snap.Equipment()))
		for _, row := range snap.Equipment() {
			em, err := NewEquipmentBuilder().SetSlot(row.Slot()).SetItemId(row.ItemId()).Build()
			if err != nil {
				return err
			}
			equipment = append(equipment, em)
		}

		m, err := NewBuilder().
			SetCharacterId(characterId).
			SetName(c.Name()).
			SetWorldId(byte(worldId)).
			SetMapId(uint32(mapId)).
			SetScriptId(scriptId).
			SetObjectId(objectId).
			SetGender(snap.Gender()).
			SetSkin(snap.SkinColor()).
			SetFace(snap.Face()).
			SetHair(snap.Hair()).
			SetJobId(snap.JobId()).
			SetX(point.X).
			SetCy(point.CY).
			SetFh(uint16(point.Fh)).
			SetStep(step).
			SetWorldRank(snap.WorldRank()).
			SetOverallRank(snap.OverallRank()).
			SetWorldJobRank(worldJobRank).
			SetOverallJobRank(overallJobRank).
			SetEquipment(equipment).
			Build()
		if err != nil {
			return err
		}

		entity := MakeEntity(tenantId, m)
		entity.Id = uuid.Nil
		if err := tx.Create(&entity).Error; err != nil {
			return err
		}
		eqEntities := MakeEquipmentEntities(tenantId, entity.Id, m)
		for i := range eqEntities {
			eqEntities[i].Id = uuid.Nil
			if err := tx.Create(&eqEntities[i]).Error; err != nil {
				return err
			}
		}

		hydrated, err := getPlayerNpcModel(tx, entity.Id)
		if err != nil {
			return err
		}
		created = hydrated
		return nil
	})
	if err != nil {
		return Model{}, err
	}

	if len(repositioned) > 0 {
		models := make([]Model, 0, len(repositioned))
		for _, id := range repositioned {
			m, hydrateErr := getPlayerNpcModel(p.db.WithContext(p.ctx), id)
			if hydrateErr != nil {
				p.l.WithError(hydrateErr).Warnf("Unable to hydrate reorganized Player NPC [%s] for the REPOSITIONED event.", id)
				continue
			}
			models = append(models, m)
		}
		p.emit(Event{Type: EventTypeRepositioned, WorldId: byte(worldId), MapId: uint32(mapId), Models: models})
	}
	p.emit(Event{Type: EventTypeDeployed, WorldId: byte(worldId), MapId: uint32(mapId), Models: []Model{created}})
	return created, nil
}

// resolveGridPosition finds a place for scriptId on a grid map, escalating
// to a design §5.4 reorganize when the current step has no free slot. Any
// reorganized existing row is persisted inside tx (design §5.4 step 3);
// their ids are returned so the caller can hydrate the single
// REPOSITIONED event after commit.
func (p *ProcessorImpl) resolveGridPosition(tx *gorm.DB, worldId byte, mapId uint32, cfg configuration.Model, bounds position.Rect, scriptId uint32, snap position.SnapFunc) (position.Point, byte, []uuid.UUID, error) {
	t := tuningFor(cfg)

	existing, err := mapEntities(tx, worldId, mapId)
	if err != nil {
		return position.Point{}, 0, nil, err
	}
	step := byte(0)
	if len(existing) > 0 {
		step = existing[0].Step
	}

	dx, dy := position.GridPitch(t, step)
	placed := make([]position.Rect, 0, len(existing))
	for _, e := range existing {
		placed = append(placed, position.Rect{X: e.X, Y: e.Cy, W: dx, H: dy})
	}

	point, err := position.NextGridPosition(t, bounds, step, placed, snap)
	if err == nil {
		return point, step, nil, nil
	}
	if !errors.Is(err, position.ErrMapFull) {
		return position.Point{}, 0, nil, err
	}
	if !cfg.OrganizeArea() || step >= byte(cfg.AreaSteps()) {
		return position.Point{}, 0, nil, ErrMapFull
	}

	toPlace := make([]position.Placement, 0, len(existing)+1)
	for _, e := range existing {
		toPlace = append(toPlace, position.Placement{ScriptId: e.ScriptId})
	}
	toPlace = append(toPlace, position.Placement{ScriptId: scriptId})

	placements, err := position.Reorganize(t, bounds, step, toPlace, snap)
	if err != nil {
		if errors.Is(err, position.ErrMapFull) {
			return position.Point{}, 0, nil, ErrMapFull
		}
		return position.Point{}, 0, nil, err
	}

	byScript := make(map[uint32]Entity, len(existing))
	for _, e := range existing {
		byScript[e.ScriptId] = e
	}

	var newPoint position.Point
	var newStep byte
	reorganized := make([]uuid.UUID, 0, len(existing))
	for _, pl := range placements {
		if pl.ScriptId == scriptId {
			newPoint, newStep = pl.Point, pl.Step
			continue
		}
		e, ok := byScript[pl.ScriptId]
		if !ok {
			continue
		}
		if err := updatePosition(tx, e.Id, pl.Point.X, pl.Point.CY, uint16(pl.Point.Fh), pl.Point.X+50, pl.Point.X-50, pl.Step); err != nil {
			return position.Point{}, 0, nil, err
		}
		reorganized = append(reorganized, e.Id)
	}
	return newPoint, newStep, reorganized, nil
}

// resolvePodiumPosition finds a podium slot for the new NPC by rank
// (worldJobRank-1, design §5.2's "rank" being the deployment ordinal on
// the podium's single job branch). This reading is verified, not assumed:
// every routing.podiumMaps entry maps 1:1 to exactly one JobCategory
// (routing.HallOfFameMapFor's switch), so every occupant of a given podium
// map shares one job category, and worldJobRank -- nextWorldJobRank's
// MAX(world_job_rank)+1 over (world, job category), starting at 1 -- runs
// gapless from 1 across that map's occupants. worldJobRank-1 therefore
// lands exactly on PodiumPosition's 0-based rank with no gaps or
// collisions; processor_test.go's "podium deploy" subtest pins the
// resulting slot for three sequential ranks, including the step raise.
// A raised step (design §5.2: occupancy reaches 3*step) repositions every
// existing occupant at the new step, persisted inside tx, and their ids
// are returned for the REPOSITIONED event. A podium map with no rows yet
// starts at step 1, not 0 (design §5.2: PodiumPosition divides by step and
// treats the encoded pair as starting at 1 for the podium path).
func (p *ProcessorImpl) resolvePodiumPosition(tx *gorm.DB, worldId byte, mapId uint32, cfg configuration.Model, worldJobRank uint32) (position.Point, byte, []uuid.UUID, error) {
	t := tuningFor(cfg)

	existing, err := mapEntities(tx, worldId, mapId)
	if err != nil {
		return position.Point{}, 0, nil, err
	}
	step := byte(1)
	if len(existing) > 0 && existing[0].Step > 0 {
		step = existing[0].Step
	}

	count := uint32(len(existing) + 1)
	newStep, raised, err := position.RaisePodiumStep(t, step, count)
	if err != nil {
		return position.Point{}, 0, nil, ErrMapFull
	}

	var reorganized []uuid.UUID
	if raised {
		reorganized = make([]uuid.UUID, 0, len(existing))
		for _, e := range existing {
			pt, err := position.PodiumPosition(podiumRank(e.WorldJobRank), newStep)
			if err != nil {
				return position.Point{}, 0, nil, err
			}
			if err := updatePosition(tx, e.Id, pt.X, pt.CY, uint16(pt.Fh), pt.X+50, pt.X-50, newStep); err != nil {
				return position.Point{}, 0, nil, err
			}
			reorganized = append(reorganized, e.Id)
		}
	}

	point, err := position.PodiumPosition(podiumRank(worldJobRank), newStep)
	if err != nil {
		return position.Point{}, 0, nil, err
	}
	return point, newStep, reorganized, nil
}

// podiumRank converts a 1-based world_job_rank deployment ordinal into
// PodiumPosition's 0-based rank.
func podiumRank(worldJobRank uint32) uint32 {
	if worldJobRank == 0 {
		return 0
	}
	return worldJobRank - 1
}

func tuningFor(cfg configuration.Model) position.Tuning {
	return position.Tuning{
		InitialX:     cfg.InitialX(),
		InitialY:     cfg.InitialY(),
		AreaX:        cfg.AreaX(),
		AreaY:        cfg.AreaY(),
		AreaSteps:    byte(cfg.AreaSteps()),
		OrganizeArea: cfg.OrganizeArea(),
	}
}

// groundSnapFunc adapts data/map's Ground request into a position.SnapFunc.
func groundSnapFunc(mp mapdata.Processor, mapId _map.Id) position.SnapFunc {
	return func(points []position.Point) ([]position.Point, error) {
		gp := make([]mapdata.GroundPoint, 0, len(points))
		for _, pt := range points {
			gp = append(gp, mapdata.NewGroundPoint(pt.X, pt.CY))
		}
		results, err := mp.Ground(mapId, gp)
		if err != nil {
			return nil, err
		}
		out := make([]position.Point, 0, len(results))
		for _, r := range results {
			out = append(out, position.Point{X: r.X(), CY: r.Y(), Fh: r.Fh()})
		}
		return out, nil
	}
}

// Redeploy refreshes appearance and current-standing ranks in place
// (design §6.2). Script id, object id and position are immutable through
// this path; the deployment-ordinal ranks (world_job_rank,
// overall_job_rank) are frozen at deploy time and untouched here (design
// §6.3).
func (p *ProcessorImpl) Redeploy(id uuid.UUID) (Model, error) {
	te := tenant.MustFromContext(p.ctx)

	existing, err := getPlayerNpcModel(p.db.WithContext(p.ctx), id)
	if err != nil {
		return Model{}, err
	}

	snap, err := snapshot.Capture(existing.CharacterId(), world.Id(existing.WorldId()), p.cp, p.ip, p.rp)
	if err != nil {
		return Model{}, err
	}

	var updated Model
	err = database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		equipment := make([]EquipmentModel, 0, len(snap.Equipment()))
		for _, row := range snap.Equipment() {
			em, err := NewEquipmentBuilder().SetSlot(row.Slot()).SetItemId(row.ItemId()).Build()
			if err != nil {
				return err
			}
			equipment = append(equipment, em)
		}
		if err := replaceEquipment(tx, te.Id(), id, equipment); err != nil {
			return err
		}
		if err := updateAppearanceAndRank(tx, id, snap.Gender(), snap.SkinColor(), snap.Face(), snap.Hair(), uint16(snap.JobId()), snap.WorldRank(), snap.OverallRank()); err != nil {
			return err
		}
		m, err := getPlayerNpcModel(tx, id)
		if err != nil {
			return err
		}
		updated = m
		return nil
	})
	if err != nil {
		return Model{}, err
	}

	p.emit(Event{Type: EventTypeUpdated, WorldId: updated.WorldId(), MapId: updated.MapId(), Models: []Model{updated}})
	return updated, nil
}

// RemoveById deletes a single Player NPC (and its equipment rows,
// cascade) and emits REMOVED.
func (p *ProcessorImpl) RemoveById(id uuid.UUID) (Model, error) {
	m, err := getPlayerNpcModel(p.db.WithContext(p.ctx), id)
	if err != nil {
		return Model{}, err
	}
	if err := deletePlayerNpc(p.db.WithContext(p.ctx), id); err != nil {
		return Model{}, err
	}
	p.emit(Event{Type: EventTypeRemoved, WorldId: m.WorldId(), MapId: m.MapId(), Models: []Model{m}})
	return m, nil
}

// Remove deletes every Player NPC belonging to characterId, optionally
// scoped to mapId, emitting one REMOVED per row removed (design's REMOVED
// shape is per-NPC, unlike REPOSITIONED's single list-carrying event).
func (p *ProcessorImpl) Remove(characterId uint32, mapId *_map.Id) ([]Model, error) {
	entities, err := entitiesByCharacter(p.db.WithContext(p.ctx), characterId, mapId)
	if err != nil {
		return nil, err
	}
	removed := make([]Model, 0, len(entities))
	for _, e := range entities {
		m, err := p.RemoveById(e.Id)
		if err != nil {
			return removed, err
		}
		removed = append(removed, m)
	}
	return removed, nil
}
