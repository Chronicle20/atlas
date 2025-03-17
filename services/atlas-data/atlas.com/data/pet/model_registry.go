package pet

import (
	"atlas-data/registry"
	"sync"
)

var mmReg *registry.Registry[uint32, Model]
var mmOnce sync.Once

func GetModelRegistry() *registry.Registry[uint32, Model] {
	mmOnce.Do(func() {
		mmReg = registry.NewRegistry[uint32, Model]()
	})
	return mmReg
}
