package storage

// Storage wires the MinIO client + LRU caches together. Handlers consume this
// type, never the raw *MC, so cache hits are transparent.
type Storage struct {
	Cfg    Config
	MC     *MC
	Caches *Caches
}

// New constructs a Storage with default cache sizes (256 atlas / 64 map /
// 1024 scope, per design §13).
func New(cfg Config) (*Storage, error) {
	mc, err := NewMC(cfg)
	if err != nil {
		return nil, err
	}
	return &Storage{
		Cfg:    cfg,
		MC:     mc,
		Caches: NewCaches(256, 64, 1024),
	}, nil
}
