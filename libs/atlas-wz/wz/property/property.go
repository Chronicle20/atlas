package property

// Type represents the type of a WZ property.
type Type int

const (
	TypeNull Type = iota
	TypeShort
	TypeInt
	TypeLong
	TypeFloat
	TypeDouble
	TypeString
	TypeSub
	TypeCanvas
	TypeVector
	TypeConvex
	TypeSound
	TypeUOL
)

// Property is the common interface for all WZ property types.
type Property interface {
	Name() string
	Type() Type
	Children() []Property
}

// NullProperty represents a null/empty property.
type NullProperty struct {
	name string
}

func NewNull(name string) *NullProperty        { return &NullProperty{name: name} }
func (p *NullProperty) Name() string           { return p.name }
func (p *NullProperty) Type() Type             { return TypeNull }
func (p *NullProperty) Children() []Property   { return nil }

// ShortProperty represents an int16 property.
type ShortProperty struct {
	name  string
	value int16
}

func NewShort(name string, value int16) *ShortProperty { return &ShortProperty{name: name, value: value} }
func (p *ShortProperty) Name() string                  { return p.name }
func (p *ShortProperty) Type() Type                    { return TypeShort }
func (p *ShortProperty) Children() []Property          { return nil }
func (p *ShortProperty) Value() int16                  { return p.value }

// IntProperty represents an int32 property.
type IntProperty struct {
	name  string
	value int32
}

func NewInt(name string, value int32) *IntProperty { return &IntProperty{name: name, value: value} }
func (p *IntProperty) Name() string                { return p.name }
func (p *IntProperty) Type() Type                  { return TypeInt }
func (p *IntProperty) Children() []Property        { return nil }
func (p *IntProperty) Value() int32                { return p.value }

// LongProperty represents an int64 property.
type LongProperty struct {
	name  string
	value int64
}

func NewLong(name string, value int64) *LongProperty { return &LongProperty{name: name, value: value} }
func (p *LongProperty) Name() string                 { return p.name }
func (p *LongProperty) Type() Type                   { return TypeLong }
func (p *LongProperty) Children() []Property         { return nil }
func (p *LongProperty) Value() int64                 { return p.value }

// FloatProperty represents a float32 property.
type FloatProperty struct {
	name  string
	value float32
}

func NewFloat(name string, value float32) *FloatProperty {
	return &FloatProperty{name: name, value: value}
}
func (p *FloatProperty) Name() string           { return p.name }
func (p *FloatProperty) Type() Type             { return TypeFloat }
func (p *FloatProperty) Children() []Property   { return nil }
func (p *FloatProperty) Value() float32         { return p.value }

// DoubleProperty represents a float64 property.
type DoubleProperty struct {
	name  string
	value float64
}

func NewDouble(name string, value float64) *DoubleProperty {
	return &DoubleProperty{name: name, value: value}
}
func (p *DoubleProperty) Name() string           { return p.name }
func (p *DoubleProperty) Type() Type             { return TypeDouble }
func (p *DoubleProperty) Children() []Property   { return nil }
func (p *DoubleProperty) Value() float64         { return p.value }

// StringProperty represents a string property.
type StringProperty struct {
	name  string
	value string
}

func NewString(name, value string) *StringProperty { return &StringProperty{name: name, value: value} }
func (p *StringProperty) Name() string             { return p.name }
func (p *StringProperty) Type() Type               { return TypeString }
func (p *StringProperty) Children() []Property     { return nil }
func (p *StringProperty) Value() string            { return p.value }

// SubProperty represents a nested property container (imgdir).
type SubProperty struct {
	name     string
	children []Property
}

func NewSub(name string, children []Property) *SubProperty {
	return &SubProperty{name: name, children: children}
}
func (p *SubProperty) Name() string           { return p.name }
func (p *SubProperty) Type() Type             { return TypeSub }
func (p *SubProperty) Children() []Property   { return p.children }

// CanvasProperty represents an image/canvas property.
type CanvasProperty struct {
	name       string
	width      int
	height     int
	format     int
	dataOffset int64
	dataSize   int32
	children   []Property
}

func NewCanvas(name string, width, height, format int, dataOffset int64, dataSize int32, children []Property) *CanvasProperty {
	return &CanvasProperty{
		name: name, width: width, height: height,
		format: format, dataOffset: dataOffset, dataSize: dataSize,
		children: children,
	}
}
func (p *CanvasProperty) Name() string           { return p.name }
func (p *CanvasProperty) Type() Type             { return TypeCanvas }
func (p *CanvasProperty) Children() []Property   { return p.children }
func (p *CanvasProperty) Width() int             { return p.width }
func (p *CanvasProperty) Height() int            { return p.height }
func (p *CanvasProperty) Format() int            { return p.format }
func (p *CanvasProperty) DataOffset() int64      { return p.dataOffset }
func (p *CanvasProperty) DataSize() int32        { return p.dataSize }

// VectorProperty represents a 2D vector (x, y).
type VectorProperty struct {
	name string
	x, y int32
}

func NewVector(name string, x, y int32) *VectorProperty {
	return &VectorProperty{name: name, x: x, y: y}
}
func (p *VectorProperty) Name() string           { return p.name }
func (p *VectorProperty) Type() Type             { return TypeVector }
func (p *VectorProperty) Children() []Property   { return nil }
func (p *VectorProperty) X() int32               { return p.x }
func (p *VectorProperty) Y() int32               { return p.y }

// ConvexProperty represents a convex shape (list of child properties).
type ConvexProperty struct {
	name     string
	children []Property
}

func NewConvex(name string, children []Property) *ConvexProperty {
	return &ConvexProperty{name: name, children: children}
}
func (p *ConvexProperty) Name() string           { return p.name }
func (p *ConvexProperty) Type() Type             { return TypeConvex }
func (p *ConvexProperty) Children() []Property   { return p.children }

// SoundProperty represents a sound property (stub).
type SoundProperty struct {
	name string
}

func NewSound(name string) *SoundProperty      { return &SoundProperty{name: name} }
func (p *SoundProperty) Name() string          { return p.name }
func (p *SoundProperty) Type() Type            { return TypeSound }
func (p *SoundProperty) Children() []Property  { return nil }

// UOLProperty represents a UOL (symbolic link) property.
type UOLProperty struct {
	name  string
	value string
}

func NewUOL(name, value string) *UOLProperty   { return &UOLProperty{name: name, value: value} }
func (p *UOLProperty) Name() string            { return p.name }
func (p *UOLProperty) Type() Type              { return TypeUOL }
func (p *UOLProperty) Children() []Property    { return nil }
func (p *UOLProperty) Value() string           { return p.value }
