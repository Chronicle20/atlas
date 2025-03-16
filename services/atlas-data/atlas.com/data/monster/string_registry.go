package monster

import (
	"atlas-data/registry"
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-tenant"
	"strconv"
	"sync"
)

type MonsterString struct {
	id   uint32
	name string
}

func (m MonsterString) Id() uint32 {
	return m.id
}

func (m MonsterString) Name() string {
	return m.name
}

var msReg *registry.Registry[uint32, MonsterString]
var msOnce sync.Once

func GetMonsterStringRegistry() *registry.Registry[uint32, MonsterString] {
	msOnce.Do(func() {
		msReg = registry.NewRegistry[uint32, MonsterString]()
	})
	return msReg
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
		err = GetMonsterStringRegistry().Add(t, MonsterString{
			id:   uint32(id),
			name: mxml.GetString("name", "MISSINGNO"),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
