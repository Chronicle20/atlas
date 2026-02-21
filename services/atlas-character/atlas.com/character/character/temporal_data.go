package character

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type temporalData struct {
	x      int16
	y      int16
	stance byte
}

func (d temporalData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		X      int16 `json:"x"`
		Y      int16 `json:"y"`
		Stance byte  `json:"stance"`
	}{X: d.x, Y: d.y, Stance: d.stance})
}

func (d *temporalData) UnmarshalJSON(data []byte) error {
	var raw struct {
		X      int16 `json:"x"`
		Y      int16 `json:"y"`
		Stance byte  `json:"stance"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.x = raw.X
	d.y = raw.Y
	d.stance = raw.Stance
	return nil
}

func (d *temporalData) X() int16 {
	return d.x
}

func (d *temporalData) Y() int16 {
	return d.y
}

func (d *temporalData) Stance() byte {
	return d.stance
}

// temporalRegistry wraps a TenantCoalescedRegistry to provide high-level
// semantic methods for character temporal data (position, stance).
// Writes are buffered locally and flushed to Redis periodically. Reads are
// served from a local cache with bounded staleness.
type temporalRegistry struct {
	reg *atlas.TenantCoalescedRegistry[uint32, temporalData]
}

func (r *temporalRegistry) UpdatePosition(ctx context.Context, t tenant.Model, characterId uint32, x int16, y int16) {
	existing, _ := r.reg.Get(ctx, t, characterId)
	_ = r.reg.Put(ctx, t, characterId, temporalData{x: x, y: y, stance: existing.stance})
}

func (r *temporalRegistry) Update(ctx context.Context, t tenant.Model, characterId uint32, x int16, y int16, stance byte) {
	_ = r.reg.Put(ctx, t, characterId, temporalData{x: x, y: y, stance: stance})
}

func (r *temporalRegistry) UpdateStance(ctx context.Context, t tenant.Model, characterId uint32, stance byte) {
	existing, _ := r.reg.Get(ctx, t, characterId)
	_ = r.reg.Put(ctx, t, characterId, temporalData{x: existing.x, y: existing.y, stance: stance})
}

func (r *temporalRegistry) GetById(ctx context.Context, t tenant.Model, characterId uint32) temporalData {
	val, err := r.reg.Get(ctx, t, characterId)
	if err != nil {
		return temporalData{}
	}
	return val
}

func (r *temporalRegistry) Shutdown() {
	r.reg.Shutdown()
}

var once sync.Once
var instance *temporalRegistry

func InitTemporalRegistry(rc *goredis.Client) {
	once.Do(func() {
		instance = &temporalRegistry{
			reg: atlas.NewTenantCoalescedRegistry[uint32, temporalData](
				rc, "character-temporal",
				func(k uint32) string { return strconv.FormatUint(uint64(k), 10) },
				100*time.Millisecond, 100*time.Millisecond,
			),
		}
	})
}

func GetTemporalRegistry() *temporalRegistry {
	return instance
}
