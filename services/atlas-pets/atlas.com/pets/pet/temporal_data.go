package pet

import "sync"

type TemporalData struct {
	x      int16
	y      int16
	stance byte
	fh     int16
}

func (d *TemporalData) UpdatePosition(x int16, y int16, fh int16) *TemporalData {
	return &TemporalData{
		x:      x,
		y:      y,
		stance: d.stance,
		fh:     fh,
	}
}

func (d *TemporalData) Update(x int16, y int16, stance byte, fh int16) *TemporalData {
	return &TemporalData{
		x:      x,
		y:      y,
		stance: stance,
		fh:     fh,
	}
}

func (d *TemporalData) UpdateStance(stance byte) *TemporalData {
	return &TemporalData{
		x:      d.x,
		y:      d.y,
		stance: stance,
		fh:     d.fh,
	}
}

func (d *TemporalData) X() int16 {
	return d.x
}

func (d *TemporalData) Y() int16 {
	return d.y
}

func (d *TemporalData) Stance() byte {
	return d.stance
}

func (d *TemporalData) FH() int16 {
	return d.fh
}

func NewTemporalData() *TemporalData {
	return &TemporalData{fh: 1}
}

type TemporalRegistry interface {
	UpdatePosition(petId uint32, x int16, y int16, fh int16)
	Update(petId uint32, x int16, y int16, stance byte, fh int16)
	UpdateStance(petId uint32, stance byte)
	GetById(petId uint32) *TemporalData
	Remove(petId uint32)
}

type temporalRegistryImpl struct {
	data     map[uint32]*TemporalData
	mutex    *sync.RWMutex
	petLocks map[uint32]*sync.RWMutex
}

func (r *temporalRegistryImpl) UpdatePosition(petId uint32, x int16, y int16, fh int16) {
	r.lockPet(petId)
	if val, ok := r.data[petId]; ok {
		r.data[petId] = val.UpdatePosition(x, y, fh)
	} else {
		r.data[petId] = NewTemporalData()
	}
	r.unlockPet(petId)
}

func (r *temporalRegistryImpl) lockPet(petId uint32) {
	r.mutex.Lock()
	lock, exists := r.petLocks[petId]
	if !exists {
		lock = &sync.RWMutex{}
		r.petLocks[petId] = lock
	}
	r.mutex.Unlock()
	lock.Lock()
}

func (r *temporalRegistryImpl) readLockPet(petId uint32) {
	r.mutex.Lock()
	lock, exists := r.petLocks[petId]
	if !exists {
		lock = &sync.RWMutex{}
		r.petLocks[petId] = lock
	}
	r.mutex.Unlock()
	lock.RLock()
}

func (r *temporalRegistryImpl) unlockPet(petId uint32) {
	if val, ok := r.petLocks[petId]; ok {
		val.Unlock()
	}
}

func (r *temporalRegistryImpl) readUnlockPet(petId uint32) {
	if val, ok := r.petLocks[petId]; ok {
		val.RUnlock()
	}
}

func (r *temporalRegistryImpl) Update(petId uint32, x int16, y int16, stance byte, fh int16) {
	r.lockPet(petId)
	if _, ok := r.data[petId]; !ok {
		r.data[petId] = NewTemporalData()
	}
	r.data[petId] = r.data[petId].Update(x, y, stance, fh)
	r.unlockPet(petId)
}

func (r *temporalRegistryImpl) UpdateStance(petId uint32, stance byte) {
	r.lockPet(petId)
	if val, ok := r.data[petId]; ok {
		r.data[petId] = val.UpdateStance(stance)
	} else {
		r.data[petId] = NewTemporalData()
	}
	r.unlockPet(petId)
}

func (r *temporalRegistryImpl) GetById(petId uint32) *TemporalData {
	r.readLockPet(petId)
	defer r.readUnlockPet(petId)

	result := r.data[petId]
	if result != nil {
		return result
	}
	return NewTemporalData()
}

func (r *temporalRegistryImpl) Remove(petId uint32) {
	r.lockPet(petId)
	defer r.unlockPet(petId)
	delete(r.data, petId)
}

var once sync.Once
var instance *temporalRegistryImpl

func GetTemporalRegistry() TemporalRegistry {
	once.Do(func() {
		instance = &temporalRegistryImpl{
			data:     make(map[uint32]*TemporalData),
			mutex:    &sync.RWMutex{},
			petLocks: make(map[uint32]*sync.RWMutex),
		}
	})
	return instance
}
