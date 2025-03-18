package monster

import (
	"atlas-data/document"
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-tenant"
	"strconv"
	"sync"
)

type MonsterString struct {
	id   uint32
	name string
}

func (m MonsterString) GetId() uint32 {
	return m.id
}

func (m MonsterString) Name() string {
	return m.name
}

var msReg *document.Registry[uint32, MonsterString]
var msOnce sync.Once

func GetMonsterStringRegistry() *document.Registry[uint32, MonsterString] {
	msOnce.Do(func() {
		msReg = document.NewRegistry[uint32, MonsterString]()
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
		_, err = GetMonsterStringRegistry().Add(t, MonsterString{
			id:   uint32(id),
			name: mxml.GetString("name", "MISSINGNO"),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
