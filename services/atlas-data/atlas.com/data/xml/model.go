package xml

import (
	"encoding/xml"
	"errors"
	"strconv"
	"strings"
)

type Node struct {
	XMLName      xml.Name      `xml:"imgdir"`
	Name         string        `xml:"name,attr"`
	ChildNodes   []Node        `xml:"imgdir"`
	CanvasNodes  []CanvasNode  `xml:"canvas"`
	IntegerNodes []IntegerNode `xml:"int"`
	StringNodes  []StringNode  `xml:"string"`
	PointNodes   []PointNode   `xml:"vector"`
	DoubleNodes  []DoubleNode  `xml:"double"`
}

func (n *Node) ChildByName(name string) (*Node, error) {
	for _, c := range n.ChildNodes {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, errors.New("child not found")
}

// resolve walks a slash-separated path (e.g. "banMap/0/field") through
// ChildNodes for each leading segment and returns the node holding the final
// segment together with that final segment's own name, so callers can look
// it up among the leaf node's typed children (IntegerNodes/StringNodes/...).
// A name with no "/" is returned unchanged (node=n, leaf=name), which keeps
// today's exact-match behavior byte-identical for every existing call site.
// An unresolvable intermediate segment returns (nil, "").
func (n *Node) resolve(path string) (*Node, string) {
	if !strings.Contains(path, "/") {
		return n, path
	}
	parts := strings.Split(path, "/")
	cur := n
	for _, seg := range parts[:len(parts)-1] {
		next := (*Node)(nil)
		for i := range cur.ChildNodes {
			if cur.ChildNodes[i].Name == seg {
				next = &cur.ChildNodes[i]
				break
			}
		}
		if next == nil {
			return nil, ""
		}
		cur = next
	}
	return cur, parts[len(parts)-1]
}

func (n *Node) GetShort(name string, def uint16) uint16 {
	node, leaf := n.resolve(name)
	if node == nil {
		return def
	}
	for _, c := range node.IntegerNodes {
		if c.Name == leaf {
			res, err := strconv.ParseUint(c.Value, 10, 16)
			if err != nil {
				return def
			}
			return uint16(res)
		}
	}
	for _, c := range node.StringNodes {
		if c.Name == leaf {
			res, err := strconv.ParseUint(c.Value, 10, 16)
			if err != nil {
				return def
			}
			return uint16(res)
		}
	}
	return def
}

func (n *Node) GetBool(name string, def bool) bool {
	node, leaf := n.resolve(name)
	if node == nil {
		return def
	}
	for _, c := range node.IntegerNodes {
		if c.Name == leaf {
			res, err := strconv.ParseUint(c.Value, 10, 16)
			if err != nil {
				return def
			}
			return res == 1
		}
	}
	for _, c := range node.StringNodes {
		if c.Name == leaf {
			res, err := strconv.ParseUint(c.Value, 10, 16)
			if err != nil {
				return def
			}
			return res == 1
		}
	}
	return def
}

func (n *Node) GetString(name string, def string) string {
	node, leaf := n.resolve(name)
	if node == nil {
		return def
	}
	for _, c := range node.StringNodes {
		if c.Name == leaf {
			return c.Value
		}
	}
	return def
}

func (n *Node) GetIntegerWithDefault(name string, def int32) int32 {
	node, leaf := n.resolve(name)
	if node == nil {
		return def
	}
	for _, c := range node.IntegerNodes {
		if c.Name == leaf {
			res, err := strconv.ParseInt(c.Value, 10, 32)
			if err != nil {
				return def
			}
			return int32(res)
		}
	}
	for _, c := range node.StringNodes {
		if c.Name == leaf {
			res, err := strconv.ParseInt(c.Value, 10, 32)
			if err != nil {
				return def
			}
			return int32(res)
		}
	}
	return def
}

func (n *Node) GetFloatWithDefault(name string, def float64) float64 {
	node, leaf := n.resolve(name)
	if node == nil {
		return def
	}
	for _, c := range node.IntegerNodes {
		if c.Name == leaf {
			res, err := strconv.ParseFloat(c.Value, 64)
			if err != nil {
				return def
			}
			return res
		}
	}
	for _, c := range node.StringNodes {
		if c.Name == leaf {
			res, err := strconv.ParseFloat(c.Value, 64)
			if err != nil {
				return def
			}
			return res
		}
	}
	return def
}

func (n *Node) GetDouble(name string, def float64) float64 {
	node, leaf := n.resolve(name)
	if node == nil {
		return def
	}
	for _, c := range node.DoubleNodes {
		if c.Name == leaf {
			// Replace comma with period for proper float parsing
			value := c.Value
			for i := 0; i < len(value); i++ {
				if value[i] == ',' {
					value = value[:i] + "." + value[i+1:]
					break
				}
			}
			res, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return def
			}
			return res
		}
	}
	return def
}

func (n *Node) GetPoint(name string, defX int32, defY int32) (int32, int32) {
	node, leaf := n.resolve(name)
	if node == nil {
		return defX, defY
	}
	for _, c := range node.PointNodes {
		if c.Name == leaf {
			x, err := strconv.ParseInt(c.X, 10, 32)
			if err != nil {
				return defX, defY
			}
			y, err := strconv.ParseInt(c.Y, 10, 32)
			if err != nil {
				return defX, defY
			}
			return int32(x), int32(y)
		}
	}
	return defX, defY
}

type IntegerNode struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type StringNode struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type PointNode struct {
	Name string `xml:"name,attr"`
	X    string `xml:"x,attr"`
	Y    string `xml:"y,attr"`
}

type DoubleNode struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type CanvasNode struct {
	Name         string        `xml:"name,attr"`
	Width        string        `xml:"width,attr"`
	Height       string        `xml:"height,attr"`
	IntegerNodes []IntegerNode `xml:"int"`
	PointNodes   []PointNode   `xml:"vector"`
}

func (i *CanvasNode) GetIntegerWithDefault(name string, def int32) int32 {
	for _, c := range i.IntegerNodes {
		if c.Name == name {
			res, err := strconv.ParseUint(c.Value, 10, 32)
			if err != nil {
				return def
			}
			return int32(res)
		}
	}
	return def
}

func (i *CanvasNode) GetPoint(name string, defX int32, defY int32) (int32, int32) {
	for _, c := range i.PointNodes {
		if c.Name == name {
			x, err := strconv.ParseInt(c.X, 10, 32)
			if err != nil {
				return defX, defY
			}
			y, err := strconv.ParseInt(c.Y, 10, 32)
			if err != nil {
				return defX, defY
			}
			return int32(x), int32(y)
		}
	}
	return defX, defY
}
