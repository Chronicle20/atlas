package skill

import (
	"atlas-data/document"
	"atlas-data/xml"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type SkillString struct {
	id   string
	name string
	desc string
}

func (s SkillString) GetID() string {
	return s.id
}

func (s SkillString) Name() string {
	return s.name
}

func (s SkillString) Desc() string {
	return s.desc
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
	exml, err := xml.FromPathProvider(path)()
	if err != nil {
		return err
	}

	// All entries (job categories and skills) are direct children of the root.
	// Job categories have "bookName", skills have "name".
	// We only want entries with a "name" attribute.
	added := 0
	for _, node := range exml.ChildNodes {
		name := node.GetString("name", "")
		if name == "" {
			// Skip job category entries (they have "bookName" instead of "name")
			continue
		}
		_, err = GetSkillStringRegistry().Add(t, SkillString{
			id:   node.Name,
			name: name,
			desc: node.GetString("desc", ""),
		})
		if err != nil {
			return err
		}
		added++
	}
	logrus.StandardLogger().Infof("skill.InitString: tenant=%s read_children=%d added=%d path=%s", t.Id().String(), len(exml.ChildNodes), added, path)
	return nil
}
