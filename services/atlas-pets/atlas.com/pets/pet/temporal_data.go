package pet

import "sync"

type temporalData struct {
	x      int16
	y      int16
	stance byte
}

func (d *temporalData) UpdatePosition(x int16, y int16) *temporalData {
	return &temporalData{
		x:      x,
		y:      y,
		stance: d.stance,
	}
}

func (d *temporalData) Update(x int16, y int16, stance byte) *temporalData {
	return &temporalData{
		x:      x,
		y:      y,
		stance: stance,
	}
}

func (d *temporalData) UpdateStance(stance byte) *temporalData {
	return &temporalData{
		x:      d.x,
		y:      d.y,
		stance: stance,
	}
}

func (d *temporalData) X() int16 {
	return d.x
}

func (d *temporalData) Y() int16 {
	return d.y
}

func (d *temporalData) Stance() byte {
	return d.stance
}

type temporalRegistry struct {
	data     map[uint32]*temporalData
	mutex    *sync.RWMutex
	petLocks map[uint32]*sync.RWMutex
}

func (r *temporalRegistry) UpdatePosition(petId uint32, x int16, y int16) {
	r.lockPet(petId)
	if val, ok := r.data[petId]; ok {
		r.data[petId] = val.UpdatePosition(x, y)
	} else {
		r.data[petId] = &temporalData{
			x:      x,
			y:      y,
			stance: 0,
		}
	}
	r.unlockPet(petId)
}

func (r *temporalRegistry) lockPet(petId uint32) {
	r.mutex.Lock()
	lock, exists := r.petLocks[petId]
	if !exists {
		lock = &sync.RWMutex{}
		r.petLocks[petId] = lock
	}
	r.mutex.Unlock()
	lock.Lock()
}

func (r *temporalRegistry) readLockPet(petId uint32) {
	r.mutex.Lock()
	lock, exists := r.petLocks[petId]
	if !exists {
		lock = &sync.RWMutex{}
		r.petLocks[petId] = lock
	}
	r.mutex.Unlock()
	lock.RLock()
}

func (r *temporalRegistry) unlockPet(petId uint32) {
	if val, ok := r.petLocks[petId]; ok {
		val.Unlock()
	}
}

func (r *temporalRegistry) readUnlockPet(petId uint32) {
	if val, ok := r.petLocks[petId]; ok {
		val.RUnlock()
	}
}

func (r *temporalRegistry) Update(petId uint32, x int16, y int16, stance byte) {
	r.lockPet(petId)
	if val, ok := r.data[petId]; ok {
		r.data[petId] = val.Update(x, y, stance)
	} else {
		r.data[petId] = &temporalData{
			x:      x,
			y:      y,
			stance: stance,
		}
	}
	r.unlockPet(petId)
}

func (r *temporalRegistry) UpdateStance(petId uint32, stance byte) {
	r.lockPet(petId)
	if val, ok := r.data[petId]; ok {
		r.data[petId] = val.UpdateStance(stance)
	} else {
		r.data[petId] = &temporalData{
			x:      0,
			y:      0,
			stance: stance,
		}
	}
	r.unlockPet(petId)
}

func (r *temporalRegistry) GetById(petId uint32) *temporalData {
	r.readLockPet(petId)
	defer r.readUnlockPet(petId)

	result := r.data[petId]
	if result != nil {
		return result
	}
	return &temporalData{
		x:      0,
		y:      0,
		stance: 0,
	}
}

var once sync.Once
var instance *temporalRegistry

func GetTemporalRegistry() *temporalRegistry {
	once.Do(func() {
		instance = &temporalRegistry{
			data:     make(map[uint32]*temporalData),
			mutex:    &sync.RWMutex{},
			petLocks: make(map[uint32]*sync.RWMutex),
		}
	})
	return instance
}
