package mapimage

import (
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

type backEntry struct {
	idx    int
	bS     string
	no     int
	x, y   int
	cx, cy int
	typ    int
	a      int
	front  int
	f      int
	ani    int
}

type tileEntry struct {
	idx  int
	tS   string
	u    string
	no   string
	x, y int
	zM   int
}

type objEntry struct {
	idx int
	oS  string
	l0  string
	l1  string
	l2  string
	x   int
	y   int
	z   int
	zM  int
	f   int
}

func loadBackEntries(root []property.Property) []backEntry {
	back := findSub(root, "back")
	if back == nil {
		return nil
	}
	var out []backEntry
	for i, c := range back.Children() {
		sub, ok := c.(*property.SubProperty)
		if !ok {
			continue
		}
		ch := sub.Children()
		out = append(out, backEntry{
			idx:   i,
			bS:    stringVal(ch, "bS", ""),
			no:    intVal(ch, "no", 0),
			x:     intVal(ch, "x", 0),
			y:     intVal(ch, "y", 0),
			cx:    intVal(ch, "cx", 0),
			cy:    intVal(ch, "cy", 0),
			typ:   intVal(ch, "type", 0),
			a:     intVal(ch, "a", 255),
			front: intVal(ch, "front", 0),
			f:     intVal(ch, "f", 0),
			ani:   intVal(ch, "ani", 0),
		})
	}
	return out
}

func loadTileEntries(layerProps []property.Property, tS string) []tileEntry {
	tile := findSub(layerProps, "tile")
	if tile == nil {
		return nil
	}
	var out []tileEntry
	for i, c := range tile.Children() {
		sub, ok := c.(*property.SubProperty)
		if !ok {
			continue
		}
		ch := sub.Children()
		out = append(out, tileEntry{
			idx: i,
			tS:  tS,
			u:   stringVal(ch, "u", ""),
			no:  intStr(ch, "no", "0"),
			x:   intVal(ch, "x", 0),
			y:   intVal(ch, "y", 0),
			zM:  intVal(ch, "zM", 0),
		})
	}
	return out
}

func loadObjEntries(layerProps []property.Property) []objEntry {
	obj := findSub(layerProps, "obj")
	if obj == nil {
		return nil
	}
	var out []objEntry
	for i, c := range obj.Children() {
		sub, ok := c.(*property.SubProperty)
		if !ok {
			continue
		}
		ch := sub.Children()
		out = append(out, objEntry{
			idx: i,
			oS:  stringVal(ch, "oS", ""),
			l0:  stringVal(ch, "l0", ""),
			l1:  stringVal(ch, "l1", ""),
			l2:  stringVal(ch, "l2", ""),
			x:   intVal(ch, "x", 0),
			y:   intVal(ch, "y", 0),
			z:   intVal(ch, "z", 0),
			zM:  intVal(ch, "zM", 0),
			f:   intVal(ch, "f", 0),
		})
	}
	return out
}
