package inventory

import (
	"fmt"
	"sync"
)

type lockRegistry struct {
	locks sync.Map
}

var lr *lockRegistry
var once sync.Once

func GetLockRegistry() *lockRegistry {
	once.Do(func() {
		lr = &lockRegistry{}
	})
	return lr
}

// lockKey is a helper function to generate a unique key for each inventory lock
func lockKey(characterID uint32, inventoryType Type) string {
	return fmt.Sprintf("%d:%d", characterID, inventoryType)
}

func (r *lockRegistry) GetById(characterId uint32, inventoryType Type) *sync.RWMutex {
	key := lockKey(characterId, inventoryType)
	val, _ := r.locks.LoadOrStore(key, &sync.RWMutex{})
	return val.(*sync.RWMutex)
}

func (r *lockRegistry) DeleteForCharacter(characterId uint32) error {
	r.locks.Delete(lockKey(characterId, TypeValueEquip))
	r.locks.Delete(lockKey(characterId, TypeValueUse))
	r.locks.Delete(lockKey(characterId, TypeValueSetup))
	r.locks.Delete(lockKey(characterId, TypeValueETC))
	r.locks.Delete(lockKey(characterId, TypeValueCash))
	return nil
}
