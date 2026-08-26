// Package wztest builds tiny, well-formed PKG1 (.wz) archives in memory for
// tests. It exists so both libs/atlas-wz's own tests and atlas-data's worker
// tests can construct fixtures without committing real game archives to the
// repo. TEST FIXTURES ONLY — never use this package in production code.
package wztest

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
)

// Kind discriminates Prop variants.
type Kind int

const (
	KindInt Kind = iota
	KindString
	KindSub
	KindCanvas
	KindUOL
	KindNull
	KindShort
	KindLong
	KindFloat
	KindDouble
	KindVector
	KindConvex
)

// Prop is one property inside an image.
type Prop struct {
	Name     string
	Kind     Kind
	Int      int32
	Str      string // also carries the UOL target string for KindUOL
	Canvas   []byte // raw payload; the builder prepends the 1-byte flag header
	Children []Prop // also carries the Convex children for KindConvex
	Short    int16
	Long     int64
	Float    float32
	Double   float64
	X, Y     int32 // Vector coordinates
}

func Int(name string, v int32) Prop { return Prop{Name: name, Kind: KindInt, Int: v} }
func Str(name, v string) Prop       { return Prop{Name: name, Kind: KindString, Str: v} }
func Sub(name string, children ...Prop) Prop {
	return Prop{Name: name, Kind: KindSub, Children: children}
}

func Canvas(name string, payload []byte) Prop {
	return Prop{Name: name, Kind: KindCanvas, Canvas: payload}
}

// UOL builds an extended-type "UOL" property (a symlink-like reference to
// another property path).
func UOL(name, target string) Prop {
	return Prop{Name: name, Kind: KindUOL, Str: target}
}

// Null builds a type-0 property with no payload.
func Null(name string) Prop { return Prop{Name: name, Kind: KindNull} }

// Short builds a type-2 int16 property.
func Short(name string, v int16) Prop { return Prop{Name: name, Kind: KindShort, Short: v} }

// Long builds a type-20 WzLong property.
func Long(name string, v int64) Prop { return Prop{Name: name, Kind: KindLong, Long: v} }

// Float builds a type-4 float32 property, always with the 0x80 marker byte
// present so the value round-trips (a missing/other marker byte reads as 0).
func Float(name string, v float32) Prop { return Prop{Name: name, Kind: KindFloat, Float: v} }

// Double builds a type-5 float64 property.
func Double(name string, v float64) Prop { return Prop{Name: name, Kind: KindDouble, Double: v} }

// Vector builds an extended-type "Shape2D#Vector2D" property.
func Vector(name string, x, y int32) Prop { return Prop{Name: name, Kind: KindVector, X: x, Y: y} }

// Convex builds an extended-type "Shape2D#Convex2D" property whose children
// are written bare (no property-name string block, no length prefix), since
// parseExtendedProperty is re-entered directly for each child.
func Convex(name string, children ...Prop) Prop {
	return Prop{Name: name, Kind: KindConvex, Children: children}
}

// Image is one .img entry. Enc overrides the file-level encryption for this
// image's contents (the mixed-encryption JMS case); nil means file encryption.
type Image struct {
	Name  string
	Props []Prop
	Enc   *crypto.EncryptionType
}

func Img(name string, props ...Prop) Image { return Image{Name: name, Props: props} }

func ImgWithKey(name string, enc crypto.EncryptionType, props ...Prop) Image {
	e := enc
	return Image{Name: name, Props: props, Enc: &e}
}

// Dir is a directory node.
type Dir struct {
	Name   string
	Dirs   []Dir
	Images []Image
}

// Builder assembles a PKG1 archive. Zero value is not usable; use NewBuilder.
type Builder struct {
	version      int
	enc          crypto.EncryptionType
	root         Dir
	rawFirstName []byte
}

func NewBuilder() *Builder {
	return &Builder{version: 83, enc: crypto.EncryptionNone}
}

func (b *Builder) SetVersion(v int) *Builder                        { b.version = v; return b }
func (b *Builder) SetEncryption(enc crypto.EncryptionType) *Builder { b.enc = enc; return b }

// SetRawRootEntryName writes raw verbatim as the on-disk name bytes of the
// FIRST root entry (no mask/key encoding). Used to construct archives whose
// entry names decode to garbage under every known key.
func (b *Builder) SetRawRootEntryName(raw []byte) *Builder { b.rawFirstName = raw; return b }

func (b *Builder) AddDir(d Dir) *Builder     { b.root.Dirs = append(b.root.Dirs, d); return b }
func (b *Builder) AddImage(i Image) *Builder { b.root.Images = append(b.root.Images, i); return b }

// ---- binary encoding ----

type chunk struct {
	data    []byte
	patches []patch
}

// patch marks a 4-byte encrypted-offset field at pos (within the chunk)
// pointing at the chunk with index target.
type patch struct {
	pos    int
	target int
}

func keyBytes(enc crypto.EncryptionType) []byte {
	return crypto.GetKeyForRegion(enc).Bytes(0x10000)
}

func writeWzInt(buf *bytes.Buffer, v int32) {
	if v >= -127 && v <= 127 {
		buf.WriteByte(byte(int8(v)))
		return
	}
	buf.WriteByte(0x80)
	_ = binary.Write(buf, binary.LittleEndian, v)
}

// writeWzLong mirrors Reader.ReadWzLong: an in-range value is a single
// signed byte, otherwise byte 0x80 (int8 -128) then a full int64.
func writeWzLong(buf *bytes.Buffer, v int64) {
	if v >= -127 && v <= 127 {
		buf.WriteByte(byte(int8(v)))
		return
	}
	buf.WriteByte(0x80)
	_ = binary.Write(buf, binary.LittleEndian, v)
}

// writeWzString emits an ASCII WZ string: int8(-len) tag, then each byte
// XOR'd with the incrementing 0xAA mask and the key. Mirrors
// Reader.readWzASCIIStringInline exactly.
func writeWzString(buf *bytes.Buffer, s string, key []byte) error {
	if len(s) == 0 {
		buf.WriteByte(0)
		return nil
	}
	if len(s) > 127 {
		return fmt.Errorf("wztest: string %q longer than 127 bytes not supported", s)
	}
	buf.WriteByte(byte(int8(-len(s))))
	mask := byte(0xAA)
	for i := 0; i < len(s); i++ {
		c := s[i] ^ mask
		if i < len(key) {
			c ^= key[i]
		}
		buf.WriteByte(c)
		mask++
	}
	return nil
}

// writeStringBlock emits the 0x73 inline-string block form read by
// Reader.ReadWzStringBlock.
func writeStringBlock(buf *bytes.Buffer, s string, key []byte) error {
	buf.WriteByte(0x73)
	return writeWzString(buf, s, key)
}

func writePropList(buf *bytes.Buffer, props []Prop, key []byte) error {
	writeWzInt(buf, int32(len(props)))
	for _, p := range props {
		if err := writeStringBlock(buf, p.Name, key); err != nil {
			return err
		}
		switch p.Kind {
		case KindNull:
			buf.WriteByte(0)
		case KindShort:
			buf.WriteByte(2)
			_ = binary.Write(buf, binary.LittleEndian, p.Short)
		case KindInt:
			buf.WriteByte(3)
			writeWzInt(buf, p.Int)
		case KindLong:
			buf.WriteByte(20)
			writeWzLong(buf, p.Long)
		case KindFloat:
			buf.WriteByte(4)
			buf.WriteByte(0x80) // marker byte; any other value reads back as 0
			_ = binary.Write(buf, binary.LittleEndian, p.Float)
		case KindDouble:
			buf.WriteByte(5)
			_ = binary.Write(buf, binary.LittleEndian, p.Double)
		case KindString:
			buf.WriteByte(8)
			if err := writeStringBlock(buf, p.Str, key); err != nil {
				return err
			}
		case KindSub, KindCanvas, KindUOL, KindVector, KindConvex:
			var inner bytes.Buffer
			if err := writeExtendedContent(&inner, p, key); err != nil {
				return err
			}
			buf.WriteByte(9)
			_ = binary.Write(buf, binary.LittleEndian, int32(inner.Len()))
			buf.Write(inner.Bytes())
		default:
			return fmt.Errorf("wztest: unknown prop kind %d", p.Kind)
		}
	}
	return nil
}

// writeExtendedContent writes the tag-plus-payload body of a type-9 extended
// property, without the outer type byte and int32 length prefix. Used both
// for the wrapped top-level case (writePropList) and for Convex children,
// which parseExtendedProperty re-enters bare with no wrapping at all
// (image.go: parseExtendedProperty is called directly, never through
// parsePropertyValue's type-9 case).
func writeExtendedContent(buf *bytes.Buffer, p Prop, key []byte) error {
	switch p.Kind {
	case KindSub:
		if err := writeStringBlock(buf, "Property", key); err != nil {
			return err
		}
		buf.Write([]byte{0, 0})
		return writePropList(buf, p.Children, key)
	case KindCanvas:
		if err := writeStringBlock(buf, "Canvas", key); err != nil {
			return err
		}
		buf.WriteByte(0)   // skipped byte
		buf.WriteByte(0)   // hasProperty = 0
		writeWzInt(buf, 1) // width
		writeWzInt(buf, 1) // height
		writeWzInt(buf, 2) // format
		buf.WriteByte(0)   // format2
		buf.Write([]byte{0, 0, 0, 0})
		_ = binary.Write(buf, binary.LittleEndian, int32(len(p.Canvas)+1))
		buf.WriteByte(0xAB) // flag byte skipped by ReadCanvasData
		buf.Write(p.Canvas)
		return nil
	case KindUOL:
		if err := writeStringBlock(buf, "UOL", key); err != nil {
			return err
		}
		buf.WriteByte(0) // skipped byte
		return writeStringBlock(buf, p.Str, key)
	case KindVector:
		if err := writeStringBlock(buf, "Shape2D#Vector2D", key); err != nil {
			return err
		}
		writeWzInt(buf, p.X)
		writeWzInt(buf, p.Y)
		return nil
	case KindConvex:
		if err := writeStringBlock(buf, "Shape2D#Convex2D", key); err != nil {
			return err
		}
		writeWzInt(buf, int32(len(p.Children)))
		for _, c := range p.Children {
			if err := writeExtendedContent(buf, c, key); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("wztest: kind %d is not an extended property type", p.Kind)
	}
}

// buildImage serializes one image block with its effective key.
func (b *Builder) buildImage(img Image) (chunk, error) {
	key := keyBytes(b.enc)
	if img.Enc != nil {
		key = keyBytes(*img.Enc)
	}
	var buf bytes.Buffer
	if err := writeStringBlock(&buf, "Property", key); err != nil {
		return chunk{}, err
	}
	buf.Write([]byte{0, 0})
	if err := writePropList(&buf, img.Props, key); err != nil {
		return chunk{}, err
	}
	return chunk{data: buf.Bytes()}, nil
}

// buildDir serializes one directory chunk. Children chunks must already be
// in chunks (post-order) so their sizes are known; offsets are patched later.
// isRoot enables the rawFirstName override for the first entry.
func (b *Builder) buildDir(d Dir, chunks *[]chunk, isRoot bool) (int, error) {
	fileKey := keyBytes(b.enc)

	type entry struct {
		typ    byte
		name   string
		target int
		size   int
	}
	var entries []entry
	for _, sd := range d.Dirs {
		idx, err := b.buildDir(sd, chunks, false)
		if err != nil {
			return 0, err
		}
		entries = append(entries, entry{typ: 3, name: sd.Name, target: idx, size: len((*chunks)[idx].data)})
	}
	for _, img := range d.Images {
		c, err := b.buildImage(img)
		if err != nil {
			return 0, err
		}
		*chunks = append(*chunks, c)
		idx := len(*chunks) - 1
		entries = append(entries, entry{typ: 4, name: img.Name + ".img", target: idx, size: len(c.data)})
	}

	var buf bytes.Buffer
	var patches []patch
	writeWzInt(&buf, int32(len(entries)))
	for i, e := range entries {
		buf.WriteByte(e.typ)
		if isRoot && i == 0 && b.rawFirstName != nil {
			if len(b.rawFirstName) == 0 || len(b.rawFirstName) > 127 {
				return 0, fmt.Errorf("wztest: raw name must be 1..127 bytes")
			}
			buf.WriteByte(byte(int8(-len(b.rawFirstName))))
			buf.Write(b.rawFirstName)
		} else if err := writeWzString(&buf, e.name, fileKey); err != nil {
			return 0, err
		}
		writeWzInt(&buf, int32(e.size))
		writeWzInt(&buf, 0) // checksum
		patches = append(patches, patch{pos: buf.Len(), target: e.target})
		buf.Write([]byte{0, 0, 0, 0}) // encrypted offset, patched in Build
	}
	*chunks = append(*chunks, chunk{data: buf.Bytes(), patches: patches})
	return len(*chunks) - 1, nil
}

func rotl32(v uint32, count byte) uint32 {
	n := uint(count) % 32
	return (v << n) | (v >> (32 - n))
}

// Build assembles the archive: header, u16 encrypted version, root directory,
// then all sub-directory and image chunks, with directory-entry offsets
// encrypted exactly the way Reader.ReadWzOffset decrypts them.
func (b *Builder) Build() ([]byte, error) {
	ev, hash := crypto.CalculateVersionHash(b.version)

	desc := "Package file test"
	contentStart := 4 + 8 + 4 + len(desc) + 1

	var chunks []chunk
	rootIdx, err := b.buildDir(b.root, &chunks, true)
	if err != nil {
		return nil, err
	}

	// Layout: [header][u16 ev][root chunk][all other chunks in index order].
	pos := make([]int, len(chunks))
	cursor := contentStart + 2
	pos[rootIdx] = cursor
	cursor += len(chunks[rootIdx].data)
	for i := range chunks {
		if i == rootIdx {
			continue
		}
		pos[i] = cursor
		cursor += len(chunks[i].data)
	}
	total := cursor

	out := make([]byte, 0, total)
	var hdr bytes.Buffer
	hdr.WriteString("PKG1")
	_ = binary.Write(&hdr, binary.LittleEndian, uint64(total))
	_ = binary.Write(&hdr, binary.LittleEndian, int32(contentStart))
	hdr.WriteString(desc)
	hdr.WriteByte(0)
	out = append(out, hdr.Bytes()...)

	var evb [2]byte
	binary.LittleEndian.PutUint16(evb[:], ev)
	out = append(out, evb[:]...)
	out = append(out, chunks[rootIdx].data...)
	for i := range chunks {
		if i == rootIdx {
			continue
		}
		out = append(out, chunks[i].data...)
	}

	// Patch encrypted offsets. Reader.ReadWzOffset computes, at field
	// position p: off = rotl((^(p-cs))*hash - 0x581C3F6D, low5); then
	// target = (off ^ enc) + 2*cs. So enc = off ^ uint32(target - 2*cs).
	cs := uint32(contentStart)
	for ci := range chunks {
		for _, pt := range chunks[ci].patches {
			fieldPos := pos[ci] + pt.pos
			off := uint32(fieldPos) - cs
			off = ^off
			off *= hash
			off -= 0x581C3F6D
			off = rotl32(off, byte(off&0x1F))
			enc := off ^ uint32(int64(pos[pt.target])-int64(cs)*2)
			binary.LittleEndian.PutUint32(out[fieldPos:], enc)
		}
	}
	return out, nil
}
