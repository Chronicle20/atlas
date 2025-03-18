package _map

import (
	"atlas-data/document"
	"atlas-data/map/monster"
	"atlas-data/map/npc"
	"atlas-data/map/portal"
	"atlas-data/map/reactor"
	"atlas-data/point"
	"atlas-data/xml"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel] {
	return document.NewStorage(l, db, GetModelRegistry(), "MAP")
}

func Register(s *document.Storage[string, RestModel]) func(ctx context.Context) func(r model.Provider[RestModel]) error {
	return func(ctx context.Context) func(r model.Provider[RestModel]) error {
		return func(r model.Provider[RestModel]) error {
			m, err := r()
			if err != nil {
				return err
			}
			_, err = s.Add(ctx)(m)()
			if err != nil {
				return err
			}
			return nil
		}
	}
}

func extractPathAndID(path string) (string, uint32, error) {
	// Extract the base filename
	base := filepath.Base(path)

	// Trim the ".img.xml" extension
	if !strings.HasSuffix(base, ".img.xml") {
		return "", 0, fmt.Errorf("invalid file format: %s", base)
	}
	idStr := strings.TrimSuffix(base, ".img.xml")

	// Convert to uint32
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return "", 0, fmt.Errorf("failed to convert ID to uint32: %w", err)
	}

	// Extract the directory
	dir := filepath.Dir(path) + "/"

	return dir, uint32(id), nil
}

func RegisterMap(db *gorm.DB) func(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
		return func(ctx context.Context) func(path string) {
			return func(path string) {
				parentPath, mapId, err := extractPathAndID(path)
				if err != nil {
					return
				}
				_ = Register(NewStorage(l, db))(ctx)(Read(l)(ctx)(parentPath, mapId, xml.FromParentPathProvider(9)))
			}
		}
	}
}

func bSearchDropPos(tree FootholdTreeRestModel, initial point.RestModel, fallback point.RestModel) point.RestModel {
	set := false
	var dropPos point.RestModel
	awayX := fallback.X
	homeX := initial.X
	y := initial.Y - 85

	for {
		distanceX := awayX - homeX
		dx := distanceX / 2
		searchX := homeX + dx
		res, ok := calcPointBelow(tree, point.RestModel{X: searchX, Y: y})
		if ok {
			awayX = searchX
			dropPos = res
			set = true
		} else {
			homeX = searchX
		}

		if math.Abs(float64(homeX-awayX)) < 5 {
			break
		}
	}

	if set {
		return dropPos
	}
	return fallback
}

func calcPointBelow(tree FootholdTreeRestModel, initial point.RestModel) (point.RestModel, bool) {
	fh := tree.findBelow(initial)
	if fh == nil {
		return point.RestModel{}, false
	}

	dropY := fh.First.Y
	if !fh.isWall() && fh.First.Y != fh.Second.Y {
		s1 := math.Abs(float64(fh.Second.Y - fh.First.Y))
		s2 := math.Abs(float64(fh.Second.X - fh.First.X))
		s5 := math.Cos(math.Atan(s2/s1)) * (math.Abs(float64(initial.X-fh.First.X)) / math.Cos(math.Atan(s1/s2)))
		if fh.Second.Y < fh.First.Y {
			dropY = fh.First.Y - int16(s5)
		} else {
			dropY = fh.First.Y + int16(s5)
		}
	}
	return point.RestModel{X: initial.X, Y: dropY}, true
}

type FootholdTreeConfigurator func(f *FootholdTreeRestModel)

func NewFootholdTree(lx int16, ly int16, ux int16, uy int16, configurations ...FootholdTreeConfigurator) *FootholdTreeRestModel {
	p1x := lx
	p1y := ly
	p2x := ux
	p2y := uy
	centerx := int16(math.Round(float64(ux-lx) / 2))
	centery := int16(math.Round(float64(uy-ly) / 2))
	ft := &FootholdTreeRestModel{
		NorthWest: nil,
		NorthEast: nil,
		SouthWest: nil,
		SouthEast: nil,
		Footholds: make([]FootholdRestModel, 0),
		P1:        &point.RestModel{X: p1x, Y: p1y},
		P2:        &point.RestModel{X: p2x, Y: p2y},
		Center:    &point.RestModel{X: centerx, Y: centery},
		Depth:     0,
		MaxDropX:  0,
		MinDropX:  0,
	}

	for _, configurator := range configurations {
		configurator(ft)
	}
	return ft
}

func SetFootholdTreeDepth(depth uint32) FootholdTreeConfigurator {
	return func(f *FootholdTreeRestModel) {
		f.Depth = depth
	}
}

func (f *FootholdTreeRestModel) Insert(footholds []FootholdRestModel) *FootholdTreeRestModel {
	for _, foothold := range footholds {
		f.InsertSingle(foothold)
	}
	return f
}

func (f *FootholdTreeRestModel) InsertSingle(foothold FootholdRestModel) *FootholdTreeRestModel {
	if f.Depth == 0 {
		if foothold.First.X > f.MaxDropX {
			f.MaxDropX = foothold.First.X
		}
		if foothold.First.X < f.MinDropX {
			f.MinDropX = foothold.First.X
		}
		if foothold.Second.X > f.MaxDropX {
			f.MaxDropX = foothold.Second.X
		}
		if foothold.Second.X < f.MinDropX {
			f.MinDropX = foothold.Second.X
		}

	}
	if f.Depth == 8 || foothold.First.X >= f.P1.X && foothold.Second.X <= f.P2.X && foothold.First.Y >= f.P1.Y && foothold.Second.Y <= f.P2.Y {
		f.Footholds = append(f.Footholds, foothold)
	} else {
		if f.NorthWest == nil {
			f.NorthWest = NewFootholdTree(f.P1.X, f.P1.Y, f.Center.X, f.Center.Y, SetFootholdTreeDepth(f.Depth+1))
			f.NorthEast = NewFootholdTree(f.Center.X, f.P1.Y, f.P2.X, f.Center.Y, SetFootholdTreeDepth(f.Depth+1))
			f.SouthWest = NewFootholdTree(f.P1.X, f.Center.Y, f.Center.X, f.P2.Y, SetFootholdTreeDepth(f.Depth+1))
			f.SouthEast = NewFootholdTree(f.Center.X, f.Center.Y, f.P2.X, f.P2.Y, SetFootholdTreeDepth(f.Depth+1))
		}
		if foothold.Second.X <= f.Center.X && foothold.Second.Y <= f.Center.Y {
			f.NorthWest = f.NorthWest.InsertSingle(foothold)
		} else if foothold.First.X > f.Center.X && foothold.Second.Y <= f.Center.Y {
			f.NorthEast = f.NorthEast.InsertSingle(foothold)
		} else if foothold.Second.X <= f.Center.X && foothold.First.Y > f.Center.Y {
			f.SouthWest = f.SouthWest.InsertSingle(foothold)
		} else {
			f.SouthEast = f.SouthEast.InsertSingle(foothold)
		}
	}
	return f
}

func portalProvider(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32) model.Provider[[]portal.RestModel] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[[]portal.RestModel] {
		return func(mapId uint32) model.Provider[[]portal.RestModel] {
			m, err := s.ByIdProvider(ctx)(strconv.Itoa(int(mapId)))()
			if err != nil {
				return model.ErrorProvider[[]portal.RestModel](err)
			}
			return model.FixedProvider(m.Portals)
		}
	}
}

func GetPortals(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32) ([]portal.RestModel, error) {
	return func(ctx context.Context) func(mapId uint32) ([]portal.RestModel, error) {
		return func(mapId uint32) ([]portal.RestModel, error) {
			return portalProvider(s)(ctx)(mapId)()
		}
	}
}

func GetPortalsByName(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32, name string) ([]portal.RestModel, error) {
	return func(ctx context.Context) func(mapId uint32, name string) ([]portal.RestModel, error) {
		return func(mapId uint32, name string) ([]portal.RestModel, error) {
			return model.FilteredProvider(portalProvider(s)(ctx)(mapId), model.Filters(PortalNameFilter(name)))()
		}
	}
}

func GetPortalById(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32, portalId uint32) (portal.RestModel, error) {
	return func(ctx context.Context) func(mapId uint32, portalId uint32) (portal.RestModel, error) {
		return func(mapId uint32, portalId uint32) (portal.RestModel, error) {
			return model.First(portalProvider(s)(ctx)(mapId), model.Filters(PortalIdFilter(portalId)))
		}
	}
}

func PortalNameFilter(portalName string) model.Filter[portal.RestModel] {
	return func(p portal.RestModel) bool {
		return p.Name == portalName
	}
}

func PortalIdFilter(portalId uint32) model.Filter[portal.RestModel] {
	return func(p portal.RestModel) bool {
		return p.Id == strconv.Itoa(int(portalId))
	}
}

func reactorProvider(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32) model.Provider[[]reactor.RestModel] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[[]reactor.RestModel] {
		return func(mapId uint32) model.Provider[[]reactor.RestModel] {
			m, err := s.ByIdProvider(ctx)(strconv.Itoa(int(mapId)))()
			if err != nil {
				return model.ErrorProvider[[]reactor.RestModel](err)
			}
			return model.FixedProvider(m.Reactors)
		}
	}
}

func GetReactors(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32) ([]reactor.RestModel, error) {
	return func(ctx context.Context) func(mapId uint32) ([]reactor.RestModel, error) {
		return func(mapId uint32) ([]reactor.RestModel, error) {
			return reactorProvider(s)(ctx)(mapId)()
		}
	}
}

func npcProvider(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32) model.Provider[[]npc.RestModel] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[[]npc.RestModel] {
		return func(mapId uint32) model.Provider[[]npc.RestModel] {
			m, err := s.ByIdProvider(ctx)(strconv.Itoa(int(mapId)))()
			if err != nil {
				return model.ErrorProvider[[]npc.RestModel](err)
			}
			return model.FixedProvider(m.NPCs)
		}
	}
}

func GetNpcs(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32) ([]npc.RestModel, error) {
	return func(ctx context.Context) func(mapId uint32) ([]npc.RestModel, error) {
		return func(mapId uint32) ([]npc.RestModel, error) {
			return npcProvider(s)(ctx)(mapId)()
		}
	}
}

func GetNpcsByObjectId(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32, objectId uint32) ([]npc.RestModel, error) {
	return func(ctx context.Context) func(mapId uint32, objectId uint32) ([]npc.RestModel, error) {
		return func(mapId uint32, objectId uint32) ([]npc.RestModel, error) {
			return model.FilteredProvider(npcProvider(s)(ctx)(mapId), model.Filters(NPCObjectIdFilter(objectId)))()
		}
	}
}

func GetNpc(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32, npcId uint32) (npc.RestModel, error) {
	return func(ctx context.Context) func(mapId uint32, npcId uint32) (npc.RestModel, error) {
		return func(mapId uint32, npcId uint32) (npc.RestModel, error) {
			return model.First(npcProvider(s)(ctx)(mapId), model.Filters(NPCIdFilter(npcId)))
		}
	}
}

func NPCIdFilter(id uint32) model.Filter[npc.RestModel] {
	return func(n npc.RestModel) bool {
		return n.Id == id
	}
}

func NPCObjectIdFilter(id uint32) model.Filter[npc.RestModel] {
	return func(n npc.RestModel) bool {
		return n.Id == id
	}
}

func monsterProvider(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32) model.Provider[[]monster.RestModel] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[[]monster.RestModel] {
		return func(mapId uint32) model.Provider[[]monster.RestModel] {
			m, err := s.ByIdProvider(ctx)(strconv.Itoa(int(mapId)))()
			if err != nil {
				return model.ErrorProvider[[]monster.RestModel](err)
			}
			return model.FixedProvider(m.Monsters)
		}
	}
}

func GetMonsters(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32) ([]monster.RestModel, error) {
	return func(ctx context.Context) func(mapId uint32) ([]monster.RestModel, error) {
		return func(mapId uint32) ([]monster.RestModel, error) {
			return monsterProvider(s)(ctx)(mapId)()
		}
	}
}

func calcDropPos(s *document.Storage[string, RestModel]) func(ctx context.Context) func(mapId uint32, initial point.RestModel, fallback point.RestModel) (point.RestModel, error) {
	return func(ctx context.Context) func(mapId uint32, initial point.RestModel, fallback point.RestModel) (point.RestModel, error) {
		return func(mapId uint32, initial point.RestModel, fallback point.RestModel) (point.RestModel, error) {
			m, err := s.GetById(ctx)(strconv.Itoa(int(mapId)))
			if err != nil {
				return fallback, nil
			}

			rp := initial
			if rp.X < int16(m.XLimit.Min) {
				rp.X = int16(m.XLimit.Min)
			} else if rp.X > int16(m.XLimit.Max) {
				rp.X = int16(m.XLimit.Max)
			}
			ret, ok := calcPointBelow(m.FootholdTree, point.RestModel{X: rp.X, Y: rp.Y - 85})
			if ok {
				ret = bSearchDropPos(m.FootholdTree, initial, fallback)
			}
			if !m.MapArea.contains(ret) {
				return fallback, nil
			}
			return ret, nil
		}
	}
}
