// Package wzxml maps a parsed WZ property tree to HaRepacker-compatible XML
// elements. This mapping is the shared, wire-compatible contract between the
// atlas-data ingest pipeline (which serializes WZ trees to `.img.xml` files
// on disk) and wzdiff (which compares two WZ trees through the same
// serialization the ingest pipeline uses). It has no dependency on any
// service module.
package wzxml

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// Element is a generic HaRepacker XML element.
type Element struct {
	XMLName  xml.Name  `xml:""`
	Name     string    `xml:"name,attr"`
	Value    string    `xml:"value,attr,omitempty"`
	Width    string    `xml:"width,attr,omitempty"`
	Height   string    `xml:"height,attr,omitempty"`
	X        string    `xml:"x,attr,omitempty"`
	Y        string    `xml:"y,attr,omitempty"`
	Children []Element `xml:",any"`
}

// PropertiesToElements converts a slice of properties to their XML element
// representations, preserving order.
func PropertiesToElements(props []property.Property) []Element {
	if len(props) == 0 {
		return nil
	}
	out := make([]Element, 0, len(props))
	for _, p := range props {
		out = append(out, PropertyToElement(p))
	}
	return out
}

// PropertyToElement converts a single property to its XML element
// representation.
func PropertyToElement(p property.Property) Element {
	switch v := p.(type) {
	case *property.NullProperty:
		return Element{XMLName: xml.Name{Local: "null"}, Name: v.Name()}
	case *property.ShortProperty:
		return Element{XMLName: xml.Name{Local: "short"}, Name: v.Name(), Value: fmt.Sprintf("%d", v.Value())}
	case *property.IntProperty:
		return Element{XMLName: xml.Name{Local: "int"}, Name: v.Name(), Value: fmt.Sprintf("%d", v.Value())}
	case *property.LongProperty:
		return Element{XMLName: xml.Name{Local: "long"}, Name: v.Name(), Value: fmt.Sprintf("%d", v.Value())}
	case *property.FloatProperty:
		return Element{XMLName: xml.Name{Local: "float"}, Name: v.Name(), Value: FormatFloat(float64(v.Value()))}
	case *property.DoubleProperty:
		return Element{XMLName: xml.Name{Local: "double"}, Name: v.Name(), Value: FormatFloat(v.Value())}
	case *property.StringProperty:
		return Element{XMLName: xml.Name{Local: "string"}, Name: v.Name(), Value: v.Value()}
	case *property.SubProperty:
		return Element{XMLName: xml.Name{Local: "imgdir"}, Name: v.Name(), Children: PropertiesToElements(v.Children())}
	case *property.CanvasProperty:
		return Element{
			XMLName:  xml.Name{Local: "canvas"},
			Name:     v.Name(),
			Width:    fmt.Sprintf("%d", v.Width()),
			Height:   fmt.Sprintf("%d", v.Height()),
			Children: PropertiesToElements(v.Children()),
		}
	case *property.VectorProperty:
		return Element{
			XMLName: xml.Name{Local: "vector"},
			Name:    v.Name(),
			X:       fmt.Sprintf("%d", v.X()),
			Y:       fmt.Sprintf("%d", v.Y()),
		}
	case *property.ConvexProperty:
		return Element{XMLName: xml.Name{Local: "extended"}, Name: v.Name(), Children: PropertiesToElements(v.Children())}
	case *property.SoundProperty:
		return Element{XMLName: xml.Name{Local: "sound"}, Name: v.Name()}
	case *property.UOLProperty:
		return Element{XMLName: xml.Name{Local: "uol"}, Name: v.Name(), Value: v.Value()}
	default:
		return Element{XMLName: xml.Name{Local: "null"}, Name: p.Name()}
	}
}

// FormatFloat formats a float ensuring it always contains a decimal point.
// MapleLib uses "0" -> "0.0", "1.5" stays "1.5".
func FormatFloat(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}
