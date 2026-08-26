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
	KindFloatNoMarker
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
	W, H     int32 // Canvas dimensions
}

func Int(name string, v int32) Prop { return Prop{Name: name, Kind: KindInt, Int: v} }
func Str(name, v string) Prop       { return Prop{Name: name, Kind: KindString, Str: v} }
func Sub(name string, children ...Prop) Prop {
	return Prop{Name: name, Kind: KindSub, Children: children}
}

// Canvas builds a 1x1, childless canvas property (the historical fixture
// shape). Equivalent to CanvasWith(name, 1, 1, payload).
func Canvas(name string, payload []byte) Prop {
	return CanvasWith(name, 1, 1, payload)
}

// CanvasWith builds a canvas property with real dimensions and, optionally,
// child properties (e.g. "origin" Vector, "z"/"delay" Int) read back via
// parseCanvasProperty's hasProperty>0 path.
func CanvasWith(name string, w, h int32, payload []byte, children ...Prop) Prop {
	return Prop{Name: name, Kind: KindCanvas, Canvas: payload, W: w, H: h, Children: children}
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

// FloatNoMarker builds a type-4 float property whose marker byte is
// deliberately NOT 0x80. image.go's forced-zero branch never reads a
// float32 payload in that case, so no payload bytes follow the marker; the
// parser is expected to yield 0 having consumed only the marker byte.
func FloatNoMarker(name string) Prop { return Prop{Name: name, Kind: KindFloatNoMarker} }

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
	stringDedup  bool
}

func NewBuilder() *Builder {
	return &Builder{version: 83, enc: crypto.EncryptionNone}
}

func (b *Builder) SetVersion(v int) *Builder                        { b.version = v; return b }
func (b *Builder) SetEncryption(enc crypto.EncryptionType) *Builder { b.enc = enc; return b }

// SetStringDedup toggles offset-referenced string blocks (tag 0x01 for
// repeated property names, tag 0x1B for repeated extended-type tag strings
// such as "Property"/"Canvas"/"UOL"/"Shape2D#Vector2D"/"Shape2D#Convex2D")
// for the string pool real client-produced archives use, exercising
// Reader.ReadWzStringBlock's offset-seek path. Default off: every existing
// fixture is built with plain inline (tag 0x73) string blocks and its bytes
// must not change.
func (b *Builder) SetStringDedup(on bool) *Builder { b.stringDedup = on; return b }

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
// Reader.ReadWzStringBlock. Always inline: used for content strings (String
// property values, UOL targets) that this builder never deduplicates.
func writeStringBlock(buf *bytes.Buffer, s string, key []byte) error {
	buf.WriteByte(0x73)
	return writeWzString(buf, s, key)
}

// stringPool tracks, per image, the absolute (image-relative, i.e. relative
// to the image's own dataOffset) byte position of each string's first
// inline occurrence, so later occurrences can be written as tag-0x01/0x1B
// offset references instead of duplicating the string bytes. A nil pool
// means dedup is off (Builder.stringDedup false, the default): every call
// site below falls back to writeStringBlock's unconditional inline form,
// so existing fixtures stay byte-identical.
type stringPool struct {
	seen map[string]int
}

// writeStringBlockDedup emits a WZ string block that participates in the
// per-image string pool when pool is non-nil: the first occurrence of s
// within the image is written inline (tag 0x73) and its position recorded;
// later occurrences are written as an offset reference using dedupTag
// (conventionally 0x01 for property names, 0x1B for extended-type tag
// strings — Reader.ReadWzStringBlock resolves both identically).
//
// base is the absolute image-relative byte offset that buf.Len()==0
// corresponds to. writeExtendedContent's type-9 wrapping builds its body in
// a separate bytes.Buffer (to compute the int32 length prefix before it is
// known how long the body is) which is later copied verbatim into the
// parent buffer; base lets positions recorded while building that separate
// buffer still land in the flattened image's coordinate space, which is
// what the offset field the reader seeks by is relative to.
func writeStringBlockDedup(buf *bytes.Buffer, s string, key []byte, pool *stringPool, base int, dedupTag byte) error {
	if pool != nil {
		if pos, ok := pool.seen[s]; ok {
			buf.WriteByte(dedupTag)
			_ = binary.Write(buf, binary.LittleEndian, int32(pos))
			return nil
		}
	}
	buf.WriteByte(0x73)
	if pool != nil {
		pool.seen[s] = base + buf.Len()
	}
	return writeWzString(buf, s, key)
}

func writePropList(buf *bytes.Buffer, props []Prop, key []byte, pool *stringPool, base int) error {
	writeWzInt(buf, int32(len(props)))
	for _, p := range props {
		if err := writeStringBlockDedup(buf, p.Name, key, pool, base, 0x01); err != nil {
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
		case KindFloatNoMarker:
			buf.WriteByte(4)
			buf.WriteByte(0x00) // not 0x80: parser forces 0, no payload bytes follow
		case KindDouble:
			buf.WriteByte(5)
			_ = binary.Write(buf, binary.LittleEndian, p.Double)
		case KindString:
			buf.WriteByte(8)
			if err := writeStringBlock(buf, p.Str, key); err != nil {
				return err
			}
		case KindSub, KindCanvas, KindUOL, KindVector, KindConvex:
			// inner's bytes land in buf right after the type byte (1) and
			// int32 length prefix (4) written below; innerBase lets pool
			// positions recorded while building inner still resolve in the
			// image's own flattened coordinate space.
			innerBase := base + buf.Len() + 1 + 4
			var inner bytes.Buffer
			if err := writeExtendedContent(&inner, p, key, pool, innerBase); err != nil {
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
func writeExtendedContent(buf *bytes.Buffer, p Prop, key []byte, pool *stringPool, base int) error {
	switch p.Kind {
	case KindSub:
		if err := writeStringBlockDedup(buf, "Property", key, pool, base, 0x1B); err != nil {
			return err
		}
		buf.Write([]byte{0, 0})
		return writePropList(buf, p.Children, key, pool, base)
	case KindCanvas:
		if err := writeStringBlockDedup(buf, "Canvas", key, pool, base, 0x1B); err != nil {
			return err
		}
		buf.WriteByte(0) // skipped byte
		if len(p.Children) > 0 {
			buf.WriteByte(1) // hasProperty = 1
			buf.Write([]byte{0, 0})
			if err := writePropList(buf, p.Children, key, pool, base); err != nil {
				return err
			}
		} else {
			buf.WriteByte(0) // hasProperty = 0
		}
		writeWzInt(buf, p.W) // width
		writeWzInt(buf, p.H) // height
		writeWzInt(buf, 2)   // format
		buf.WriteByte(0)   // format2
		buf.Write([]byte{0, 0, 0, 0})
		_ = binary.Write(buf, binary.LittleEndian, int32(len(p.Canvas)+1))
		buf.WriteByte(0xAB) // flag byte skipped by ReadCanvasData
		buf.Write(p.Canvas)
		return nil
	case KindUOL:
		if err := writeStringBlockDedup(buf, "UOL", key, pool, base, 0x1B); err != nil {
			return err
		}
		buf.WriteByte(0) // skipped byte
		return writeStringBlock(buf, p.Str, key)
	case KindVector:
		if err := writeStringBlockDedup(buf, "Shape2D#Vector2D", key, pool, base, 0x1B); err != nil {
			return err
		}
		writeWzInt(buf, p.X)
		writeWzInt(buf, p.Y)
		return nil
	case KindConvex:
		if err := writeStringBlockDedup(buf, "Shape2D#Convex2D", key, pool, base, 0x1B); err != nil {
			return err
		}
		writeWzInt(buf, int32(len(p.Children)))
		for _, c := range p.Children {
			if err := writeExtendedContent(buf, c, key, pool, base); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("wztest: kind %d is not an extended property type", p.Kind)
	}
}

// buildImage serializes one image block with its effective key. When
// b.stringDedup is on, a fresh per-image stringPool is threaded through so
// repeated property names and extended-type tags are written as
// offset-referenced string blocks; positions are recorded relative to the
// image's own start (base 0), matching what every ReadWzStringBlock call
// site in image.go passes as fileStart (the image's dataOffset).
func (b *Builder) buildImage(img Image) (chunk, error) {
	key := keyBytes(b.enc)
	if img.Enc != nil {
		key = keyBytes(*img.Enc)
	}
	var pool *stringPool
	if b.stringDedup {
		pool = &stringPool{seen: make(map[string]int)}
	}
	var buf bytes.Buffer
	if err := writeStringBlockDedup(&buf, "Property", key, pool, 0, 0x1B); err != nil {
		return chunk{}, err
	}
	buf.Write([]byte{0, 0})
	if err := writePropList(&buf, img.Props, key, pool, 0); err != nil {
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
