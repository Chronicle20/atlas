package position

import "sort"

// GridPitch returns the lattice spacing at step: dx narrows as
// areaX/(step+1); dy narrows from a full-height row (step 0) toward
// areaY/2 as the step count rises, per FR-4.7.
func GridPitch(t Tuning, step byte) (dx int16, dy int16) {
	dx = t.AreaX / int16(step+1)
	divisor := int16(1) << uint(step+1)
	dy = t.AreaY/2 + t.AreaY/divisor
	return dx, dy
}

// NextGridPosition walks the lattice for step within bounds -- starting at
// the inset origin (bounds.X+InitialX, bounds.Y+InitialY) and stepping by
// GridPitch(t, step) -- and returns the first candidate whose dx*dy
// footprint does not intersect any rect already in placed. Ground
// resolution for the whole step's lattice is batched into a single snap
// call (design 5.3), not one call per candidate. ErrMapFull is returned
// when every lattice slot is occupied.
func NextGridPosition(t Tuning, bounds Rect, step byte, placed []Rect, snap SnapFunc) (Point, error) {
	dx, dy := GridPitch(t, step)
	if dx <= 0 || dy <= 0 {
		return Point{}, ErrMapFull
	}

	var candidates []Point
	for y := bounds.Y + t.InitialY; y < bounds.Y+bounds.H; y += dy {
		for x := bounds.X + t.InitialX; x < bounds.X+bounds.W; x += dx {
			candidates = append(candidates, Point{X: x, CY: y})
		}
	}
	if len(candidates) == 0 {
		return Point{}, ErrMapFull
	}

	resolved, err := snap(candidates)
	if err != nil {
		return Point{}, err
	}

	for _, p := range resolved {
		r := Rect{X: p.X, Y: p.CY, W: dx, H: dy}
		if !overlapsAny(r, placed) {
			return p, nil
		}
	}
	return Point{}, ErrMapFull
}

// Reorganize recomputes positions for existing at step+1, packing entries
// in ascending script-id order so a given NPC set lands deterministically
// regardless of arrival order.
func Reorganize(t Tuning, bounds Rect, step byte, existing []Placement, snap SnapFunc) ([]Placement, error) {
	newStep := step + 1

	sorted := make([]Placement, len(existing))
	copy(sorted, existing)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ScriptId < sorted[j].ScriptId })

	dx, dy := GridPitch(t, newStep)

	result := make([]Placement, 0, len(sorted))
	var placed []Rect
	for _, p := range sorted {
		point, err := NextGridPosition(t, bounds, newStep, placed, snap)
		if err != nil {
			return nil, err
		}
		rect := Rect{X: point.X, Y: point.CY, W: dx, H: dy}
		placed = append(placed, rect)
		result = append(result, Placement{ScriptId: p.ScriptId, Rect: rect, Point: point, Step: newStep})
	}
	return result, nil
}

func overlapsAny(r Rect, placed []Rect) bool {
	for _, p := range placed {
		if rectsOverlap(r, p) {
			return true
		}
	}
	return false
}

func rectsOverlap(a, b Rect) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}
