package xml

import (
	"atlas-wz-extractor/wz"
	"atlas-wz-extractor/wz/property"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// SerializeToDirectory serializes a parsed WZ file to HaRepacker-compatible XML files.
// Output structure mirrors the WZ directory tree:
//   {outputDir}/{wzName}.wz/{dirPath}/{imageName}.img.xml
func SerializeToDirectory(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return fmt.Errorf("WZ file [%s] has no root directory", f.Name())
	}

	wzDir := filepath.Join(outputDir, f.Name()+".wz")
	if err := os.MkdirAll(wzDir, 0755); err != nil {
		return fmt.Errorf("unable to create output directory [%s]: %w", wzDir, err)
	}

	return serializeDirectory(l, root, wzDir)
}

func serializeDirectory(l logrus.FieldLogger, dir *wz.Directory, outputPath string) error {
	// Serialize all images in this directory
	for _, img := range dir.Images() {
		if err := serializeImage(l, img, outputPath); err != nil {
			l.WithError(err).Warnf("Unable to serialize image [%s].", img.Name())
		}
	}

	// Recurse into sub-directories
	for _, sub := range dir.Directories() {
		subPath := filepath.Join(outputPath, sub.Name())
		if err := os.MkdirAll(subPath, 0755); err != nil {
			return fmt.Errorf("unable to create directory [%s]: %w", subPath, err)
		}
		if err := serializeDirectory(l, sub, subPath); err != nil {
			return err
		}
	}

	return nil
}

func serializeImage(l logrus.FieldLogger, img *wz.Image, outputPath string) error {
	xmlPath := filepath.Join(outputPath, img.Name()+".img.xml")

	f, err := os.Create(xmlPath)
	if err != nil {
		return fmt.Errorf("unable to create XML file [%s]: %w", xmlPath, err)
	}
	defer f.Close()

	// Write XML header
	if _, err := f.WriteString(xml.Header); err != nil {
		return err
	}

	// Root imgdir element for the image
	e := xml.NewEncoder(f)
	e.Indent("", "  ")

	root := xmlElement{
		XMLName: xml.Name{Local: "imgdir"},
		Name:    img.Name() + ".img",
	}
	root.Children = propertiesToElements(img.Properties())

	if err := e.Encode(root); err != nil {
		return fmt.Errorf("unable to encode XML for [%s]: %w", img.Name(), err)
	}

	return nil
}

// xmlElement represents a generic HaRepacker XML element.
type xmlElement struct {
	XMLName  xml.Name      `xml:""`
	Name     string        `xml:"name,attr"`
	Value    string        `xml:"value,attr,omitempty"`
	Width    string        `xml:"width,attr,omitempty"`
	Height   string        `xml:"height,attr,omitempty"`
	X        string        `xml:"x,attr,omitempty"`
	Y        string        `xml:"y,attr,omitempty"`
	Children []xmlElement  `xml:",any"`
}

func propertiesToElements(props []property.Property) []xmlElement {
	if len(props) == 0 {
		return nil
	}
	elements := make([]xmlElement, 0, len(props))
	for _, p := range props {
		elements = append(elements, propertyToElement(p))
	}
	return elements
}

func propertyToElement(p property.Property) xmlElement {
	switch v := p.(type) {
	case *property.NullProperty:
		return xmlElement{
			XMLName: xml.Name{Local: "null"},
			Name:    v.Name(),
		}

	case *property.ShortProperty:
		return xmlElement{
			XMLName: xml.Name{Local: "short"},
			Name:    v.Name(),
			Value:   fmt.Sprintf("%d", v.Value()),
		}

	case *property.IntProperty:
		return xmlElement{
			XMLName: xml.Name{Local: "int"},
			Name:    v.Name(),
			Value:   fmt.Sprintf("%d", v.Value()),
		}

	case *property.LongProperty:
		return xmlElement{
			XMLName: xml.Name{Local: "long"},
			Name:    v.Name(),
			Value:   fmt.Sprintf("%d", v.Value()),
		}

	case *property.FloatProperty:
		return xmlElement{
			XMLName: xml.Name{Local: "float"},
			Name:    v.Name(),
			Value:   formatFloat(float64(v.Value())),
		}

	case *property.DoubleProperty:
		return xmlElement{
			XMLName: xml.Name{Local: "double"},
			Name:    v.Name(),
			Value:   formatFloat(v.Value()),
		}

	case *property.StringProperty:
		return xmlElement{
			XMLName: xml.Name{Local: "string"},
			Name:    v.Name(),
			Value:   v.Value(),
		}

	case *property.SubProperty:
		return xmlElement{
			XMLName:  xml.Name{Local: "imgdir"},
			Name:     v.Name(),
			Children: propertiesToElements(v.Children()),
		}

	case *property.CanvasProperty:
		return xmlElement{
			XMLName:  xml.Name{Local: "canvas"},
			Name:     v.Name(),
			Width:    fmt.Sprintf("%d", v.Width()),
			Height:   fmt.Sprintf("%d", v.Height()),
			Children: propertiesToElements(v.Children()),
		}

	case *property.VectorProperty:
		return xmlElement{
			XMLName: xml.Name{Local: "vector"},
			Name:    v.Name(),
			X:       fmt.Sprintf("%d", v.X()),
			Y:       fmt.Sprintf("%d", v.Y()),
		}

	case *property.ConvexProperty:
		return xmlElement{
			XMLName:  xml.Name{Local: "extended"},
			Name:     v.Name(),
			Children: propertiesToElements(v.Children()),
		}

	case *property.SoundProperty:
		return xmlElement{
			XMLName: xml.Name{Local: "sound"},
			Name:    v.Name(),
		}

	case *property.UOLProperty:
		return xmlElement{
			XMLName: xml.Name{Local: "uol"},
			Name:    v.Name(),
			Value:   v.Value(),
		}

	default:
		return xmlElement{
			XMLName: xml.Name{Local: "null"},
			Name:    p.Name(),
		}
	}
}

// formatFloat formats a float value ensuring it always contains a decimal point.
// MapleLib uses this convention: "0" -> "0.0", "1.5" stays "1.5".
func formatFloat(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}
