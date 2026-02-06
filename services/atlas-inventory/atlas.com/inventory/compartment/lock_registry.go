package compartment

import (
	"fmt"
	"sync"

	"github.com/Chronicle20/atlas-constants/inventory"
)

type lockRegistry struct {
	locks sync.Map
}

var lr *lockRegistry
var once sync.Once

func LockRegistry() *lockRegistry {
	if lr == nil {
		once.Do(func() {
			lr = &lockRegistry{}
		})
	}
	return lr
}

// lockKey is a helper function to generate a unique key for each inventory lock
func lockKey(characterID uint32, inventoryType inventory.Type) string {
	return fmt.Sprintf("%d:%d", characterID, inventoryType)
}

func (r *lockRegistry) Get(characterId uint32, inventoryType inventory.Type) *sync.RWMutex {
	key := lockKey(characterId, inventoryType)
	val, _ := r.locks.LoadOrStore(key, &sync.RWMutex{})
	if mtx, ok := val.(*sync.RWMutex); ok {
		return mtx
	}
	mtx := &sync.RWMutex{}
	r.locks.Store(key, mtx)
	return mtx
}

func (r *lockRegistry) Delete(characterId uint32, inventoryType inventory.Type) {
	r.locks.Delete(lockKey(characterId, inventoryType))
}

func (r *lockRegistry) DeleteForCharacter(characterId uint32) {
	for _, t := range inventory.Types {
		r.locks.Delete(lockKey(characterId, t))
	}
}
