package mapimage

import (
	"atlas-wz-extractor/wz/property"
	"strconv"
)

// findSub returns the named SubProperty child, or nil.
func findSub(props []property.Property, name string) *property.SubProperty {
	for _, p := range props {
		if sub, ok := p.(*property.SubProperty); ok && sub.Name() == name {
			return sub
		}
	}
	return nil
}

// findCanvas returns the named CanvasProperty child, or nil.
func findCanvas(props []property.Property, name string) *property.CanvasProperty {
	for _, p := range props {
		if cp, ok := p.(*property.CanvasProperty); ok && cp.Name() == name {
			return cp
		}
	}
	return nil
}

// findVector returns the named VectorProperty child, or nil.
func findVector(props []property.Property, name string) *property.VectorProperty {
	for _, p := range props {
		if vp, ok := p.(*property.VectorProperty); ok && vp.Name() == name {
			return vp
		}
	}
	return nil
}

// intVal returns the named int/short property value, or `def` if absent.
func intVal(props []property.Property, name string, def int) int {
	for _, p := range props {
		switch v := p.(type) {
		case *property.IntProperty:
			if v.Name() == name {
				return int(v.Value())
			}
		case *property.ShortProperty:
			if v.Name() == name {
				return int(v.Value())
			}
		}
	}
	return def
}

// stringVal returns the named string property value, or `def` if absent.
func stringVal(props []property.Property, name string, def string) string {
	for _, p := range props {
		if v, ok := p.(*property.StringProperty); ok && v.Name() == name {
			return v.Value()
		}
	}
	return def
}

// intStr returns the named property's value as a string whether it's stored as
// an int or string. Returns `def` when missing.
func intStr(props []property.Property, name, def string) string {
	for _, p := range props {
		switch v := p.(type) {
		case *property.IntProperty:
			if v.Name() == name {
				return strconv.Itoa(int(v.Value()))
			}
		case *property.StringProperty:
			if v.Name() == name {
				return v.Value()
			}
		}
	}
	return def
}

// childrenOf returns the children of a named SubProperty, or nil.
func childrenOf(props []property.Property, name string) []property.Property {
	sub := findSub(props, name)
	if sub == nil {
		return nil
	}
	return sub.Children()
}

// canvasOrigin returns the origin vector of a canvas, defaulting to (0,0).
func canvasOrigin(cp *property.CanvasProperty) (int, int) {
	v := findVector(cp.Children(), "origin")
	if v == nil {
		return 0, 0
	}
	return int(v.X()), int(v.Y())
}

// canvasZ returns the per-sprite z, or 0 if missing.
func canvasZ(cp *property.CanvasProperty) int {
	return intVal(cp.Children(), "z", 0)
}
