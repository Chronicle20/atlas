package npc

import (
	"atlas-data/document"
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-tenant"
	"strconv"
	"sync"
)

type NpcString struct {
	id   string
	name string
}

func NewNpcString(id string, name string) NpcString {
	return NpcString{
		id:   id,
		name: name,
	}
}

func (m NpcString) GetID() string {
	return m.id
}

func (m NpcString) Name() string {
	return m.name
}

var nsReg *document.Registry[string, NpcString]
var nsOnce sync.Once

func GetNpcStringRegistry() *document.Registry[string, NpcString] {
	nsOnce.Do(func() {
		nsReg = document.NewRegistry[string, NpcString]()
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
		_, err = GetNpcStringRegistry().Add(t, NpcString{
			id:   strconv.Itoa(id),
			name: mxml.GetString("name", "MISSINGNO"),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
