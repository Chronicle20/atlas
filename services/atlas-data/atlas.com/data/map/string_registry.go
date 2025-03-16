package _map

import (
	"atlas-data/registry"
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-tenant"
	"strconv"
	"sync"
)

type MapString struct {
	id         uint32
	mapName    string
	streetName string
}

func (m MapString) Id() uint32 {
	return m.id
}

func (m MapString) MapName() string {
	return m.mapName
}

func (m MapString) StreetName() string {
	return m.streetName
}

var msRg *registry.Registry[uint32, MapString]
var msOnce sync.Once

func GetMapStringRegistry() *registry.Registry[uint32, MapString] {
	msOnce.Do(func() {
		msRg = registry.NewRegistry[uint32, MapString]()
	})
	return msRg
}

func InitString(t tenant.Model, path string) error {
	exml, err := xml.Read(path)
	if err != nil {
		return err
	}

	for _, cat := range exml.ChildNodes {
		for _, mxml := range cat.ChildNodes {
			var id int
			id, err = strconv.Atoi(mxml.Name)
			if err != nil {
				return err
			}
			err = GetMapStringRegistry().Add(t, MapString{
				id:         uint32(id),
				mapName:    mxml.GetString("mapName", ""),
				streetName: mxml.GetString("streetName", ""),
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
