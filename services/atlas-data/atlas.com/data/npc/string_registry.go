package npc

import (
	"atlas-data/document"
	"atlas-data/xml"
	"fmt"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	exml, err := xml.FromPathProvider(path)()
	if err != nil {
		return fmt.Errorf("npc.InitString FromPathProvider %s: %w", path, err)
	}

	added := 0
	for _, mxml := range exml.ChildNodes {
		var id int
		id, err = strconv.Atoi(mxml.Name)
		if err != nil {
			return fmt.Errorf("npc.InitString parse id %q: %w", mxml.Name, err)
		}
		_, err = GetNpcStringRegistry().Add(t, NpcString{
			id:   strconv.Itoa(id),
			name: mxml.GetString("name", "MISSINGNO"),
		})
		if err != nil {
			return err
		}
		added++
	}
	// Diagnostic: PR-544 saw 1620 NPC docs all with empty name even though no
	// warning was logged here. If `added` is zero, the XML root had no
	// imgdir children — points at a wztoxml output mismatch or a race that
	// truncated the file mid-write.
	logrus.StandardLogger().Infof("npc.InitString: tenant=%s read_children=%d added=%d path=%s", t.Id().String(), len(exml.ChildNodes), added, path)
	return nil
}
