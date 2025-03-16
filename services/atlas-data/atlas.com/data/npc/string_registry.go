package npc

import (
	"atlas-data/registry"
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-tenant"
	"strconv"
	"sync"
)

type NpcString struct {
	id   uint32
	name string
}

func (m NpcString) Id() uint32 {
	return m.id
}

func (m NpcString) Name() string {
	return m.name
}

var nsReg *registry.Registry[uint32, NpcString]
var nsOnce sync.Once

func GetNpcStringRegistry() *registry.Registry[uint32, NpcString] {
	nsOnce.Do(func() {
		nsReg = registry.NewRegistry[uint32, NpcString]()
	})
	return nsReg
}

func InitString(t tenant.Model, path string) error {
	exml, err := xml.Read(path)
	if err != nil {
		return err
	}

	for _, mxml := range exml.ChildNodes {
		var id int
		id, err = strconv.Atoi(mxml.Name)
		if err != nil {
			return err
		}
		err = GetNpcStringRegistry().Add(t, NpcString{
			id:   uint32(id),
			name: mxml.GetString("name", "MISSINGNO"),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
