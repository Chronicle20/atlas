package storage

import "github.com/sirupsen/logrus"

// Storage wires the MinIO client + LRU caches together. Handlers consume this
// type, never the raw *MC, so cache hits are transparent.
type Storage struct {
	Cfg    Config
	MC     *MC
	Caches *Caches
	// WZ caches parsed *wz.File handles for Map.wz / Character.wz / etc.
	// Populated lazily — the cache is constructed eagerly in New so handlers
	// don't have to nil-check, but its first Get downloads the archive.
	// Nil only when storage init explicitly skipped WZ caching for tests.
	WZ *WZCache
}

// New constructs a Storage with default cache sizes (256 atlas / 64 map /
// 1024 scope, per design §13).
func New(l logrus.FieldLogger, cfg Config) (*Storage, error) {
	mc, err := NewMC(cfg)
	if err != nil {
		return nil, err
	}
	wzc, err := NewWZCache(l, mc, cfg.BucketWZ, cfg.WZScratchDir)
	if err != nil {
		return nil, err
	}
	return &Storage{
		Cfg:    cfg,
		MC:     mc,
		Caches: NewCaches(256, 64, 1024),
		WZ:     wzc,
	}, nil
}
