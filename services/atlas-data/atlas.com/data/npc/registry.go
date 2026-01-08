package npc

import (
	"atlas-data/document"
	"sync"
)

var nmReg *document.Registry[string, RestModel]
var nmOnce sync.Once

func GetModelRegistry() *document.Registry[string, RestModel] {
	nmOnce.Do(func() {
		nmReg = document.NewRegistry[string, RestModel]()
	})
	return nmReg
}
