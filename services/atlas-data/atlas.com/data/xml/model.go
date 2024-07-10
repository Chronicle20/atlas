package xml

import (
	"encoding/xml"
	"errors"
	"strconv"
)

type Node struct {
	XMLName      xml.Name      `xml:"imgdir"`
	Name         string        `xml:"name,attr"`
	ChildNodes   []Node        `xml:"imgdir"`
	CanvasNodes  []CanvasNode  `xml:"canvas"`
	IntegerNodes []IntegerNode `xml:"int"`
	StringNodes  []StringNode  `xml:"string"`
	PointNodes   []PointNode   `xml:"vector"`
}

func (i *Node) ChildByName(name string) (*Node, error) {
	for _, c := range i.ChildNodes {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, errors.New("child not found")
}

func (i *Node) GetShort(name string, def uint16) uint16 {
	for _, c := range i.IntegerNodes {
		if c.Name == name {
			res, err := strconv.ParseUint(c.Value, 10, 16)
			if err != nil {
				return def
			}
			return uint16(res)
		}
	}
	return def
}

func (i *Node) GetBool(name string, def bool) bool {
	for _, c := range i.IntegerNodes {
		if c.Name == name {
			res, err := strconv.ParseUint(c.Value, 10, 16)
			if err != nil {
				return def
			}
			return res == 1
		}
	}
	return def
}

func (i *Node) GetString(name string, def string) string {
	for _, c := range i.StringNodes {
		if c.Name == name {
			return c.Value
		}
	}
	return def
}

func (i *Node) GetIntegerWithDefault(name string, def int32) int32 {
	for _, c := range i.IntegerNodes {
		if c.Name == name {
			res, err := strconv.ParseInt(c.Value, 10, 32)
			if err != nil {
				return def
			}
			return int32(res)
		}
	}
	return def
}

func (i *Node) GetFloatWithDefault(name string, def float64) float64 {
	for _, c := range i.IntegerNodes {
		if c.Name == name {
			res, err := strconv.ParseFloat(c.Value, 64)
			if err != nil {
				return def
			}
			return res
		}
	}
	return def
}

func (i *Node) GetPoint(name string, defX int32, defY int32) (int32, int32) {
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
