package mobskill

import (
	"atlas-data/document"
	"atlas-data/xml"
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type MobSkillString struct {
	id   string
	name string
}

func (m MobSkillString) GetID() string {
	return m.id
}

func (m MobSkillString) Name() string {
	return m.name
}

var msReg *document.Registry[string, MobSkillString]
var msOnce sync.Once

func GetMobSkillStringRegistry() *document.Registry[string, MobSkillString] {
	msOnce.Do(func() {
		msReg = document.NewRegistry[string, MobSkillString]()
	})
	return msReg
}

func InitString(t tenant.Model, path string) error {
	exml, err := xml.FromPathProvider(path)()
	if err != nil {
		return err
	}

	for _, mxml := range exml.ChildNodes {
		_, err = GetMobSkillStringRegistry().Add(t, MobSkillString{
			id:   mxml.Name,
			name: mxml.GetString("name", "MISSINGNO"),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
