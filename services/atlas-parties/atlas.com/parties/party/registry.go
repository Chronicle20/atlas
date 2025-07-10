package party

import (
	"sync"

	"github.com/Chronicle20/atlas-tenant"
)

type Registry struct {
	lock          sync.Mutex
	tenantPartyId map[tenant.Model]uint32

	partyReg   map[tenant.Model]map[uint32]Model
	tenantLock map[tenant.Model]*sync.RWMutex

	// Efficient character-to-party mapping for fast lookups
	characterToParty map[tenant.Model]map[uint32]uint32
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.tenantPartyId = make(map[tenant.Model]uint32)
		registry.partyReg = make(map[tenant.Model]map[uint32]Model)
		registry.tenantLock = make(map[tenant.Model]*sync.RWMutex)
		registry.characterToParty = make(map[tenant.Model]map[uint32]uint32)
	})
	return registry
}

func (r *Registry) Create(t tenant.Model, leaderId uint32) Model {
	var partyId uint32
	var partyReg map[uint32]Model
	var tenantLock *sync.RWMutex
	var charToParty map[uint32]uint32
	var ok bool

	r.lock.Lock()
	if partyId, ok = r.tenantPartyId[t]; ok {
		partyId += 1
		partyReg = r.partyReg[t]
		tenantLock = r.tenantLock[t]
		charToParty = r.characterToParty[t]
	} else {
		partyId = StartPartyId
		partyReg = make(map[uint32]Model)
		tenantLock = &sync.RWMutex{}
		charToParty = make(map[uint32]uint32)
	}
	r.tenantPartyId[t] = partyId
	r.partyReg[t] = partyReg
	r.tenantLock[t] = tenantLock
	r.characterToParty[t] = charToParty
	r.lock.Unlock()

	m := Model{
		tenantId: t.Id(),
		id:       partyId,
		leaderId: leaderId,
		members:  make([]uint32, 0),
	}
	m.members = append(m.members, leaderId)

	tenantLock.Lock()
	partyReg[partyId] = m
	charToParty[leaderId] = partyId
	tenantLock.Unlock()
	return m
}

func (r *Registry) GetAll(t tenant.Model) []Model {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.partyReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = tl
		r.characterToParty[t] = make(map[uint32]uint32)
		r.lock.Unlock()
	}

	tl.RLock()
	defer tl.RUnlock()
	var res = make([]Model, 0)
	for _, v := range r.partyReg[t] {
		res = append(res, v)
	}
	return res
}

func (r *Registry) Get(t tenant.Model, partyId uint32) (Model, error) {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.partyReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = tl
		r.characterToParty[t] = make(map[uint32]uint32)
		r.lock.Unlock()
	}

	tl.RLock()
	defer tl.RUnlock()
	if m, ok := r.partyReg[t][partyId]; ok {
		return m, nil
	}
	return Model{}, ErrNotFound
}

func (r *Registry) Update(t tenant.Model, id uint32, updaters ...func(m Model) Model) (Model, error) {
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()

	oldModel := r.partyReg[t][id]
	newModel := oldModel
	for _, updater := range updaters {
		newModel = updater(newModel)
	}

	if len(newModel.members) > 6 {
		return Model{}, ErrAtCapacity
	}

	// Update character-to-party mapping for membership changes
	r.updateCharacterToPartyMapping(t, oldModel, newModel)

	r.partyReg[t][id] = newModel
	return newModel, nil
}

func (r *Registry) Remove(t tenant.Model, partyId uint32) {
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()

	// Remove character-to-party mappings for all members
	if party, ok := r.partyReg[t][partyId]; ok {
		for _, memberId := range party.members {
			delete(r.characterToParty[t], memberId)
		}
	}

	delete(r.partyReg[t], partyId)
}

// Helper method to update character-to-party mapping when membership changes
func (r *Registry) updateCharacterToPartyMapping(t tenant.Model, oldModel, newModel Model) {
	// Create sets of old and new members for efficient comparison
	oldMembers := make(map[uint32]bool)
	for _, memberId := range oldModel.members {
		oldMembers[memberId] = true
	}

	newMembers := make(map[uint32]bool)
	for _, memberId := range newModel.members {
		newMembers[memberId] = true
	}

	// Remove mapping for members who left
	for memberId := range oldMembers {
		if !newMembers[memberId] {
			delete(r.characterToParty[t], memberId)
		}
	}

	// Add mapping for new members
	for memberId := range newMembers {
		if !oldMembers[memberId] {
			r.characterToParty[t][memberId] = newModel.id
		}
	}
}

// Efficient character-to-party lookup using O(1) mapping with caching
func (r *Registry) GetPartyByCharacter(t tenant.Model, characterId uint32) (Model, error) {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.partyReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = tl
		r.characterToParty[t] = make(map[uint32]uint32)
		r.lock.Unlock()
	}

	tl.RLock()
	defer tl.RUnlock()
	if partyId, ok := r.characterToParty[t][characterId]; ok {
		if party, ok := r.partyReg[t][partyId]; ok {
			return party, nil
		}
	}

	return Model{}, ErrNotFound
}
