package wz

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"

	"github.com/sirupsen/logrus"
)

// File represents a parsed WZ file.
//
// The lazy parse path (per-Image Properties() / parse()) drives Seek+Read on
// the underlying *os.File. Multiple goroutines hitting different *wz.Image
// instances backed by the same *wz.File would otherwise race the seek
// cursor. atlas-renders' WZCache (storage/wzcache.go) intentionally shares a
// single *wz.File across concurrent map renders, so this mutex is
// load-bearing — without it the parser produces torn property trees under
// load.
//
// parseMu serialises any Reader.Seek-based parsing. ReadCanvasData uses
// positional ReadAt and stays outside this mutex; canvas decompression is
// the hot path during compositing and benefits from staying concurrent.
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
	parseMu       sync.Mutex
	// keyRanges maps image byte extents to per-image fallback keys
	// (task-172 C-2). Written under parseMu during lazy image parse; read
	// concurrently by canvas decompression, hence its own RWMutex.
	keyRangesMu sync.RWMutex
	keyRanges   []keyRange
	// parent links a sub-archive veneer (NewSubFile) back to the File that
	// owns the underlying reader, key table, and parse mutex. nil for
	// normally-opened files (task-172 C-3, monolithic Data.wz).
	parent *File
}

// keyRange records that bytes in [start, end) decode with key (a per-image
// fallback key, task-172 C-2).
type keyRange struct {
	start, end int64
	key        []byte
}

func (wz *File) registerImageKey(start int64, size int32, key []byte) {
	if wz.parent != nil {
		wz.parent.registerImageKey(start, size, key)
		return
	}
	wz.keyRangesMu.Lock()
	defer wz.keyRangesMu.Unlock()
	wz.keyRanges = append(wz.keyRanges, keyRange{start: start, end: start + int64(size), key: key})
}

// CanvasEncryptionKeyFor returns the canvas-block decryption key for a
// canvas whose data begins at offset: the per-image fallback key when the
// owning image parsed under one (its canvases lie inside the image's byte
// extent), else the file-level key. Canvas decompression call sites must
// use this instead of CanvasEncryptionKey (task-172 C-2).
func (wz *File) CanvasEncryptionKeyFor(offset int64) []byte {
	if wz.parent != nil {
		return wz.parent.CanvasEncryptionKeyFor(offset)
	}
	wz.keyRangesMu.RLock()
	defer wz.keyRangesMu.RUnlock()
	for _, kr := range wz.keyRanges {
		if offset >= kr.start && offset < kr.end {
			return kr.key
		}
	}
	return wz.CanvasEncryptionKey()
}

// LockParse acquires the file-wide parse mutex and returns an unlock func
// for `defer`. Used by lazy-parse paths (Image.parse) to serialise
// Seek+Read sequences across goroutines. Sub-file views (NewSubFile)
// delegate to the parent so parsing under a sub-root serialises against
// parsing under the parent or any sibling sub-view (task-172 C-3).
func (wz *File) LockParse() func() {
	if wz.parent != nil {
		return wz.parent.LockParse()
	}
	wz.parseMu.Lock()
	return wz.parseMu.Unlock
}

// NewFileWithRoot constructs an in-memory File backed only by the given root
// directory. No real file on disk is needed; ReadCanvasData will fail if
// called (pass a stubbed canvasWriter to avoid that in tests).
// Intended for constructing in-memory WZ trees in tests and tooling.
func NewFileWithRoot(name string, root *Directory) *File {
	return &File{name: name, root: root}
}

// NewSubFile returns a virtual archive rooted at a subdirectory of parent,
// sharing the parent's reader, encryption key, version, and parse
// serialization (task-172 C-3 — monolithic Data.wz category views). The
// images under root keep pointing at parent, so lazy parsing naturally
// locks the parent's parseMu. Close on a sub-file is a no-op: the parent
// owns the file handle and must outlive every sub-file view.
func NewSubFile(parent *File, root *Directory, name string) *File {
	return &File{
		l:             parent.l,
		path:          parent.path,
		name:          name,
		f:             parent.f,
		reader:        parent.reader,
		root:          root,
		contentStart:  parent.contentStart,
		versionHash:   parent.versionHash,
		gameVersion:   parent.gameVersion,
		encryptionKey: parent.encryptionKey,
		parent:        parent,
	}
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

// Close releases resources associated with the WZ file. No-op for sub-file
// views — the parent owns the handle.
func (wz *File) Close() {
	if wz.parent != nil {
		return
	}
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

// GameVersion returns the game version detected while opening the archive
// (e.g. 83). Zero for in-memory files constructed via NewFileWithRoot.
func (wz *File) GameVersion() int {
	return wz.gameVersion
}

// EncryptionKey returns the WZ encryption key.
func (wz *File) EncryptionKey() *crypto.WzKey {
	return wz.encryptionKey
}

// ReadCanvasData reads raw canvas data from the WZ file at the given offset and size.
// The first byte at the offset is a flag/header byte that is skipped (matching MapleLib's approach).
// Returns (size - 1) bytes of actual compressed canvas data.
//
// Uses os.File.ReadAt (positional read) which does not modify the file's seek
// pointer and is safe for concurrent calls from multiple goroutines.
func (wz *File) ReadCanvasData(offset int64, size int32) ([]byte, error) {
	if size <= 1 {
		return nil, nil
	}
	return wz.reader.ReadAt(offset+1, int(size-1))
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

// detectVersion runs two-phase detection (task-172 C-1):
//
// Phase 1 — version: brute-force 1..1000 validating the first directory
// entry's decoded offset. Offset decryption depends only on the version
// hash, never on the AES key, so a single probe key suffices. (The old
// code probed once per encryption type and locked in whichever key came
// first — for unencrypted archives that silently selected the GMS key and
// every name decoded to garbage.)
//
// Phase 2 — key: with the version fixed, decode the first directory-entry
// names under each candidate key and keep candidates whose names are sane
// printable ASCII. Exactly one candidate must survive; zero or several is
// a hard, descriptive error — never a silent guess.
func (wz *File) detectVersion() error {
	r := wz.reader

	if _, err := r.Seek(wz.contentStart, io.SeekStart); err != nil {
		return err
	}
	encryptedVersion, err := r.ReadUInt16()
	if err != nil {
		return err
	}

	probeKey := crypto.GetKeyForRegion(crypto.EncryptionNone)
	version := 0
	var hash uint32
	for v := 1; v <= 1000; v++ {
		ev, h := crypto.CalculateVersionHash(v)
		if ev != encryptedVersion {
			continue
		}
		if wz.tryParseWithVersion(h, probeKey) {
			version, hash = v, h
			break
		}
	}
	if version == 0 {
		return fmt.Errorf("unable to detect WZ version for encrypted version %d", encryptedVersion)
	}

	type candidate struct {
		enc crypto.EncryptionType
		key *crypto.WzKey
	}
	var sane []candidate
	var tried []string
	for _, enc := range crypto.AllEncryptionTypes() {
		tried = append(tried, enc.String())
		key := crypto.GetKeyForRegion(enc)
		names, err := wz.readFirstEntryNames(hash, key, 4)
		if err != nil || len(names) == 0 {
			continue
		}
		if allSaneEntryNames(names) {
			sane = append(sane, candidate{enc: enc, key: key})
		}
	}
	switch len(sane) {
	case 1:
		wz.versionHash = hash
		wz.gameVersion = version
		wz.encryptionKey = sane[0].key
		wz.reader.SetKey(sane[0].key.Bytes(0x10000))
		wz.l.Infof("Detected version %d (hash=%d) with encryption=%v, keyEmpty=%v", version, hash, sane[0].enc, sane[0].key.IsEmpty())
		return nil
	case 0:
		return fmt.Errorf("wz key detection: no encryption candidate (tried %s) produced sane directory-entry names for version %d", strings.Join(tried, ", "), version)
	default:
		var names []string
		for _, c := range sane {
			names = append(names, c.enc.String())
		}
		return fmt.Errorf("wz key detection: ambiguous — candidates %s all produced sane directory-entry names for version %d", strings.Join(names, ", "), version)
	}
}

// readFirstEntryNames decodes up to max directory-entry names from the root
// directory under the candidate key. Structural reads (lengths, sizes,
// offsets) are key-independent, so a wrong key changes only the decoded
// characters — which is exactly what the sanity check inspects. Runs during
// Open() before the File is published; no parseMu needed (same guarantee as
// tryParseWithVersion).
func (wz *File) readFirstEntryNames(hash uint32, key *crypto.WzKey, max int) ([]string, error) {
	r := wz.reader
	if _, err := r.Seek(wz.contentStart+2, io.SeekStart); err != nil {
		return nil, err
	}
	count, err := r.ReadWzInt()
	if err != nil {
		return nil, err
	}
	if count <= 0 || count > 100000 {
		return nil, fmt.Errorf("implausible directory entry count %d", count)
	}
	savedKey := r.Key()
	r.SetKey(key.Bytes(0x10000))
	defer r.SetKey(savedKey)

	var names []string
	for i := int32(0); i < count && len(names) < max; i++ {
		elemType, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		switch elemType {
		case 1:
			if err := r.Skip(10); err != nil {
				return nil, err
			}
			continue
		case 2:
			if _, err := r.ReadInt32(); err != nil {
				return nil, err
			}
		case 3, 4:
			name, err := r.ReadWzString()
			if err != nil {
				return nil, err
			}
			names = append(names, name)
		default:
			return nil, fmt.Errorf("unknown directory entry type: %d", elemType)
		}
		if _, err := r.ReadWzInt(); err != nil { // size
			return nil, err
		}
		if _, err := r.ReadWzInt(); err != nil { // checksum
			return nil, err
		}
		if _, err := r.ReadWzOffset(uint32(wz.contentStart), hash); err != nil {
			return nil, err
		}
	}
	return names, nil
}

// isSaneEntryName reports whether a decoded directory-entry name looks like
// real WZ content. Root entry names across every known client generation are
// printable ASCII ("Mob.img", "Character", "smap.img"); a wrong AES key
// decodes to pseudo-random bytes that fail this check with overwhelming
// probability.
func isSaneEntryName(s string) bool {
	if s == "" || len(s) > 100 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7E {
			return false
		}
	}
	return true
}

func allSaneEntryNames(names []string) bool {
	for _, n := range names {
		if !isSaneEntryName(n) {
			return false
		}
	}
	return true
}

// Concurrency: tryParseWithVersion runs only during Open() (called from
// detectVersion) before the *File is published to any consumer goroutine.
// The Seek+Read against wz.reader is therefore single-threaded by
// construction and needs no parseMu coverage. Image.parse() acquires
// parseMu for all post-Open seek-based parsing; this is the canonical
// invariant — adding new public seek paths requires either parseMu
// coverage or an analogous single-threaded guarantee.
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
