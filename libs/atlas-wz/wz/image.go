package wz

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// errBadImageTag marks the tag-validation failure that triggers the
// per-image key fallback (task-172 C-2). Any other parse error (I/O,
// truncation, unknown property type) must NOT trigger a retry.
var errBadImageTag = errors.New("unexpected image tag")

// Image represents a WZ image (a group of properties, corresponding to a .img entry).
type Image struct {
	name       string
	wzFile     *File
	dataOffset int64
	dataSize   int32
	properties []property.Property
	// parsed is read on Properties()'s lock-free fast path and written
	// under File.parseMu on the slow path; atomic.Bool makes that
	// cross-goroutine publish race-free (task-172 C-2 — the per-image key
	// fallback's own concurrency test was the first to call Properties()
	// from multiple goroutines against the *same* *Image, which exposed a
	// pre-existing unsynchronized read here under `go test -race`).
	parsed   atomic.Bool
	parseErr error
	// keyOverride is the fallback key this image parsed under, nil when the
	// file-level key worked. Set under File.parseMu (task-172 C-2, mixed
	// per-image encryption — JMS List.wz-listed images).
	keyOverride []byte
}

// NewParsedImage constructs an Image whose properties are already populated
// (parsed == true). Intended for constructing in-memory WZ trees in tests and
// tooling without requiring a real WZ file on disk.
func NewParsedImage(name string, props []property.Property) *Image {
	img := &Image{
		name:       name,
		properties: props,
	}
	img.parsed.Store(true)
	return img
}

// Name returns the image name.
func (i *Image) Name() string {
	return i.name
}

// File returns the WZ file that backs this image, or nil for in-memory images
// constructed via NewParsedImage. Required by callers that need to dereference
// canvas data (ReadCanvasData lives on *File).
func (i *Image) File() *File {
	return i.wzFile
}

// Properties returns the parsed properties of this image plus any error
// observed during lazy parsing. Parses on first access; subsequent calls
// return the cached result.
//
// Returning the error surface forces every caller to make an explicit
// decision about parse failures instead of silently consuming an empty
// property slice. See task-076 F6: a previous version logged the error
// and dropped it, which made downstream zero-row imports indistinguishable
// from "parsed and genuinely empty."
//
// Goroutine safety: parse() Seek+Reads the shared *os.File; the file-wide
// parseMu in *File serialises every Seek-based parse. The double-check
// inside the critical section makes the lazy initialisation idempotent.
//
// In-memory images created via NewParsedImage have parsed=true and
// wzFile=nil, so they short-circuit without lock acquisition and always
// return a nil error.
func (i *Image) Properties() ([]property.Property, error) {
	if i.parsed.Load() {
		return i.properties, i.parseErr
	}
	if i.wzFile == nil {
		i.parsed.Store(true)
		return i.properties, nil
	}
	unlock := i.wzFile.LockParse()
	defer unlock()
	if i.parsed.Load() {
		return i.properties, i.parseErr
	}
	if err := i.parse(); err != nil {
		i.wzFile.l.WithError(err).Warnf("Unable to parse image [%s].", i.name)
		i.parseErr = err
		i.parsed.Store(true)
		return i.properties, err
	}
	i.parsed.Store(true)
	return i.properties, nil
}

// parse decodes the image with the file-level key; when the image tag fails
// validation it retries under each other known key (task-172 C-2 — JMS
// archives mix unencrypted and KMS-encrypted images in one file). On a
// fallback hit the winning key is cached for this image's strings and
// registered for canvas-block decryption. Caller holds File.parseMu.
func (i *Image) parse() error {
	fileKey := i.wzFile.reader.Key()
	err := i.parseWithKey(fileKey)
	if err == nil || !errors.Is(err, errBadImageTag) {
		return err
	}
	for _, enc := range crypto.AllEncryptionTypes() {
		kb := crypto.GetKeyForRegion(enc).Bytes(0x10000)
		if bytes.Equal(kb, fileKey) {
			continue
		}
		if retryErr := i.parseWithKey(kb); retryErr == nil {
			i.keyOverride = kb
			i.wzFile.registerImageKey(i.dataOffset, i.dataSize, kb)
			i.wzFile.l.Debugf("image [%s]: parsed with fallback key (%v)", i.name, enc)
			return nil
		}
	}
	return err
}

// parseWithKey runs one parse attempt with the reader temporarily switched
// to key, restoring the file-level key afterwards. Caller holds parseMu.
func (i *Image) parseWithKey(key []byte) error {
	r := i.wzFile.reader
	hook := i.wzFile.traceHook()
	savedKey := r.Key()
	r.SetKey(key)
	defer r.SetKey(savedKey)

	if _, err := r.Seek(i.dataOffset, io.SeekStart); err != nil {
		return err
	}
	var tagStart int64
	if hook != nil {
		var err error
		tagStart, err = r.Pos()
		if err != nil {
			return err
		}
	}
	tag, err := r.ReadWzStringBlock(i.dataOffset)
	if err != nil {
		return fmt.Errorf("unable to read image tag: %w", err)
	}
	if hook != nil {
		if tagEnd, perr := r.Pos(); perr == nil {
			hook(TraceEvent{Path: "", Kind: "stringblock", Name: "", StartOff: tagStart, EndOff: tagEnd, Detail: tag})
		}
	}
	if tag != "Property" {
		return fmt.Errorf("%w: %s (expected Property)", errBadImageTag, tag)
	}
	if err := r.Skip(2); err != nil {
		return err
	}
	props, err := i.wzFile.parsePropertyList("", i.dataOffset)
	if err != nil {
		return err
	}
	i.properties = props
	return nil
}

// parsePropertyList reads a list of key-value property entries.
// imageOffset is the base offset for resolving offset-referenced strings
// within the image.
//
// Invariant: caller holds wz.parseMu. Entered via Image.parse() which
// acquires the lock unconditionally. Future contributors must not call
// this from outside that path without first acquiring the lock — the
// underlying wz.reader is shared across all Image instances backed by
// the same *File and is not safe to Seek concurrently.
func (wz *File) parsePropertyList(path string, imageOffset int64) ([]property.Property, error) {
	r := wz.reader
	hook := wz.traceHook()

	var listStart int64
	var err error
	if hook != nil {
		listStart, err = r.Pos()
		if err != nil {
			return nil, err
		}
	}

	count, err := r.ReadWzInt()
	if err != nil {
		return nil, err
	}

	props := make([]property.Property, 0, count)
	for j := int32(0); j < count; j++ {
		var nameStart int64
		if hook != nil {
			nameStart, err = r.Pos()
			if err != nil {
				return nil, err
			}
		}
		name, err := r.ReadWzStringBlock(imageOffset)
		if err != nil {
			return nil, err
		}
		if hook != nil {
			if nameEnd, perr := r.Pos(); perr == nil {
				hook(TraceEvent{Path: path, Kind: "stringblock", Name: name, StartOff: nameStart, EndOff: nameEnd, Detail: name})
			}
		}

		propType, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		childPath := path + "/" + name
		prop, err := wz.parsePropertyValue(childPath, name, propType, imageOffset)
		if err != nil {
			return nil, fmt.Errorf("error parsing property [%s] type %d: %w", name, propType, err)
		}
		if prop != nil {
			props = append(props, prop)
		}
	}

	if hook != nil {
		if listEnd, perr := r.Pos(); perr == nil {
			hook(TraceEvent{Path: path, Kind: "list", StartOff: listStart, EndOff: listEnd, Detail: fmt.Sprintf("count=%d", count)})
		}
	}

	return props, nil
}

// parsePropertyValue parses a single property value based on its type tag.
// path is the property's full path from the image root (e.g. "/info/state"),
// used only for trace emission.
func (wz *File) parsePropertyValue(path, name string, propType byte, imageOffset int64) (property.Property, error) {
	r := wz.reader
	hook := wz.traceHook()

	var start int64
	if hook != nil {
		var err error
		start, err = r.Pos()
		if err != nil {
			return nil, err
		}
	}

	switch propType {
	case 0:
		// Null
		emitPropTrace(hook, r, path, name, propType, start)
		return property.NewNull(name), nil

	case 2, 11:
		// Short (int16)
		v, err := r.ReadInt16()
		if err != nil {
			return nil, err
		}
		emitPropTrace(hook, r, path, name, propType, start)
		return property.NewShort(name, v), nil

	case 3, 19:
		// Int (WZ compressed int32)
		v, err := r.ReadWzInt()
		if err != nil {
			return nil, err
		}
		emitPropTrace(hook, r, path, name, propType, start)
		return property.NewInt(name, v), nil

	case 20:
		// Long (WZ compressed int64)
		v, err := r.ReadWzLong()
		if err != nil {
			return nil, err
		}
		emitPropTrace(hook, r, path, name, propType, start)
		return property.NewLong(name, v), nil

	case 4:
		// Float
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == 0x80 {
			v, err := r.ReadFloat32()
			if err != nil {
				return nil, err
			}
			emitPropTrace(hook, r, path, name, propType, start)
			return property.NewFloat(name, v), nil
		}
		emitPropTrace(hook, r, path, name, propType, start)
		return property.NewFloat(name, 0), nil

	case 5:
		// Double
		v, err := r.ReadFloat64()
		if err != nil {
			return nil, err
		}
		emitPropTrace(hook, r, path, name, propType, start)
		return property.NewDouble(name, v), nil

	case 8:
		// String
		v, err := r.ReadWzStringBlock(imageOffset)
		if err != nil {
			return nil, err
		}
		emitPropTrace(hook, r, path, name, propType, start)
		return property.NewString(name, v), nil

	case 9:
		// Sub-object (extended type)
		size, err := r.ReadInt32()
		if err != nil {
			return nil, err
		}

		pos, err := r.Pos()
		if err != nil {
			return nil, err
		}
		endPos := pos + int64(size)

		prop, err := wz.parseExtendedProperty(path, name, imageOffset)
		if err != nil {
			// Skip to end of sub-object on error
			_, _ = r.Seek(endPos, io.SeekStart)
			return nil, err
		}

		// actualEnd is captured immediately before the recovery reseek
		// below — the whole point of this event is to make an over- or
		// under-read visible before it gets silently healed by the reseek
		// to endPos (task-262). Only captured when a trace hook is
		// installed; on the nil-hook path it costs no syscall.
		if hook != nil {
			actualEnd, perr := r.Pos()
			if perr != nil {
				return nil, perr
			}
			hook(TraceEvent{
				Path:        path,
				Kind:        "sub",
				Name:        name,
				Type:        propType,
				StartOff:    start,
				EndOff:      actualEnd,
				Detail:      fmt.Sprintf("declaredSize=%d endPos=%d actualEnd=%d", size, endPos, actualEnd),
				DeclaredEnd: endPos,
				ActualEnd:   actualEnd,
			})
		}

		// Ensure we're at the correct position after parsing
		if _, err := r.Seek(endPos, io.SeekStart); err != nil {
			return nil, err
		}

		return prop, nil

	default:
		return nil, fmt.Errorf("unknown property type: %d", propType)
	}
}

// parseExtendedProperty parses an extended (type 9) sub-object. path is the
// property's full path from the image root, used only for trace emission.
func (wz *File) parseExtendedProperty(path, name string, imageOffset int64) (property.Property, error) {
	r := wz.reader
	hook := wz.traceHook()

	var start int64
	var err error
	if hook != nil {
		start, err = r.Pos()
		if err != nil {
			return nil, err
		}
	}
	tag, err := r.ReadWzStringBlock(imageOffset)
	if err != nil {
		return nil, err
	}
	if hook != nil {
		if end, perr := r.Pos(); perr == nil {
			hook(TraceEvent{Path: path, Kind: "extended", Name: name, StartOff: start, EndOff: end, Detail: tag})
		}
	}

	switch tag {
	case "Property":
		if err := r.Skip(2); err != nil {
			return nil, err
		}
		children, err := wz.parsePropertyList(path, imageOffset)
		if err != nil {
			return nil, err
		}
		return property.NewSub(name, children), nil

	case "Canvas":
		return wz.parseCanvasProperty(path, name, imageOffset)

	case "Shape2D#Vector2D":
		x, err := r.ReadWzInt()
		if err != nil {
			return nil, err
		}
		y, err := r.ReadWzInt()
		if err != nil {
			return nil, err
		}
		return property.NewVector(name, x, y), nil

	case "Shape2D#Convex2D":
		count, err := r.ReadWzInt()
		if err != nil {
			return nil, err
		}
		children := make([]property.Property, 0, count)
		for k := int32(0); k < count; k++ {
			childName := fmt.Sprintf("%d", k)
			child, err := wz.parseExtendedProperty(path+"/"+childName, childName, imageOffset)
			if err != nil {
				return nil, err
			}
			children = append(children, child)
		}
		return property.NewConvex(name, children), nil

	case "UOL":
		var uolStart int64
		if hook != nil {
			var perr error
			uolStart, perr = r.Pos()
			if perr != nil {
				return nil, perr
			}
		}
		if err := r.Skip(1); err != nil {
			return nil, err
		}
		v, err := r.ReadWzStringBlock(imageOffset)
		if err != nil {
			return nil, err
		}
		if hook != nil {
			if end, perr := r.Pos(); perr == nil {
				hook(TraceEvent{Path: path, Kind: "uol", Name: name, StartOff: uolStart, EndOff: end, Detail: v})
			}
		}
		return property.NewUOL(name, v), nil

	case "Sound_DX8":
		return wz.parseSoundProperty(name)

	default:
		return nil, fmt.Errorf("unknown extended property tag: %s", tag)
	}
}

// parseCanvasProperty parses a canvas (image) property. path is the
// property's full path from the image root, used only for trace emission.
func (wz *File) parseCanvasProperty(path, name string, imageOffset int64) (property.Property, error) {
	r := wz.reader
	hook := wz.traceHook()

	var canvasStart int64
	if hook != nil {
		var err error
		canvasStart, err = r.Pos()
		if err != nil {
			return nil, err
		}
	}

	// Skip 1 byte
	if err := r.Skip(1); err != nil {
		return nil, err
	}

	// Check if canvas has child properties
	hasProperty, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	var children []property.Property
	if hasProperty > 0 {
		if err := r.Skip(2); err != nil {
			return nil, err
		}
		children, err = wz.parsePropertyList(path, imageOffset)
		if err != nil {
			return nil, err
		}
	}

	// Read canvas dimensions
	width, err := r.ReadWzInt()
	if err != nil {
		return nil, err
	}
	height, err := r.ReadWzInt()
	if err != nil {
		return nil, err
	}

	// Read format
	format, err := r.ReadWzInt()
	if err != nil {
		return nil, err
	}
	format2, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	pixelFormat := int(format) + int(format2)

	// Skip 4 bytes (must be 0)
	if err := r.Skip(4); err != nil {
		return nil, err
	}

	// Read data size
	dataSize, err := r.ReadInt32()
	if err != nil {
		return nil, err
	}

	// Record data offset (skip 1 byte header before actual data)
	dataOffset, err := r.Pos()
	if err != nil {
		return nil, err
	}

	// Skip past the canvas data
	if err := r.Skip(int64(dataSize)); err != nil {
		return nil, err
	}

	if hook != nil {
		if end, perr := r.Pos(); perr == nil {
			hook(TraceEvent{
				Path:     path,
				Kind:     "canvas",
				Name:     name,
				StartOff: canvasStart,
				EndOff:   end,
				Detail:   fmt.Sprintf("w=%d h=%d format=%d dataOffset=%d dataSize=%d hasProperty=%d", width, height, pixelFormat, dataOffset, dataSize, hasProperty),
			})
		}
	}

	return property.NewCanvas(name, int(width), int(height), pixelFormat, dataOffset, dataSize, children), nil
}

// parseSoundProperty parses a sound property (stub - just skips the data).
func (wz *File) parseSoundProperty(name string) (property.Property, error) {
	r := wz.reader

	// Skip 1 byte
	if err := r.Skip(1); err != nil {
		return nil, err
	}

	dataSize, err := r.ReadWzInt()
	if err != nil {
		return nil, err
	}

	// Skip duration
	if _, err := r.ReadWzInt(); err != nil {
		return nil, err
	}

	// Record offset and skip data
	offset, err := r.Pos()
	if err != nil {
		return nil, err
	}
	_ = offset

	if err := r.Skip(int64(dataSize)); err != nil {
		return nil, err
	}

	return property.NewSound(name), nil
}
