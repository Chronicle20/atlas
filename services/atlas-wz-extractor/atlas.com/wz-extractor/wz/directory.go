package wz

import (
	"fmt"
	"io"
	"strings"
)

// Directory represents a directory node in the WZ tree.
type Directory struct {
	name        string
	directories []*Directory
	images      []*Image
}

// Name returns the directory name.
func (d *Directory) Name() string {
	return d.name
}

// Directories returns child directories.
func (d *Directory) Directories() []*Directory {
	return d.directories
}

// Images returns child images.
func (d *Directory) Images() []*Image {
	return d.images
}

// parseDirectory parses a WZ directory structure from the current position.
func (wz *File) parseDirectory(name string) (*Directory, error) {
	r := wz.reader

	count, err := r.ReadWzInt()
	if err != nil {
		return nil, fmt.Errorf("unable to read directory entry count: %w", err)
	}

	dir := &Directory{name: name}

	for i := int32(0); i < count; i++ {
		elemType, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		var entryName string
		switch elemType {
		case 1:
			// Unknown entry type - skip 10 bytes
			if err := r.Skip(10); err != nil {
				return nil, err
			}
			continue
		case 2:
			// UOL reference - name at offset
			offset, err := r.ReadInt32()
			if err != nil {
				return nil, err
			}
			absOffset := int64(offset) + wz.contentStart
			err = r.Peek(func() error {
				if _, err := r.Seek(absOffset, io.SeekStart); err != nil {
					return err
				}
				elemType, err = r.ReadByte()
				if err != nil {
					return err
				}
				entryName, err = r.ReadWzString()
				return err
			})
			if err != nil {
				return nil, err
			}
		case 3, 4:
			// 3 = sub-directory, 4 = image
			entryName, err = r.ReadWzString()
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown directory entry type: %d", elemType)
		}

		// Read size, checksum, data offset
		size, err := r.ReadWzInt()
		if err != nil {
			return nil, err
		}
		_, err = r.ReadWzInt() // checksum
		if err != nil {
			return nil, err
		}
		dataOffset, err := r.ReadWzOffset(uint32(wz.contentStart), wz.versionHash)
		if err != nil {
			return nil, err
		}

		if elemType == 3 {
			// Sub-directory: parse recursively
			err := r.Peek(func() error {
				if _, err := r.Seek(int64(dataOffset), io.SeekStart); err != nil {
					return err
				}
				subDir, err := wz.parseDirectory(entryName)
				if err != nil {
					return err
				}
				dir.directories = append(dir.directories, subDir)
				return nil
			})
			if err != nil {
				wz.l.WithError(err).Warnf("Unable to parse sub-directory [%s].", entryName)
			}
		} else {
			// Image entry: defer parsing (lazy load)
			img := &Image{
				name:       strings.TrimSuffix(entryName, ".img"),
				wzFile:     wz,
				dataOffset: int64(dataOffset),
				dataSize:   size,
			}
			dir.images = append(dir.images, img)
		}
	}

	return dir, nil
}
