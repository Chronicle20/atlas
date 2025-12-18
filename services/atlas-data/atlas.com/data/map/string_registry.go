package _map

import (
	"atlas-data/document"
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type MapString struct {
	id         string
	mapName    string
	streetName string
}

func (m MapString) GetID() string {
	return m.id
}

func (m MapString) MapName() string {
	return m.mapName
}

func (m MapString) StreetName() string {
	return m.streetName
}

var msRg *document.Registry[string, MapString]
var msOnce sync.Once

func GetMapStringRegistry() *document.Registry[string, MapString] {
	msOnce.Do(func() {
		msRg = document.NewRegistry[string, MapString]()
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
			_, err = GetMapStringRegistry().Add(t, MapString{
				id:         mxml.Name,
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
