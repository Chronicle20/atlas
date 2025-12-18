package monster

import (
	"atlas-data/document"
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type MonsterString struct {
	id   string
	name string
}

func (m MonsterString) GetID() string {
	return m.id
}

func (m MonsterString) Name() string {
	return m.name
}

var msReg *document.Registry[string, MonsterString]
var msOnce sync.Once

func GetMonsterStringRegistry() *document.Registry[string, MonsterString] {
	msOnce.Do(func() {
		msReg = document.NewRegistry[string, MonsterString]()
	})
	return msReg
}

func InitString(t tenant.Model, path string) error {
	exml, err := xml.Read(path)
	if err != nil {
		return err
	}

	for _, mxml := range exml.ChildNodes {
		_, err = GetMonsterStringRegistry().Add(t, MonsterString{
			id:   mxml.Name,
			name: mxml.GetString("name", "MISSINGNO"),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
