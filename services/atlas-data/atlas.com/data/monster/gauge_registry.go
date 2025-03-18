package monster

import (
	"atlas-data/document"
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type Gauge struct {
	id     string
	exists bool
}

func (g Gauge) GetID() string {
	return g.id
}

func (g Gauge) Exists() bool {
	return g.exists
}

var mgReg *document.Registry[string, Gauge]
var mgOnce sync.Once

func GetMonsterGaugeRegistry() *document.Registry[string, Gauge] {
	mgOnce.Do(func() {
		mgReg = document.NewRegistry[string, Gauge]()
	})
	return mgReg
}

func InitGauge(t tenant.Model, path string) error {
	exml, err := xml.Read(path)
	if err != nil {
		return err
	}
	d, err := exml.ChildByName("MobGage/Mob")
	if err != nil {
		return err
	}

	for _, mxml := range d.CanvasNodes {
		_, err = GetMonsterGaugeRegistry().Add(t, Gauge{id: mxml.Name, exists: true})
		if err != nil {
			return err
		}
	}
	return nil
}
