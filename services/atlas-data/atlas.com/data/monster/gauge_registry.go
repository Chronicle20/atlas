package monster

import (
	"atlas-data/document"
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-tenant"
	"strconv"
	"sync"
)

type Gauge struct {
	id     uint32
	exists bool
}

func (g Gauge) GetId() uint32 {
	return g.id
}

func (g Gauge) Exists() bool {
	return g.exists
}

var mgReg *document.Registry[uint32, Gauge]
var mgOnce sync.Once

func GetMonsterGaugeRegistry() *document.Registry[uint32, Gauge] {
	mgOnce.Do(func() {
		mgReg = document.NewRegistry[uint32, Gauge]()
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
		var id int
		id, err = strconv.Atoi(mxml.Name)
		if err != nil {
			return err
		}
		_, err = GetMonsterGaugeRegistry().Add(t, Gauge{id: uint32(id), exists: true})
		if err != nil {
			return err
		}
	}
	return nil
}
