package wzdiff

import (
	"encoding/xml"
	"fmt"
	"os"
)

// LoadImageXML reads a HaRepacker-style ".img.xml" dump and returns the
// image's top-level property nodes in document order. The outer <imgdir
// name="...img"> wrapper is stripped: it names the image file, not a
// property, and contributes nothing to the path notation.
//
// It reads the file through encoding/xml's token stream rather than struct
// unmarshalling so unknown attributes and element names are preserved into
// Attrs/Kind rather than silently dropped: a dropped attribute here would
// make the differ lie about what the dump actually contains.
func LoadImageXML(path string) ([]Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	root, err := decodeElement(dec, nil)
	if err != nil {
		return nil, fmt.Errorf("wzdiff: decode %s: %w", path, err)
	}
	if root == nil {
		return nil, fmt.Errorf("wzdiff: %s: no root element", path)
	}
	return root.Children, nil
}

// decodeElement decodes one XML element and its descendants into a Node.
// If start is non-nil it is the StartElement already consumed by the
// caller (used when recursing into a child); otherwise decodeElement scans
// forward past any leading tokens (the XML declaration, whitespace) to
// find the next StartElement.
func decodeElement(dec *xml.Decoder, start *xml.StartElement) (*Node, error) {
	var tok xml.StartElement
	if start != nil {
		tok = *start
	} else {
		for {
			t, err := dec.Token()
			if err != nil {
				return nil, err
			}
			if se, ok := t.(xml.StartElement); ok {
				tok = se
				break
			}
		}
	}

	n := &Node{Kind: tok.Name.Local, Attrs: map[string]string{}}
	for _, a := range tok.Attr {
		if a.Name.Local == "name" {
			n.Name = a.Value
			continue
		}
		n.Attrs[a.Name.Local] = a.Value
	}
	if len(n.Attrs) == 0 {
		n.Attrs = nil
	}

	for {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch tt := t.(type) {
		case xml.StartElement:
			child, err := decodeElement(dec, &tt)
			if err != nil {
				return nil, err
			}
			n.Children = append(n.Children, *child)
		case xml.EndElement:
			return n, nil
		}
	}
}
