package wz

import (
	"atlas-wz-extractor/wz/property"
	"fmt"
	"io"
)

// Image represents a WZ image (a group of properties, corresponding to a .img entry).
type Image struct {
	name       string
	wzFile     *File
	dataOffset int64
	dataSize   int32
	properties []property.Property
	parsed     bool
}

// Name returns the image name.
func (i *Image) Name() string {
	return i.name
}

// Properties returns the parsed properties of this image. Parses on first access (lazy).
func (i *Image) Properties() []property.Property {
	if !i.parsed {
		if err := i.parse(); err != nil {
			i.wzFile.l.WithError(err).Warnf("Unable to parse image [%s].", i.name)
		}
		i.parsed = true
	}
	return i.properties
}

func (i *Image) parse() error {
	r := i.wzFile.reader

	if _, err := r.Seek(i.dataOffset, io.SeekStart); err != nil {
		return err
	}

	// Read the object tag
	tag, err := r.ReadWzStringBlock(i.wzFile.contentStart)
	if err != nil {
		return fmt.Errorf("unable to read image tag: %w", err)
	}

	if tag != "Property" {
		return fmt.Errorf("unexpected image tag: %s (expected Property)", tag)
	}

	// Skip 2 bytes (always 0)
	if err := r.Skip(2); err != nil {
		return err
	}

	props, err := i.wzFile.parsePropertyList()
	if err != nil {
		return err
	}
	i.properties = props
	return nil
}

// parsePropertyList reads a list of key-value property entries.
func (wz *File) parsePropertyList() ([]property.Property, error) {
	r := wz.reader

	count, err := r.ReadWzInt()
	if err != nil {
		return nil, err
	}

	props := make([]property.Property, 0, count)
	for j := int32(0); j < count; j++ {
		name, err := r.ReadWzStringBlock(wz.contentStart)
		if err != nil {
			return nil, err
		}

		propType, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		prop, err := wz.parsePropertyValue(name, propType)
		if err != nil {
			return nil, fmt.Errorf("error parsing property [%s] type %d: %w", name, propType, err)
		}
		if prop != nil {
			props = append(props, prop)
		}
	}

	return props, nil
}

// parsePropertyValue parses a single property value based on its type tag.
func (wz *File) parsePropertyValue(name string, propType byte) (property.Property, error) {
	r := wz.reader

	switch propType {
	case 0:
		// Null
		return property.NewNull(name), nil

	case 2, 11:
		// Short (int16)
		v, err := r.ReadInt16()
		if err != nil {
			return nil, err
		}
		return property.NewShort(name, v), nil

	case 3, 19:
		// Int (WZ compressed int32)
		v, err := r.ReadWzInt()
		if err != nil {
			return nil, err
		}
		return property.NewInt(name, v), nil

	case 20:
		// Long (WZ compressed int64)
		v, err := r.ReadWzLong()
		if err != nil {
			return nil, err
		}
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
			return property.NewFloat(name, v), nil
		}
		return property.NewFloat(name, 0), nil

	case 5:
		// Double
		v, err := r.ReadFloat64()
		if err != nil {
			return nil, err
		}
		return property.NewDouble(name, v), nil

	case 8:
		// String
		v, err := r.ReadWzStringBlock(wz.contentStart)
		if err != nil {
			return nil, err
		}
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

		prop, err := wz.parseExtendedProperty(name)
		if err != nil {
			// Skip to end of sub-object on error
			_, _ = r.Seek(endPos, io.SeekStart)
			return nil, err
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

// parseExtendedProperty parses an extended (type 9) sub-object.
func (wz *File) parseExtendedProperty(name string) (property.Property, error) {
	r := wz.reader

	tag, err := r.ReadWzStringBlock(wz.contentStart)
	if err != nil {
		return nil, err
	}

	switch tag {
	case "Property":
		if err := r.Skip(2); err != nil {
			return nil, err
		}
		children, err := wz.parsePropertyList()
		if err != nil {
			return nil, err
		}
		return property.NewSub(name, children), nil

	case "Canvas":
		return wz.parseCanvasProperty(name)

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
			child, err := wz.parseExtendedProperty(fmt.Sprintf("%d", k))
			if err != nil {
				return nil, err
			}
			children = append(children, child)
		}
		return property.NewConvex(name, children), nil

	case "UOL":
		if err := r.Skip(1); err != nil {
			return nil, err
		}
		v, err := r.ReadWzStringBlock(wz.contentStart)
		if err != nil {
			return nil, err
		}
		return property.NewUOL(name, v), nil

	case "Sound_DX8":
		return wz.parseSoundProperty(name)

	default:
		return nil, fmt.Errorf("unknown extended property tag: %s", tag)
	}
}

// parseCanvasProperty parses a canvas (image) property.
func (wz *File) parseCanvasProperty(name string) (property.Property, error) {
	r := wz.reader

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
		children, err = wz.parsePropertyList()
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
