// Package wztoxml converts an in-memory WZ tree to HaRepacker-compatible XML
// files on disk. Output mirrors what atlas-wz-extractor's deleted
// xml/serializer.go used to produce, so the existing atlas-data domain readers
// (which still consume `.img.xml` files via xml.FromPathProvider) work
// unchanged.
//
// Layout (rooted at outputDir, mirroring the WZ directory tree):
//
//	{outputDir}/{wzName}.wz/{dirPath}/{imageName}.img.xml
package wztoxml

import (
	stdxml "encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/sirupsen/logrus"
)

// SerializeToDirectory serializes a parsed WZ file to HaRepacker-compatible XML
// files. Output layout:
//
//	{outputDir}/{wzName}.wz/{dirPath}/{imageName}.img.xml
func SerializeToDirectory(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return fmt.Errorf("wz file [%s] has no root directory", f.Name())
	}
	wzDir := filepath.Join(outputDir, f.Name()+".wz")
	if err := os.MkdirAll(wzDir, 0o755); err != nil {
		return fmt.Errorf("create output directory [%s]: %w", wzDir, err)
	}
	return serializeDirectory(l, root, wzDir)
}

func serializeDirectory(l logrus.FieldLogger, dir *wz.Directory, outputPath string) error {
	for _, img := range dir.Images() {
		if err := SerializeImage(img, outputPath); err != nil {
			l.WithError(err).Warnf("unable to serialize image [%s]", img.Name())
		}
	}
	for _, sub := range dir.Directories() {
		subPath := filepath.Join(outputPath, sub.Name())
		if err := os.MkdirAll(subPath, 0o755); err != nil {
			return fmt.Errorf("create directory [%s]: %w", subPath, err)
		}
		if err := serializeDirectory(l, sub, subPath); err != nil {
			return err
		}
	}
	return nil
}

// SerializeImage writes a single WZ image to {outputPath}/{imageName}.img.xml.
func SerializeImage(img *wz.Image, outputPath string) error {
	xmlPath := filepath.Join(outputPath, img.Name()+".img.xml")
	f, err := os.Create(xmlPath)
	if err != nil {
		return fmt.Errorf("create xml file [%s]: %w", xmlPath, err)
	}
	defer f.Close()
	if _, err := f.WriteString(stdxml.Header); err != nil {
		return err
	}
	e := stdxml.NewEncoder(f)
	e.Indent("", "  ")
	root := xmlElement{
		XMLName: stdxml.Name{Local: "imgdir"},
		Name:    img.Name() + ".img",
	}
	props, err := img.Properties()
	if err != nil {
		return fmt.Errorf("wztoxml adapter: %s: %w", img.Name(), err)
	}
	root.Children = propertiesToElements(props)
	if err := e.Encode(root); err != nil {
		return fmt.Errorf("encode xml for [%s]: %w", img.Name(), err)
	}
	return nil
}

// xmlElement is a generic HaRepacker XML element.
type xmlElement struct {
	XMLName  stdxml.Name  `xml:""`
	Name     string       `xml:"name,attr"`
	Value    string       `xml:"value,attr,omitempty"`
	Width    string       `xml:"width,attr,omitempty"`
	Height   string       `xml:"height,attr,omitempty"`
	X        string       `xml:"x,attr,omitempty"`
	Y        string       `xml:"y,attr,omitempty"`
	Children []xmlElement `xml:",any"`
}

func propertiesToElements(props []property.Property) []xmlElement {
	if len(props) == 0 {
		return nil
	}
	out := make([]xmlElement, 0, len(props))
	for _, p := range props {
		out = append(out, propertyToElement(p))
	}
	return out
}

func propertyToElement(p property.Property) xmlElement {
	switch v := p.(type) {
	case *property.NullProperty:
		return xmlElement{XMLName: stdxml.Name{Local: "null"}, Name: v.Name()}
	case *property.ShortProperty:
		return xmlElement{XMLName: stdxml.Name{Local: "short"}, Name: v.Name(), Value: fmt.Sprintf("%d", v.Value())}
	case *property.IntProperty:
		return xmlElement{XMLName: stdxml.Name{Local: "int"}, Name: v.Name(), Value: fmt.Sprintf("%d", v.Value())}
	case *property.LongProperty:
		return xmlElement{XMLName: stdxml.Name{Local: "long"}, Name: v.Name(), Value: fmt.Sprintf("%d", v.Value())}
	case *property.FloatProperty:
		return xmlElement{XMLName: stdxml.Name{Local: "float"}, Name: v.Name(), Value: formatFloat(float64(v.Value()))}
	case *property.DoubleProperty:
		return xmlElement{XMLName: stdxml.Name{Local: "double"}, Name: v.Name(), Value: formatFloat(v.Value())}
	case *property.StringProperty:
		return xmlElement{XMLName: stdxml.Name{Local: "string"}, Name: v.Name(), Value: v.Value()}
	case *property.SubProperty:
		return xmlElement{XMLName: stdxml.Name{Local: "imgdir"}, Name: v.Name(), Children: propertiesToElements(v.Children())}
	case *property.CanvasProperty:
		return xmlElement{
			XMLName:  stdxml.Name{Local: "canvas"},
			Name:     v.Name(),
			Width:    fmt.Sprintf("%d", v.Width()),
			Height:   fmt.Sprintf("%d", v.Height()),
			Children: propertiesToElements(v.Children()),
		}
	case *property.VectorProperty:
		return xmlElement{
			XMLName: stdxml.Name{Local: "vector"},
			Name:    v.Name(),
			X:       fmt.Sprintf("%d", v.X()),
			Y:       fmt.Sprintf("%d", v.Y()),
		}
	case *property.ConvexProperty:
		return xmlElement{XMLName: stdxml.Name{Local: "extended"}, Name: v.Name(), Children: propertiesToElements(v.Children())}
	case *property.SoundProperty:
		return xmlElement{XMLName: stdxml.Name{Local: "sound"}, Name: v.Name()}
	case *property.UOLProperty:
		return xmlElement{XMLName: stdxml.Name{Local: "uol"}, Name: v.Name(), Value: v.Value()}
	default:
		return xmlElement{XMLName: stdxml.Name{Local: "null"}, Name: p.Name()}
	}
}

// formatFloat formats a float ensuring it always contains a decimal point.
// MapleLib uses "0" -> "0.0", "1.5" stays "1.5".
func formatFloat(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}
