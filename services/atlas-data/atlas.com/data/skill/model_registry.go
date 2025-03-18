package skill

import (
	"atlas-data/document"
	"sync"
)

var mmReg *document.Registry[uint32, Model]
var mmOnce sync.Once

func GetModelRegistry() *document.Registry[uint32, Model] {
	mmOnce.Do(func() {
		mmReg = document.NewRegistry[uint32, Model]()
	})
	return mmReg
}
