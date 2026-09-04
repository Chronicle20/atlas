package _map

import (
	"atlas-data/document"
	"atlas-data/xml"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// The two object kinds. These literal strings are a cross-service contract:
// task-278's environment-object kind uses the same two values, and the UI
// merges the two collections on the composite key "{kind}:{name}".
const (
	ObjKindEnvironment = "ENVIRONMENT"
	ObjKindObstacle    = "OBSTACLE"
)

// MapObjectDefinition is one Map.wz/Obj definition node keyed by
// "{oS}/{l0}/{l1}/{l2}". Only nodes carrying obstacle=1 are indexed; absence
// from the registry means ENVIRONMENT.
type MapObjectDefinition struct {
	id       string
	obstacle bool
}

func (m MapObjectDefinition) GetID() string { return m.id }

func (m MapObjectDefinition) Obstacle() bool { return m.obstacle }

var (
	moRg   *document.Registry[string, MapObjectDefinition]
	moOnce sync.Once
)

func GetMapObjectRegistry() *document.Registry[string, MapObjectDefinition] {
	moOnce.Do(func() {
		moRg = document.NewRegistry[string, MapObjectDefinition]()
	})
	return moRg
}

// InitObj walks dir (= {root}/Map.wz/Obj) once and indexes every
// {l0}/{l1}/{l2} node carrying obstacle=1. Doing this per-map inside the
// reader would re-parse the same Obj images thousands of times across a
// 5261-map ingest. It returns the number of obstacle definitions indexed.
func InitObj(t tenant.Model, dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	indexed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".img.xml") {
			continue
		}
		oS := strings.TrimSuffix(e.Name(), ".img.xml")
		exml, err := xml.FromPathProvider(filepath.Join(dir, e.Name()))()
		if err != nil {
			return 0, err
		}
		for _, l0 := range exml.ChildNodes {
			for _, l1 := range l0.ChildNodes {
				for _, l2 := range l1.ChildNodes {
					if l2.GetIntegerWithDefault("obstacle", 0) != 1 {
						continue
					}
					id := strings.Join([]string{oS, l0.Name, l1.Name, l2.Name}, "/")
					if _, err := GetMapObjectRegistry().Add(t, MapObjectDefinition{id: id, obstacle: true}); err != nil {
						return 0, err
					}
					indexed++
				}
			}
		}
	}
	return indexed, nil
}

// ResolveObjKind returns OBSTACLE when the referenced Obj definition carries
// obstacle=1, ENVIRONMENT otherwise. An uninitialised or empty registry
// therefore resolves everything to ENVIRONMENT, which is the correct default.
func ResolveObjKind(t tenant.Model, oS string, l0 string, l1 string, l2 string) string {
	d, err := GetMapObjectRegistry().Get(t, strings.Join([]string{oS, l0, l1, l2}, "/"))
	if err != nil || !d.Obstacle() {
		return ObjKindEnvironment
	}
	return ObjKindObstacle
}
