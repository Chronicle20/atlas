package _map

import (
	"atlas-data/document"
	"atlas-data/map/monster"
	"atlas-data/map/npc"
	"atlas-data/map/portal"
	"atlas-data/map/reactor"
	"atlas-data/point"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
	"strconv"
)

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[uint32, Model] {
	return document.NewStorage(l, db, GetModelRegistry(), "MAP")
}

func Register(s *document.Storage[uint32, Model]) func(ctx context.Context) func(r model.Provider[Model]) error {
	return func(ctx context.Context) func(r model.Provider[Model]) error {
		return func(r model.Provider[Model]) error {
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

// deprecated
func RegisterMap(db *gorm.DB) func(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
		return func(ctx context.Context) func(path string) {
			return func(path string) {
				_ = Register(NewStorage(l, db))(ctx)(ReadFromFile(l)(ctx)(path))
			}
		}
	}
}

func bSearchDropPos(tree *FootholdTree, initial *point.Model, fallback *point.Model) *point.Model {
	var dropPos *point.Model
	awayX := fallback.X
	homeX := initial.X
	y := initial.Y - 85

	for math.Abs(float64(homeX-awayX)) > 5 {
		distanceX := awayX - homeX
		dx := distanceX / 2
		searchX := homeX + dx
		res := calcPointBelow(tree, &point.Model{X: searchX, Y: y})
		if res != nil {
			awayX = searchX
			dropPos = res
		} else {
			homeX = searchX
		}
	}

	if dropPos != nil {
		return dropPos
	}
	return fallback
}

func calcPointBelow(tree *FootholdTree, initial *point.Model) *point.Model {
	fh := tree.findBelow(initial)
	if fh == nil {
		return nil
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
	ret := &point.Model{X: initial.X, Y: dropY}
	return ret
}

type FootholdTreeConfigurator func(f *FootholdTree)

func NewFootholdTree(lx int16, ly int16, ux int16, uy int16, configurations ...FootholdTreeConfigurator) *FootholdTree {
	p1x := lx
	p1y := ly
	p2x := ux
	p2y := uy
	centerx := int16(math.Round(float64(ux-lx) / 2))
	centery := int16(math.Round(float64(uy-ly) / 2))
	ft := &FootholdTree{
		NorthWest: nil,
		NorthEast: nil,
		SouthWest: nil,
		SouthEast: nil,
		Footholds: make([]Foothold, 0),
		P1:        &point.Model{X: p1x, Y: p1y},
		P2:        &point.Model{X: p2x, Y: p2y},
		Center:    &point.Model{X: centerx, Y: centery},
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
	return func(f *FootholdTree) {
		f.Depth = depth
	}
}

func (f *FootholdTree) Insert(footholds []Foothold) *FootholdTree {
	for _, foothold := range footholds {
		f.InsertSingle(foothold)
	}
	return f
}

func (f *FootholdTree) InsertSingle(foothold Foothold) *FootholdTree {
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

func portalProvider(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32) model.Provider[[]portal.Model] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[[]portal.Model] {
		return func(mapId uint32) model.Provider[[]portal.Model] {
			m, err := s.ByIdProvider(ctx)(mapId)()
			if err != nil {
				return model.ErrorProvider[[]portal.Model](err)
			}
			return model.FixedProvider(m.Portals)
		}
	}
}

func GetPortals(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32) ([]portal.Model, error) {
	return func(ctx context.Context) func(mapId uint32) ([]portal.Model, error) {
		return func(mapId uint32) ([]portal.Model, error) {
			return portalProvider(s)(ctx)(mapId)()
		}
	}
}

func GetPortalsByName(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32, name string) ([]portal.Model, error) {
	return func(ctx context.Context) func(mapId uint32, name string) ([]portal.Model, error) {
		return func(mapId uint32, name string) ([]portal.Model, error) {
			return model.FilteredProvider(portalProvider(s)(ctx)(mapId), model.Filters(PortalNameFilter(name)))()
		}
	}
}

func GetPortalById(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32, portalId uint32) (portal.Model, error) {
	return func(ctx context.Context) func(mapId uint32, portalId uint32) (portal.Model, error) {
		return func(mapId uint32, portalId uint32) (portal.Model, error) {
			return model.First(portalProvider(s)(ctx)(mapId), model.Filters(PortalIdFilter(portalId)))
		}
	}
}

func PortalNameFilter(portalName string) model.Filter[portal.Model] {
	return func(p portal.Model) bool {
		return p.Name == portalName
	}
}

func PortalIdFilter(portalId uint32) model.Filter[portal.Model] {
	return func(p portal.Model) bool {
		return p.Id == strconv.Itoa(int(portalId))
	}
}

func reactorProvider(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32) model.Provider[[]reactor.Model] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[[]reactor.Model] {
		return func(mapId uint32) model.Provider[[]reactor.Model] {
			m, err := s.ByIdProvider(ctx)(mapId)()
			if err != nil {
				return model.ErrorProvider[[]reactor.Model](err)
			}
			return model.FixedProvider(m.Reactors)
		}
	}
}

func GetReactors(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32) ([]reactor.Model, error) {
	return func(ctx context.Context) func(mapId uint32) ([]reactor.Model, error) {
		return func(mapId uint32) ([]reactor.Model, error) {
			return reactorProvider(s)(ctx)(mapId)()
		}
	}
}

func npcProvider(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32) model.Provider[[]npc.Model] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[[]npc.Model] {
		return func(mapId uint32) model.Provider[[]npc.Model] {
			m, err := s.ByIdProvider(ctx)(mapId)()
			if err != nil {
				return model.ErrorProvider[[]npc.Model](err)
			}
			return model.FixedProvider(m.Npcs)
		}
	}
}

func GetNpcs(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32) ([]npc.Model, error) {
	return func(ctx context.Context) func(mapId uint32) ([]npc.Model, error) {
		return func(mapId uint32) ([]npc.Model, error) {
			return npcProvider(s)(ctx)(mapId)()
		}
	}
}

func GetNpcsByObjectId(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32, objectId uint32) ([]npc.Model, error) {
	return func(ctx context.Context) func(mapId uint32, objectId uint32) ([]npc.Model, error) {
		return func(mapId uint32, objectId uint32) ([]npc.Model, error) {
			return model.FilteredProvider(npcProvider(s)(ctx)(mapId), model.Filters(NPCObjectIdFilter(objectId)))()
		}
	}
}

func GetNpc(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32, npcId uint32) (npc.Model, error) {
	return func(ctx context.Context) func(mapId uint32, npcId uint32) (npc.Model, error) {
		return func(mapId uint32, npcId uint32) (npc.Model, error) {
			return model.First(npcProvider(s)(ctx)(mapId), model.Filters(NPCIdFilter(npcId)))
		}
	}
}

func NPCIdFilter(id uint32) model.Filter[npc.Model] {
	return func(n npc.Model) bool {
		return n.Id == strconv.Itoa(int(id))
	}
}

func NPCObjectIdFilter(id uint32) model.Filter[npc.Model] {
	return func(n npc.Model) bool {
		return n.Id == strconv.Itoa(int(id))
	}
}

func monsterProvider(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32) model.Provider[[]monster.Model] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[[]monster.Model] {
		return func(mapId uint32) model.Provider[[]monster.Model] {
			m, err := s.ByIdProvider(ctx)(mapId)()
			if err != nil {
				return model.ErrorProvider[[]monster.Model](err)
			}
			return model.FixedProvider(m.Monsters)
		}
	}
}

func GetMonsters(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32) ([]monster.Model, error) {
	return func(ctx context.Context) func(mapId uint32) ([]monster.Model, error) {
		return func(mapId uint32) ([]monster.Model, error) {
			return monsterProvider(s)(ctx)(mapId)()
		}
	}
}

func calcDropPos(s *document.Storage[uint32, Model]) func(ctx context.Context) func(mapId uint32, initial *point.Model, fallback *point.Model) *point.Model {
	return func(ctx context.Context) func(mapId uint32, initial *point.Model, fallback *point.Model) *point.Model {
		return func(mapId uint32, initial *point.Model, fallback *point.Model) *point.Model {
			m, err := s.GetById(ctx)(mapId)
			if err != nil {
				return fallback
			}

			rp := initial
			if rp.X < int16(m.XLimit.Min) {
				rp.X = int16(m.XLimit.Min)
			} else if rp.X > int16(m.XLimit.Max) {
				rp.X = int16(m.XLimit.Max)
			}
			ret := calcPointBelow(m.FootholdTree, &point.Model{X: rp.X, Y: rp.Y - 85})
			if ret == nil {
				ret = bSearchDropPos(m.FootholdTree, initial, fallback)
			}
			if !m.MapArea.contains(*ret) {
				return fallback
			}
			return ret
		}
	}
}
