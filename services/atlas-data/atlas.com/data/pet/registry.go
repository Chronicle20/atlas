package pet

import (
	"atlas-data/document"
	"sync"
)

var mmReg *document.Registry[uint32, RestModel]
var mmOnce sync.Once

func GetModelRegistry() *document.Registry[uint32, RestModel] {
	mmOnce.Do(func() {
		mmReg = document.NewRegistry[uint32, RestModel]()
	})
	return mmReg
}
