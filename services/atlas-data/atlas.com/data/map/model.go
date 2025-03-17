package _map

import (
	"atlas-data/map/monster"
	"atlas-data/map/npc"
	"atlas-data/map/portal"
	"atlas-data/map/reactor"
	"atlas-data/point"
	"math"
	"sort"
)

type Model struct {
	Id                uint32           `json:"id"`
	Name              string           `json:"name"`
	StreetName        string           `json:"street_name"`
	ReturnMapId       uint32           `json:"return_map_id"`
	MonsterRate       float64          `json:"monster_rate"`
	OnFirstUserEnter  string           `json:"on_first_user_enter"`
	OnUserEnter       string           `json:"on_user_enter"`
	FieldLimit        uint32           `json:"field_limit"`
	MobInterval       uint32           `json:"mob_interval"`
	Portals           []portal.Model   `json:"portals"`
	TimeMob           *TimeMob         `json:"time_mob"`
	MapArea           Rectangle        `json:"map_area"`
	FootholdTree      *FootholdTree    `json:"foothold_tree"`
	Areas             []Rectangle      `json:"areas"`
	Seats             uint32           `json:"seats"`
	Clock             bool             `json:"clock"`
	EverLast          bool             `json:"ever_last"`
	Town              bool             `json:"town"`
	DecHp             uint32           `json:"dec_hp"`
	ProtectItem       uint32           `json:"protect_item"`
	ForcedReturnMapId uint32           `json:"forced_return_map_id"`
	Boat              bool             `json:"boat"`
	TimeLimit         int32            `json:"time_limit"`
	FieldType         uint32           `json:"field_type"`
	MobCapacity       uint32           `json:"mob_capacity"`
	Recovery          float64          `json:"recovery"`
	BackgroundTypes   []BackgroundType `json:"background_types"`
	XLimit            XLimit           `json:"x_limit"`
	Reactors          []reactor.Model  `json:"reactors"`
	Npcs              []npc.Model      `json:"npcs"`
	Monsters          []monster.Model  `json:"monsters"`
}

// Keep accessor method for id since field is private
func (m Model) GetId() uint32 {
	return m.Id
}

type TimeMob struct {
	Id      uint32 `json:"id"`
	Message string `json:"message"`
}

type Rectangle struct {
	X      int16 `json:"x"`
	Y      int16 `json:"y"`
	Width  int16 `json:"width"`
	Height int16 `json:"height"`
}

func (r Rectangle) contains(ret point.Model) bool {
	w := r.Width
	h := r.Height
	if (w | h) < 0 {
		// At least one of the dimensions is negative...
		return false
	}
	// Note: if either dimension is zero, tests below must return false...
	x := r.X
	y := r.Y

	if ret.X < x || ret.Y < y {
		return false
	}
	w += x
	h += y
	//    overflow || intersect
	return (w < x || w > ret.X) &&
		(h < y || h > ret.Y)
}

type FootholdTree struct {
	NorthWest *FootholdTree `json:"north_west,omitempty"`
	NorthEast *FootholdTree `json:"north_east,omitempty"`
	SouthWest *FootholdTree `json:"south_west,omitempty"`
	SouthEast *FootholdTree `json:"south_east,omitempty"`
	Footholds []Foothold    `json:"footholds"`
	P1        *point.Model  `json:"p1,omitempty"`
	P2        *point.Model  `json:"p2,omitempty"`
	Center    *point.Model  `json:"center,omitempty"`
	Depth     uint32        `json:"depth"`
	MaxDropX  int16         `json:"max_drop_x"`
	MinDropX  int16         `json:"min_drop_x"`
}

func (f *FootholdTree) findBelow(initial *point.Model) *Foothold {
	relevants := f.GetRelevant(initial)
	matches := make([]Foothold, 0)

	for _, fh := range relevants {
		if fh.First.X <= initial.X && fh.Second.X >= initial.X {
			matches = append(matches, fh)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Second.Y < matches[j].First.Y {
			return true
		}
		return false
	})
	for _, fh := range matches {
		if !fh.isWall() {
			if fh.First.Y != fh.Second.Y {
				s1 := math.Abs(float64(fh.Second.Y - fh.First.Y))
				s2 := math.Abs(float64(fh.Second.X - fh.First.X))
				s4 := math.Abs(float64(initial.X - fh.First.X))
				alpha := math.Atan(s2 / s1)
				beta := math.Atan(s1 / s2)
				s5 := math.Cos(alpha) * (s4 / math.Cos(beta))
				var calcY int16
				if fh.Second.Y < fh.First.Y {
					calcY = fh.First.Y - int16(s5)
				} else {
					calcY = fh.First.Y + int16(s5)
				}
				if calcY >= initial.Y {
					return &fh
				}
			} else {
				if fh.First.Y >= initial.Y {
					return &fh
				}
			}
		}
	}
	return nil
}

func (f *FootholdTree) GetRelevant(point *point.Model) []Foothold {
	results := make([]Foothold, 0)
	results = append(results, f.Footholds...)

	if f.NorthWest != nil {
		if point.X <= f.Center.X && point.Y <= f.Center.Y {
			results = append(results, f.NorthWest.GetRelevant(point)...)
		} else if point.X > f.Center.X && point.Y <= f.Center.Y {
			results = append(results, f.NorthEast.GetRelevant(point)...)
		} else if point.X <= f.Center.X && point.Y > f.Center.Y {
			results = append(results, f.SouthWest.GetRelevant(point)...)
		} else {
			results = append(results, f.SouthEast.GetRelevant(point)...)
		}
	}
	return results
}

type Foothold struct {
	Id     uint32       `json:"id"`
	First  *point.Model `json:"first,omitempty"`
	Second *point.Model `json:"second,omitempty"`
}

func (f Foothold) isWall() bool {
	return f.First.X == f.Second.X
}

type BackgroundType struct {
	LayerNumber    uint32 `json:"layer_number"`
	BackgroundType uint32 `json:"background_type"`
}

type XLimit struct {
	Min uint32 `json:"min"`
	Max uint32 `json:"max"`
}
