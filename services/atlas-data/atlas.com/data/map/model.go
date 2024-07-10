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
	id                uint32
	name              string
	streetName        string
	returnMapId       uint32
	monsterRate       float64
	onFirstUserEnter  string
	onUserEnter       string
	fieldLimit        uint32
	mobInterval       uint32
	portals           []portal.Model
	timeMob           *TimeMob
	mapArea           Rectangle
	footholdTree      *FootholdTree
	areas             []Rectangle
	seats             uint32
	clock             bool
	everLast          bool
	town              bool
	decHp             uint32
	protectItem       uint32
	forcedReturnMapId uint32
	boat              bool
	timeLimit         int32
	fieldType         uint32
	mobCapacity       uint32
	recovery          float64
	backgroundTypes   []BackgroundType
	xLimit            XLimit
	reactors          []reactor.Model
	npcs              []npc.Model
	monsters          []monster.Model
}

type TimeMob struct {
	id      uint32
	message string
}

type Rectangle struct {
	x      int16
	y      int16
	width  int16
	height int16
}

func (r Rectangle) contains(ret point.Model) bool {
	w := r.width
	h := r.height
	if (w | h) < 0 {
		// At least one of the dimensions is negative...
		return false
	}
	// Note: if either dimension is zero, tests below must return false...
	x := r.x
	y := r.y

	if ret.X() < x || ret.Y() < y {
		return false
	}
	w += x
	h += y
	//    overflow || intersect
	return (w < x || w > ret.X()) &&
		(h < y || h > ret.Y())
}

type FootholdTree struct {
	northWest *FootholdTree
	northEast *FootholdTree
	southWest *FootholdTree
	southEast *FootholdTree
	footholds []Foothold
	p1        *point.Model
	p2        *point.Model
	center    *point.Model
	depth     uint32
	maxDropX  int16
	minDropX  int16
}

func (f *FootholdTree) findBelow(initial *point.Model) *Foothold {
	relevants := f.GetRelevant(initial)
	matches := make([]Foothold, 0)

	for _, fh := range relevants {
		if fh.first.X() <= initial.X() && fh.second.X() >= initial.X() {
			matches = append(matches, fh)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].second.Y() < matches[j].first.Y() {
			return true
		}
		return false
	})
	for _, fh := range matches {
		if !fh.isWall() {
			if fh.first.Y() != fh.second.Y() {
				s1 := math.Abs(float64(fh.second.Y() - fh.first.Y()))
				s2 := math.Abs(float64(fh.second.X() - fh.first.X()))
				s4 := math.Abs(float64(initial.X() - fh.first.X()))
				alpha := math.Atan(s2 / s1)
				beta := math.Atan(s1 / s2)
				s5 := math.Cos(alpha) * (s4 / math.Cos(beta))
				var calcY int16
				if fh.second.Y() < fh.first.Y() {
					calcY = fh.first.Y() - int16(s5)
				} else {
					calcY = fh.first.Y() + int16(s5)
				}
				if calcY >= initial.Y() {
					return &fh
				}
			} else {
				if fh.first.Y() >= initial.Y() {
					return &fh
				}
			}
		}
	}
	return nil
}

func (f *FootholdTree) GetRelevant(point *point.Model) []Foothold {
	results := make([]Foothold, 0)
	results = append(results, f.footholds...)

	if f.northWest != nil {
		if point.X() <= f.center.X() && point.Y() <= f.center.Y() {
			results = append(results, f.northWest.GetRelevant(point)...)
		} else if point.X() > f.center.X() && point.Y() <= f.center.Y() {
			results = append(results, f.northEast.GetRelevant(point)...)
		} else if point.X() <= f.center.X() && point.Y() > f.center.Y() {
			results = append(results, f.southWest.GetRelevant(point)...)
		} else {
			results = append(results, f.southEast.GetRelevant(point)...)
		}
	}
	return results
}

type Foothold struct {
	id     uint32
	first  *point.Model
	second *point.Model
}

func (f Foothold) isWall() bool {
	return f.first.X() == f.second.X()
}

type BackgroundType struct {
	layerNumber    uint32
	backgroundType uint32
}

type XLimit struct {
	min uint32
	max uint32
}
