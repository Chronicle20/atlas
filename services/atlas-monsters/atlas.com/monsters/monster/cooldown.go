package monster

import (
	"sync"
	"time"
)

type cooldownEntry struct {
	expiresAt time.Time
}

type cooldownRegistry struct {
	mutex     sync.RWMutex
	cooldowns map[uint32]map[uint16]cooldownEntry // monsterId -> skillId -> entry
}

var cooldownReg *cooldownRegistry
var cooldownOnce sync.Once

func GetCooldownRegistry() *cooldownRegistry {
	cooldownOnce.Do(func() {
		cooldownReg = &cooldownRegistry{
			cooldowns: make(map[uint32]map[uint16]cooldownEntry),
		}
	})
	return cooldownReg
}

func (r *cooldownRegistry) IsOnCooldown(monsterId uint32, skillId uint16) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if skills, ok := r.cooldowns[monsterId]; ok {
		if cd, ok := skills[skillId]; ok {
			return time.Now().Before(cd.expiresAt)
		}
	}
	return false
}

func (r *cooldownRegistry) SetCooldown(monsterId uint32, skillId uint16, duration time.Duration) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.cooldowns[monsterId]; !ok {
		r.cooldowns[monsterId] = make(map[uint16]cooldownEntry)
	}
	r.cooldowns[monsterId][skillId] = cooldownEntry{
		expiresAt: time.Now().Add(duration),
	}
}

func (r *cooldownRegistry) ClearCooldowns(monsterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.cooldowns, monsterId)
}
