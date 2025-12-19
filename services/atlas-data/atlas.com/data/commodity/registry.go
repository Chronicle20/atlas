package commodity

import (
	"atlas-data/document"
	"sync"
)

var mmReg *document.Registry[string, RestModel]
var mmOnce sync.Once

func GetModelRegistry() *document.Registry[string, RestModel] {
	mmOnce.Do(func() {
		mmReg = document.NewRegistry[string, RestModel]()
	})
	return mmReg
}
