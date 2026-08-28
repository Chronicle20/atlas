// Package snapshot holds the session-scoped character snapshot (task-122,
// PS-1): a per-pod, tenant-scoped projection of character core / skills /
// inventory / buffs plus a locally-fed position overlay, maintained from
// Kafka events and REST miss-fallbacks. Entries are created ONLY by the
// lazy read path (View) and evicted with the session; event mutators are
// update-only. Every mutation bumps the touched component's generation so
// an in-flight REST backfill recorded against an older generation is
// discarded instead of clobbering newer event-driven state (design §4.4).
package snapshot

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/compartment"
	"atlas-channel/inventory"
	"sync"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type entry struct {
	core      character.Model
	coreGen   uint64
	coreValid bool

	skills      []skill.Model
	skillsGen   uint64
	skillsValid bool

	inv      inventory.Model
	invGen   uint64
	invValid bool

	buffs      []buff.Model
	buffsGen   uint64
	buffsValid bool

	posX, posY int16
	posValid   bool

	composed      character.Model
	composedValid bool
}

type Registry struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]*entry
}

var (
	registryOnce sync.Once
	registry     *Registry
)

func GetRegistry() *Registry {
	registryOnce.Do(func() {
		registry = &Registry{perTenant: map[uuid.UUID]map[uint32]*entry{}}
	})
	return registry
}

// ComponentView is a value-copy view of one entry's components, validity,
// and generations, used by the read path to fetch and backfill outside the
// lock.
type ComponentView struct {
	Core      character.Model
	CoreValid bool
	CoreGen   uint64

	Skills      []skill.Model
	SkillsValid bool
	SkillsGen   uint64

	Inv      inventory.Model
	InvValid bool
	InvGen   uint64

	Buffs      []buff.Model
	BuffsValid bool
	BuffsGen   uint64

	PosX, PosY int16
	PosValid   bool
}

// View returns the current component view for characterId, creating an
// all-invalid entry when absent. This is the ONLY call that creates
// entries: events must never create them (consumers start at LastOffset,
// so an event-created entry would be a partial hallucination).
func (r *Registry) View(t tenant.Model, characterId uint32) ComponentView {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.entryLocked(t, characterId, true)
	return ComponentView{
		Core: e.core, CoreValid: e.coreValid, CoreGen: e.coreGen,
		Skills: e.skills, SkillsValid: e.skillsValid, SkillsGen: e.skillsGen,
		Inv: e.inv, InvValid: e.invValid, InvGen: e.invGen,
		Buffs: e.buffs, BuffsValid: e.buffsValid, BuffsGen: e.buffsGen,
		PosX: e.posX, PosY: e.posY, PosValid: e.posValid,
	}
}

// entryLocked fetches (optionally creating) the entry. Caller holds mu.
func (r *Registry) entryLocked(t tenant.Model, characterId uint32, create bool) *entry {
	tm, ok := r.perTenant[t.Id()]
	if !ok {
		if !create {
			return nil
		}
		tm = map[uint32]*entry{}
		r.perTenant[t.Id()] = tm
	}
	e, ok := tm[characterId]
	if !ok {
		if !create {
			return nil
		}
		e = &entry{}
		tm[characterId] = e
	}
	return e
}

// ComposedIfValid returns the composed decorated model when every
// component is valid, rebuilding the cached composition if stale. Position
// is an overlay: when invalid, the core model's REST-sourced X/Y are used
// (exactly today's source).
func (r *Registry) ComposedIfValid(t tenant.Model, characterId uint32) (character.Model, bool) {
	r.mu.RLock()
	e := r.entryLocked(t, characterId, false)
	if e == nil || !e.coreValid || !e.skillsValid || !e.invValid || !e.buffsValid {
		r.mu.RUnlock()
		return character.Model{}, false
	}
	if e.composedValid {
		m := e.composed
		r.mu.RUnlock()
		return m, true
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	e = r.entryLocked(t, characterId, false)
	if e == nil || !e.coreValid || !e.skillsValid || !e.invValid || !e.buffsValid {
		return character.Model{}, false
	}
	if !e.composedValid {
		e.composed = composeLocked(e)
		e.composedValid = true
	}
	return e.composed, true
}

// composeLocked builds the same decorated model
// GetById(InventoryDecorator, SkillModelDecorator) returns today:
// base core -> position overlay -> SetInventory (rebuilds the equipment
// map from negative slots, character/model.go SetInventory) -> SetSkills.
func composeLocked(e *entry) character.Model {
	m := e.core
	if e.posValid {
		m = character.CloneModel(m).SetX(e.posX).SetY(e.posY).MustBuild()
	}
	m = m.SetInventory(e.inv)
	m = m.SetSkills(e.skills)
	return m
}

// --- Backfills (generation-checked) -----------------------------------------

func (r *Registry) BackfillCore(t tenant.Model, characterId uint32, m character.Model, gen uint64) bool {
	return r.backfill(t, characterId, componentCore,
		func(e *entry) bool { return e.coreGen == gen },
		func(e *entry) { e.core = m; e.coreValid = true })
}

func (r *Registry) BackfillSkills(t tenant.Model, characterId uint32, ms []skill.Model, gen uint64) bool {
	cp := append([]skill.Model(nil), ms...)
	return r.backfill(t, characterId, componentSkills,
		func(e *entry) bool { return e.skillsGen == gen },
		func(e *entry) { e.skills = cp; e.skillsValid = true })
}

func (r *Registry) BackfillInventory(t tenant.Model, characterId uint32, inv inventory.Model, gen uint64) bool {
	return r.backfill(t, characterId, componentInventory,
		func(e *entry) bool { return e.invGen == gen },
		func(e *entry) { e.inv = inv; e.invValid = true })
}

func (r *Registry) BackfillBuffs(t tenant.Model, characterId uint32, bs []buff.Model, gen uint64) bool {
	cp := append([]buff.Model(nil), bs...)
	return r.backfill(t, characterId, componentBuffs,
		func(e *entry) bool { return e.buffsGen == gen },
		func(e *entry) { e.buffs = cp; e.buffsValid = true })
}

func (r *Registry) backfill(t tenant.Model, characterId uint32, component string, genOK func(*entry) bool, apply func(*entry)) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.entryLocked(t, characterId, false)
	if e == nil {
		// Entry evicted while the fetch was in flight (session destroyed):
		// do not resurrect it.
		recordUpdate(t, component, kindBackfillDiscarded)
		return false
	}
	if !genOK(e) {
		recordUpdate(t, component, kindBackfillDiscarded)
		return false
	}
	apply(e)
	e.composedValid = false
	recordUpdate(t, component, kindBackfill)
	return true
}

// --- Update-only event mutators ----------------------------------------------
//
// mutate runs f against an existing entry (no-op when absent), and clears
// the composed cache after every mutation. In-place updates must apply only
// when the component is valid; when it is invalid the gen bump alone
// matters (it kills any in-flight backfill so the next read refetches
// state that includes this event's effect).

func (r *Registry) mutate(t tenant.Model, characterId uint32, f func(e *entry)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.entryLocked(t, characterId, false)
	if e == nil {
		return
	}
	f(e)
	e.composedValid = false
}

// statValueKeys maps stat update types to the snake_case Values keys the
// atlas-character producer uses (convention pinned by the existing
// populated sites; task-122 event-coverage.md §1).
var statValueKeys = map[stat.Type]string{
	stat.TypeSkin:               "skin",
	stat.TypeFace:               "face",
	stat.TypeHair:               "hair",
	stat.TypeLevel:              "level",
	stat.TypeJob:                "job",
	stat.TypeStrength:           "strength",
	stat.TypeDexterity:          "dexterity",
	stat.TypeIntelligence:       "intelligence",
	stat.TypeLuck:               "luck",
	stat.TypeHp:                 "hp",
	stat.TypeMaxHp:              "max_hp",
	stat.TypeMp:                 "mp",
	stat.TypeMaxMp:              "max_mp",
	stat.TypeAvailableAP:        "available_ap",
	stat.TypeExperience:         "experience",
	stat.TypeFame:               "fame",
	stat.TypeMeso:               "meso",
	stat.TypeGachaponExperience: "gachapon_experience",
}

// applyStat returns the model with one absolute stat value applied. The
// second return is false for types the snapshot cannot apply in place
// (AVAILABLE_SP is a per-book string table on the model; unknown types are
// fail-safe).
func applyStat(m character.Model, u stat.Type, v float64) (character.Model, bool) {
	switch u {
	case stat.TypeSkin:
		return character.CloneModel(m).SetSkinColor(byte(v)).MustBuild(), true
	case stat.TypeFace:
		return character.CloneModel(m).SetFace(uint32(v)).MustBuild(), true
	case stat.TypeHair:
		return character.CloneModel(m).SetHair(uint32(v)).MustBuild(), true
	case stat.TypeLevel:
		return character.CloneModel(m).SetLevel(byte(v)).MustBuild(), true
	case stat.TypeJob:
		return character.CloneModel(m).SetJobId(job.Id(uint16(v))).MustBuild(), true
	case stat.TypeStrength:
		return character.CloneModel(m).SetStrength(uint16(v)).MustBuild(), true
	case stat.TypeDexterity:
		return character.CloneModel(m).SetDexterity(uint16(v)).MustBuild(), true
	case stat.TypeIntelligence:
		return character.CloneModel(m).SetIntelligence(uint16(v)).MustBuild(), true
	case stat.TypeLuck:
		return character.CloneModel(m).SetLuck(uint16(v)).MustBuild(), true
	case stat.TypeHp:
		return character.CloneModel(m).SetHp(uint16(v)).MustBuild(), true
	case stat.TypeMaxHp:
		return character.CloneModel(m).SetMaxHp(uint16(v)).MustBuild(), true
	case stat.TypeMp:
		return character.CloneModel(m).SetMp(uint16(v)).MustBuild(), true
	case stat.TypeMaxMp:
		return character.CloneModel(m).SetMaxMp(uint16(v)).MustBuild(), true
	case stat.TypeAvailableAP:
		return character.CloneModel(m).SetAp(uint16(v)).MustBuild(), true
	case stat.TypeExperience:
		return character.CloneModel(m).SetExperience(uint32(v)).MustBuild(), true
	case stat.TypeFame:
		return character.CloneModel(m).SetFame(int16(v)).MustBuild(), true
	case stat.TypeMeso:
		return character.CloneModel(m).SetMeso(uint32(v)).MustBuild(), true
	case stat.TypeGachaponExperience:
		return character.CloneModel(m).SetGachaponExperience(uint32(v)).MustBuild(), true
	}
	return m, false
}

// isPetSnStat reports stat types that are not fields of the base character
// model (pets arrive only via decorators the attack path never applies) —
// skipped rather than invalidating.
func isPetSnStat(u stat.Type) bool {
	return u == stat.TypePetSn1 || u == stat.TypePetSn2 || u == stat.TypePetSn3
}

// ApplyStatChanged applies a STAT_CHANGED event: complete absolute Values
// update the core in place; anything less invalidates the core component
// (invalidate-and-refetch — never guess). Empty Updates (error-path
// emissions) are a no-op.
func (r *Registry) ApplyStatChanged(t tenant.Model, characterId uint32, updates []stat.Type, values map[string]interface{}) {
	if len(updates) == 0 {
		return
	}
	r.mutate(t, characterId, func(e *entry) {
		e.coreGen++
		if !e.coreValid {
			return
		}
		m := e.core
		for _, u := range updates {
			if isPetSnStat(u) {
				continue
			}
			key, known := statValueKeys[u]
			if !known {
				e.coreValid = false
				recordUpdate(t, componentCore, kindInvalidation)
				return
			}
			raw, present := values[key]
			if !present {
				e.coreValid = false
				recordUpdate(t, componentCore, kindInvalidation)
				return
			}
			f, isNum := raw.(float64)
			if !isNum {
				e.coreValid = false
				recordUpdate(t, componentCore, kindInvalidation)
				return
			}
			next, applied := applyStat(m, u, f)
			if !applied {
				e.coreValid = false
				recordUpdate(t, componentCore, kindInvalidation)
				return
			}
			m = next
		}
		e.core = m
		recordUpdate(t, componentCore, kindEventUpdate)
	})
}

func (r *Registry) SetLevel(t tenant.Model, characterId uint32, level byte) {
	r.mutate(t, characterId, func(e *entry) {
		e.coreGen++
		if !e.coreValid {
			return
		}
		e.core = character.CloneModel(e.core).SetLevel(level).MustBuild()
		recordUpdate(t, componentCore, kindEventUpdate)
	})
}

func (r *Registry) SetExperience(t tenant.Model, characterId uint32, exp uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.coreGen++
		if !e.coreValid {
			return
		}
		e.core = character.CloneModel(e.core).SetExperience(exp).MustBuild()
		recordUpdate(t, componentCore, kindEventUpdate)
	})
}

func (r *Registry) InvalidateCore(t tenant.Model, characterId uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.coreGen++
		e.coreValid = false
		recordUpdate(t, componentCore, kindInvalidation)
	})
}

func (r *Registry) UpsertSkill(t tenant.Model, characterId uint32, sm skill.Model) {
	r.mutate(t, characterId, func(e *entry) {
		e.skillsGen++
		if !e.skillsValid {
			return
		}
		out := make([]skill.Model, 0, len(e.skills)+1)
		replaced := false
		for _, s := range e.skills {
			if s.Id() == sm.Id() {
				// Preserve the cooldown the REST populate carried; skill
				// CREATED/UPDATED events do not include it (v1: cooldown
				// events are ignored, not in the attack read-set).
				out = append(out, skill.Clone(sm).SetCooldownExpiresAt(s.CooldownExpiresAt()).MustBuild())
				replaced = true
			} else {
				out = append(out, s)
			}
		}
		if !replaced {
			out = append(out, sm)
		}
		e.skills = out
		recordUpdate(t, componentSkills, kindEventUpdate)
	})
}

func (r *Registry) RemoveSkill(t tenant.Model, characterId uint32, skillId skillconst.Id) {
	r.mutate(t, characterId, func(e *entry) {
		e.skillsGen++
		if !e.skillsValid {
			return
		}
		out := make([]skill.Model, 0, len(e.skills))
		for _, s := range e.skills {
			if s.Id() != skillId {
				out = append(out, s)
			}
		}
		e.skills = out
		recordUpdate(t, componentSkills, kindEventUpdate)
	})
}

func (r *Registry) InvalidateSkills(t tenant.Model, characterId uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.skillsGen++
		e.skillsValid = false
		recordUpdate(t, componentSkills, kindInvalidation)
	})
}

// replaceCompartment rebuilds the inventory with comp swapped in. A fresh
// builder is mandatory: inventory.CloneModel shares the compartments map
// with the source model, so building through it would mutate models
// already handed to readers (see TestRegistry_AssetMutationDoesNotAliasPriorReads).
func replaceCompartment(inv inventory.Model, comp compartment.Model) inventory.Model {
	b := inventory.NewBuilder(inv.CharacterId())
	for _, c := range inv.Compartments() {
		if c.Id() == comp.Id() {
			b.SetCompartment(comp)
		} else {
			b.SetCompartment(c)
		}
	}
	return b.MustBuild()
}

// mutateAssetInInventory finds the compartment holding assetId, applies
// transform to a fresh copy of its asset slice, and swaps the rebuilt
// compartment in. Returns false when no compartment holds the asset.
func mutateAssetInInventory(inv inventory.Model, assetId uint32, transform func(a asset.Model) asset.Model) (inventory.Model, bool) {
	for _, c := range inv.Compartments() {
		if _, ok := c.FindById(assetId); !ok {
			continue
		}
		out := make([]asset.Model, 0, len(c.Assets()))
		for _, a := range c.Assets() {
			if a.Id() == assetId {
				out = append(out, transform(a))
			} else {
				out = append(out, a)
			}
		}
		return replaceCompartment(inv, compartment.CloneModel(c).SetAssets(out).MustBuild()), true
	}
	return inv, false
}

// UpsertAsset replaces (by AssetId) or inserts the full asset into the
// compartment identified by compartmentId. An unknown compartment
// invalidates the inventory component instead of guessing.
func (r *Registry) UpsertAsset(t tenant.Model, characterId uint32, compartmentId uuid.UUID, a asset.Model) {
	r.mutate(t, characterId, func(e *entry) {
		e.invGen++
		if !e.invValid {
			return
		}
		comp, ok := e.inv.CompartmentById(compartmentId)
		if !ok {
			e.invValid = false
			recordUpdate(t, componentInventory, kindInvalidation)
			return
		}
		out := make([]asset.Model, 0, len(comp.Assets())+1)
		replaced := false
		for _, existing := range comp.Assets() {
			if existing.Id() == a.Id() {
				out = append(out, a)
				replaced = true
			} else {
				out = append(out, existing)
			}
		}
		if !replaced {
			out = append(out, a)
		}
		e.inv = replaceCompartment(e.inv, compartment.CloneModel(comp).SetAssets(out).MustBuild())
		recordUpdate(t, componentInventory, kindEventUpdate)
	})
}

func (r *Registry) SetAssetQuantity(t tenant.Model, characterId uint32, assetId uint32, quantity uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.invGen++
		if !e.invValid {
			return
		}
		next, ok := mutateAssetInInventory(e.inv, assetId, func(a asset.Model) asset.Model {
			return asset.Clone(a).SetQuantity(quantity).MustBuild()
		})
		if !ok {
			e.invValid = false
			recordUpdate(t, componentInventory, kindInvalidation)
			return
		}
		e.inv = next
		recordUpdate(t, componentInventory, kindEventUpdate)
	})
}

func (r *Registry) SetAssetSlot(t tenant.Model, characterId uint32, assetId uint32, slot int16) {
	r.mutate(t, characterId, func(e *entry) {
		e.invGen++
		if !e.invValid {
			return
		}
		next, ok := mutateAssetInInventory(e.inv, assetId, func(a asset.Model) asset.Model {
			return asset.Clone(a).SetSlot(slot).MustBuild()
		})
		if !ok {
			e.invValid = false
			recordUpdate(t, componentInventory, kindInvalidation)
			return
		}
		e.inv = next
		recordUpdate(t, componentInventory, kindEventUpdate)
	})
}

func (r *Registry) RemoveAsset(t tenant.Model, characterId uint32, assetId uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.invGen++
		if !e.invValid {
			return
		}
		for _, c := range e.inv.Compartments() {
			if _, ok := c.FindById(assetId); !ok {
				continue
			}
			out := make([]asset.Model, 0, len(c.Assets()))
			for _, a := range c.Assets() {
				if a.Id() != assetId {
					out = append(out, a)
				}
			}
			e.inv = replaceCompartment(e.inv, compartment.CloneModel(c).SetAssets(out).MustBuild())
			recordUpdate(t, componentInventory, kindEventUpdate)
			return
		}
		// Deleting an asset we never held: harmless for correctness of the
		// planner (it can only over-refetch), but the state is suspect —
		// invalidate.
		e.invValid = false
		recordUpdate(t, componentInventory, kindInvalidation)
	})
}

func (r *Registry) InvalidateInventory(t tenant.Model, characterId uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.invGen++
		e.invValid = false
		recordUpdate(t, componentInventory, kindInvalidation)
	})
}

func (r *Registry) UpsertBuff(t tenant.Model, characterId uint32, b buff.Model) {
	r.mutate(t, characterId, func(e *entry) {
		e.buffsGen++
		if !e.buffsValid {
			return
		}
		out := make([]buff.Model, 0, len(e.buffs)+1)
		replaced := false
		for _, existing := range e.buffs {
			if existing.SourceId() == b.SourceId() {
				out = append(out, b)
				replaced = true
			} else {
				out = append(out, existing)
			}
		}
		if !replaced {
			out = append(out, b)
		}
		e.buffs = out
		recordUpdate(t, componentBuffs, kindEventUpdate)
	})
}

func (r *Registry) RemoveBuff(t tenant.Model, characterId uint32, sourceId int32) {
	r.mutate(t, characterId, func(e *entry) {
		e.buffsGen++
		if !e.buffsValid {
			return
		}
		out := make([]buff.Model, 0, len(e.buffs))
		for _, existing := range e.buffs {
			if existing.SourceId() != sourceId {
				out = append(out, existing)
			}
		}
		e.buffs = out
		recordUpdate(t, componentBuffs, kindEventUpdate)
	})
}

func (r *Registry) InvalidateBuffs(t tenant.Model, characterId uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.buffsGen++
		e.buffsValid = false
		recordUpdate(t, componentBuffs, kindInvalidation)
	})
}

// SetPosition feeds the locally-observed movement fold (zero hops; strictly
// fresher than the REST projection of the same packets — FR-2.5).
func (r *Registry) SetPosition(t tenant.Model, characterId uint32, x, y int16) {
	r.mutate(t, characterId, func(e *entry) {
		e.posX, e.posY = x, y
		e.posValid = true
		recordUpdate(t, componentPosition, kindEventUpdate)
	})
}

func (r *Registry) InvalidatePosition(t tenant.Model, characterId uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.posValid = false
		recordUpdate(t, componentPosition, kindInvalidation)
	})
}

// Evict removes one character's snapshot (session destroy: logout,
// disconnect, channel change all funnel through session.Processor.Destroy).
func (r *Registry) Evict(t tenant.Model, characterId uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tm, ok := r.perTenant[t.Id()]
	if !ok {
		return
	}
	delete(tm, characterId)
	if len(tm) == 0 {
		delete(r.perTenant, t.Id())
	}
}

// EvictTenant drops every entry for the tenant (listener drain).
func (r *Registry) EvictTenant(tid uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.perTenant, tid)
}
