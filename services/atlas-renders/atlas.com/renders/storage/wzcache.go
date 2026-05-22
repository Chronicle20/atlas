package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
)

// WZCache owns parsed *wz.File handles keyed by (scope, region, version,
// archive). The first request for a given key downloads the archive from
// MinIO bucket BucketWZ into wzScratchDir/<scope>/<region>/<version>/<archive>
// and opens it with wz.Open; subsequent requests reuse the open handle.
//
// Memory cost per entry is the parsed directory tree plus whatever images
// have had Properties() called on them — the parser is lazy on .img content
// and never holds canvas pixels (those live in the file on disk and are
// read via positional ReadAt on demand; see libs/atlas-wz/wz/file.go:106-117).
// Steady-state working set is in the low hundreds of MB for the Map.wz
// case even after a sustained run of distinct map renders.
//
// Concurrent calls for the same key are serialized by a per-key sync.Once;
// the loser of the race waits for the winner and reads the same *wz.File.
type WZCache struct {
	mc           *MC
	bucket       string
	scratchDir   string
	mu           sync.Mutex
	entries      map[string]*wzEntry
	l            logrus.FieldLogger
}

type wzEntry struct {
	once sync.Once
	file *wz.File
	err  error
}

// NewWZCache constructs a cache backed by mc. The scratchDir is created if
// it does not already exist; callers are expected to provide a directory
// on a writable volume (in production: an emptyDir).
func NewWZCache(l logrus.FieldLogger, mc *MC, bucket, scratchDir string) (*WZCache, error) {
	if err := os.MkdirAll(scratchDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir wz scratch %s: %w", scratchDir, err)
	}
	return &WZCache{
		l:          l,
		mc:         mc,
		bucket:     bucket,
		scratchDir: scratchDir,
		entries:    make(map[string]*wzEntry),
	}, nil
}

// Get returns the parsed *wz.File for (scope, region, version, archive),
// downloading + opening it on first miss. Subsequent calls reuse the open
// handle. The archive parameter is the WZ file name with extension, e.g.
// "Map.wz".
//
// scope is one of "shared" or "tenants/<id>" (the same shape the rest of
// atlas-renders uses for MinIO keys). The on-disk path uses the scope
// verbatim with `/` collapsed to `_` so multi-segment scopes (`tenants/<id>`)
// don't pollute the scratch directory layout.
func (c *WZCache) Get(ctx context.Context, scope, region, version, archive string) (*wz.File, error) {
	key := fmt.Sprintf("%s|%s|%s|%s", scope, region, version, archive)
	c.mu.Lock()
	e, ok := c.entries[key]
	if !ok {
		e = &wzEntry{}
		c.entries[key] = e
	}
	c.mu.Unlock()

	e.once.Do(func() {
		bucketKey := fmt.Sprintf("%s/regions/%s/versions/%s/%s", scope, region, version, archive)
		localPath := filepath.Join(c.scratchDir, sanitizeScope(scope), region, version, archive)
		if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
			e.err = fmt.Errorf("mkdir %s: %w", filepath.Dir(localPath), err)
			return
		}
		c.l.Infof("wzcache: downloading %s/%s -> %s", c.bucket, bucketKey, localPath)
		if err := c.mc.FGet(ctx, c.bucket, bucketKey, localPath); err != nil {
			e.err = fmt.Errorf("download %s/%s: %w", c.bucket, bucketKey, err)
			return
		}
		f, err := wz.Open(c.l, localPath)
		if err != nil {
			_ = os.Remove(localPath)
			e.err = fmt.Errorf("open %s: %w", localPath, err)
			return
		}
		e.file = f
		c.l.Infof("wzcache: opened %s for %s", archive, key)
	})
	return e.file, e.err
}

// Close releases all cached *wz.File handles. Intended for graceful shutdown.
func (c *WZCache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.entries {
		if e.file != nil {
			e.file.Close()
		}
	}
}

// sanitizeScope flattens "tenants/<id>" or "shared" into a single path
// segment for filesystem layout. Mirrors runtime/rest/jobs.go:sanitizeLabel
// in intent but stays inside this package to avoid cross-service coupling.
func sanitizeScope(scope string) string {
	out := make([]byte, 0, len(scope))
	for i := 0; i < len(scope); i++ {
		c := scope[i]
		if c == '/' {
			out = append(out, '_')
		} else {
			out = append(out, c)
		}
	}
	return string(out)
}
