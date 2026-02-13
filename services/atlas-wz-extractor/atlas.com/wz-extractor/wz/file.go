package wz

import (
	"atlas-wz-extractor/wz/crypto"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// File represents a parsed WZ file.
type File struct {
	l             logrus.FieldLogger
	path          string
	name          string
	f             *os.File
	reader        *Reader
	root          *Directory
	contentStart  int64
	versionHash   uint32
	gameVersion   int
	encryptionKey *crypto.WzKey
}

// Open opens and parses a WZ file from disk.
func Open(l logrus.FieldLogger, path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open WZ file: %w", err)
	}

	wz := &File{
		l:      l,
		path:   path,
		name:   strings.TrimSuffix(filepath.Base(path), ".wz"),
		f:      f,
		reader: NewReader(f),
	}

	if err := wz.parseHeader(); err != nil {
		f.Close()
		return nil, fmt.Errorf("unable to parse WZ header: %w", err)
	}

	if err := wz.detectVersion(); err != nil {
		f.Close()
		return nil, fmt.Errorf("unable to detect WZ version: %w", err)
	}

	if err := wz.parseRoot(); err != nil {
		f.Close()
		return nil, fmt.Errorf("unable to parse WZ root: %w", err)
	}

	return wz, nil
}

// Close releases resources associated with the WZ file.
func (wz *File) Close() {
	if wz.f != nil {
		wz.f.Close()
	}
}

// Name returns the WZ file name (without .wz extension).
func (wz *File) Name() string {
	return wz.name
}

// Root returns the root directory of the WZ file.
func (wz *File) Root() *Directory {
	return wz.root
}

// Reader returns the underlying binary reader.
func (wz *File) Reader() *Reader {
	return wz.reader
}

// ContentStart returns the content start offset.
func (wz *File) ContentStart() int64 {
	return wz.contentStart
}

// VersionHash returns the computed version hash.
func (wz *File) VersionHash() uint32 {
	return wz.versionHash
}

// EncryptionKey returns the WZ encryption key.
func (wz *File) EncryptionKey() *crypto.WzKey {
	return wz.encryptionKey
}

// ReadCanvasData reads raw canvas data from the WZ file at the given offset and size.
func (wz *File) ReadCanvasData(offset int64, size int32) ([]byte, error) {
	if _, err := wz.reader.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	return wz.reader.ReadBytes(int(size))
}

// CanvasEncryptionKey returns the raw key bytes for canvas block decryption.
func (wz *File) CanvasEncryptionKey() []byte {
	if wz.encryptionKey == nil {
		return nil
	}
	return wz.encryptionKey.Bytes(0x10000)
}

func (wz *File) parseHeader() error {
	r := wz.reader

	// Read magic bytes "PKG1"
	magic, err := r.ReadASCIIString(4)
	if err != nil {
		return err
	}
	if magic != "PKG1" {
		return fmt.Errorf("invalid WZ magic: expected PKG1, got %s", magic)
	}

	// Skip file size (8 bytes) - uint64
	if err := r.Skip(8); err != nil {
		return err
	}

	// Read content start position
	contentStart, err := r.ReadInt32()
	if err != nil {
		return err
	}
	wz.contentStart = int64(contentStart)

	// Read file description (null-terminated string)
	_, err = r.ReadASCIIZString()
	if err != nil {
		return err
	}

	wz.l.Debugf("WZ header parsed: contentStart=%d", wz.contentStart)
	return nil
}

func (wz *File) detectVersion() error {
	r := wz.reader

	// Seek to content start to read encrypted version
	if _, err := r.Seek(wz.contentStart, io.SeekStart); err != nil {
		return err
	}

	encryptedVersion, err := r.ReadUInt16()
	if err != nil {
		return err
	}

	// Try each encryption type
	for _, enc := range crypto.AllEncryptionTypes() {
		key := crypto.GetKeyForRegion(enc)

		// Brute-force the version (1-1000)
		for version := 1; version <= 1000; version++ {
			ev, hash := crypto.CalculateVersionHash(version)
			if ev != encryptedVersion {
				continue
			}

			// Try parsing with this version hash and key
			if wz.tryParseWithVersion(hash, key) {
				wz.versionHash = hash
				wz.gameVersion = version
				wz.encryptionKey = key
				wz.reader.SetKey(key.Bytes(0x10000))
				wz.l.Debugf("Detected version %d (hash=%d) with encryption=%d", version, hash, enc)
				return nil
			}
		}
	}

	return fmt.Errorf("unable to detect WZ version for encrypted version %d", encryptedVersion)
}

func (wz *File) tryParseWithVersion(hash uint32, key *crypto.WzKey) bool {
	r := wz.reader

	// Position after the encrypted version uint16
	if _, err := r.Seek(wz.contentStart+2, io.SeekStart); err != nil {
		return false
	}

	// Try to read the directory entry count and first few entries
	count, err := r.ReadWzInt()
	if err != nil || count <= 0 || count > 100000 {
		return false
	}

	// Try to read the first entry - if it fails, this isn't the right version
	for i := int32(0); i < count && i < 1; i++ {
		elemType, err := r.ReadByte()
		if err != nil {
			return false
		}

		switch elemType {
		case 1:
			// Skip 10 bytes
			if err := r.Skip(10); err != nil {
				return false
			}
		case 2:
			// UOL - read offset
			_, err := r.ReadInt32()
			if err != nil {
				return false
			}
		case 3, 4:
			// Directory or image - read name
			savedKey := r.Key()
			r.SetKey(key.Bytes(0x10000))
			_, err := r.ReadWzString()
			r.SetKey(savedKey)
			if err != nil {
				return false
			}
		default:
			return false
		}

		// Read size, checksum, offset
		if _, err := r.ReadWzInt(); err != nil {
			return false
		}
		if _, err := r.ReadWzInt(); err != nil {
			return false
		}

		// Try reading the WZ offset - this validates the hash
		offset, err := r.ReadWzOffset(uint32(wz.contentStart), hash)
		if err != nil {
			return false
		}

		// Sanity check: offset should be within file bounds
		fileInfo, err := wz.f.Stat()
		if err != nil {
			return false
		}
		if int64(offset) >= fileInfo.Size() || offset == 0 {
			return false
		}
	}

	return true
}

func (wz *File) parseRoot() error {
	r := wz.reader

	// Seek past the encrypted version
	if _, err := r.Seek(wz.contentStart+2, io.SeekStart); err != nil {
		return err
	}

	root, err := wz.parseDirectory(wz.name)
	if err != nil {
		return err
	}
	wz.root = root
	return nil
}
