package skill

import (
	"atlas-data/document"
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type SkillString struct {
	id   string
	name string
}

func (s SkillString) GetID() string {
	return s.id
}

func (s SkillString) Name() string {
	return s.name
}

var ssReg *document.Registry[string, SkillString]
var ssOnce sync.Once

func GetSkillStringRegistry() *document.Registry[string, SkillString] {
	ssOnce.Do(func() {
		ssReg = document.NewRegistry[string, SkillString]()
	})
	return ssReg
}

func InitString(t tenant.Model, path string) error {
	exml, err := xml.Read(path)
	if err != nil {
		return err
	}

	// All entries (job categories and skills) are direct children of the root.
	// Job categories have "bookName", skills have "name".
	// We only want entries with a "name" attribute.
	for _, node := range exml.ChildNodes {
		name := node.GetString("name", "")
		if name == "" {
			// Skip job category entries (they have "bookName" instead of "name")
			continue
		}
		_, err = GetSkillStringRegistry().Add(t, SkillString{
			id:   node.Name,
			name: name,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
