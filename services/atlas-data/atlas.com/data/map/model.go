package _map

import (
	"atlas-data/point"
	"math"
	"sort"
)

type TimeMobRestModel struct {
	Id      uint32 `json:"id"`
	Message string `json:"message"`
}

func (r RectangleRestModel) contains(ret point.RestModel) bool {
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

func (f *FootholdTreeRestModel) findBelow(initial point.RestModel) *FootholdRestModel {
	relevants := f.GetRelevant(initial)
	matches := make([]FootholdRestModel, 0)

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

func (f *FootholdTreeRestModel) GetRelevant(point point.RestModel) []FootholdRestModel {
	results := make([]FootholdRestModel, 0)
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

func (f FootholdRestModel) isWall() bool {
	return f.First.X == f.Second.X
}

type XLimitRestModel struct {
	Min uint32 `json:"min"`
	Max uint32 `json:"max"`
}
