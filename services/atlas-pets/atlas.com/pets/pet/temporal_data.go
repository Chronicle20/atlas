package pet

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

type TemporalData struct {
	x      int16
	y      int16
	stance byte
	fh     int16
}

func (d TemporalData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		X      int16 `json:"x"`
		Y      int16 `json:"y"`
		Stance byte  `json:"stance"`
		FH     int16 `json:"fh"`
	}{X: d.x, Y: d.y, Stance: d.stance, FH: d.fh})
}

func (d *TemporalData) UnmarshalJSON(data []byte) error {
	var raw struct {
		X      int16 `json:"x"`
		Y      int16 `json:"y"`
		Stance byte  `json:"stance"`
		FH     int16 `json:"fh"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.x = raw.X
	d.y = raw.Y
	d.stance = raw.Stance
	d.fh = raw.FH
	return nil
}

func (d *TemporalData) X() int16 {
	return d.x
}

func (d *TemporalData) Y() int16 {
	return d.y
}

func (d *TemporalData) Stance() byte {
	return d.stance
}

func (d *TemporalData) FH() int16 {
	return d.fh
}

func NewTemporalData() *TemporalData {
	return &TemporalData{fh: 1}
}

type TemporalRegistry interface {
	UpdatePosition(ctx context.Context, t tenant.Model, petId uint32, x int16, y int16, fh int16)
	Update(ctx context.Context, t tenant.Model, petId uint32, x int16, y int16, stance byte, fh int16)
	UpdateStance(ctx context.Context, t tenant.Model, petId uint32, stance byte)
	GetById(ctx context.Context, t tenant.Model, petId uint32) *TemporalData
	Remove(ctx context.Context, t tenant.Model, petId uint32)
}

type temporalRegistryImpl struct {
	reg *atlas.TenantCoalescedRegistry[uint32, TemporalData]
}

func (r *temporalRegistryImpl) UpdatePosition(ctx context.Context, t tenant.Model, petId uint32, x int16, y int16, fh int16) {
	existing, _ := r.reg.Get(ctx, t, petId)
	_ = r.reg.Put(ctx, t, petId, TemporalData{x: x, y: y, stance: existing.stance, fh: fh})
}

func (r *temporalRegistryImpl) Update(ctx context.Context, t tenant.Model, petId uint32, x int16, y int16, stance byte, fh int16) {
	_ = r.reg.Put(ctx, t, petId, TemporalData{x: x, y: y, stance: stance, fh: fh})
}

func (r *temporalRegistryImpl) UpdateStance(ctx context.Context, t tenant.Model, petId uint32, stance byte) {
	existing, _ := r.reg.Get(ctx, t, petId)
	_ = r.reg.Put(ctx, t, petId, TemporalData{x: existing.x, y: existing.y, stance: stance, fh: existing.fh})
}

func (r *temporalRegistryImpl) GetById(ctx context.Context, t tenant.Model, petId uint32) *TemporalData {
	val, err := r.reg.Get(ctx, t, petId)
	if err != nil {
		return NewTemporalData()
	}
	return &val
}

func (r *temporalRegistryImpl) Remove(ctx context.Context, t tenant.Model, petId uint32) {
	_ = r.reg.Remove(ctx, t, petId)
}

func (r *temporalRegistryImpl) Shutdown() {
	r.reg.Shutdown()
}

var once sync.Once
var instance *temporalRegistryImpl

func InitTemporalRegistry(rc *goredis.Client) {
	once.Do(func() {
		instance = &temporalRegistryImpl{
			reg: atlas.NewTenantCoalescedRegistry[uint32, TemporalData](
				rc, "pet-temporal",
				func(k uint32) string { return strconv.FormatUint(uint64(k), 10) },
				100*time.Millisecond, 100*time.Millisecond,
			),
		}
	})
}

func GetTemporalRegistry() TemporalRegistry {
	return instance
}
