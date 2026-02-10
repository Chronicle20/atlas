package item

import (
	"atlas-data/document"
	"atlas-data/xml"
	"sync"

	"github.com/Chronicle20/atlas-tenant"
)

type ItemString struct {
	id   string
	name string
}

func (m ItemString) GetID() string {
	return m.id
}

func (m ItemString) Name() string {
	return m.name
}

var isReg *document.Registry[string, ItemString]
var isOnce sync.Once

func GetItemStringRegistry() *document.Registry[string, ItemString] {
	isOnce.Do(func() {
		isReg = document.NewRegistry[string, ItemString]()
	})
	return isReg
}

func InitStringFlat(t tenant.Model, path string) error {
	exml, err := xml.Read(path)
	if err != nil {
		return err
	}

	for _, mxml := range exml.ChildNodes {
		_, err = GetItemStringRegistry().Add(t, ItemString{
			id:   mxml.Name,
			name: mxml.GetString("name", "MISSINGNO"),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func InitStringNested(t tenant.Model, path string) error {
	exml, err := xml.Read(path)
	if err != nil {
		return err
	}

	for _, cat := range exml.ChildNodes {
		for _, subCat := range cat.ChildNodes {
			for _, mxml := range subCat.ChildNodes {
				_, err = GetItemStringRegistry().Add(t, ItemString{
					id:   mxml.Name,
					name: mxml.GetString("name", "MISSINGNO"),
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
