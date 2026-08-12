package mist

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Protection is a live protection (Smokescreen) mist as this channel knows
// it: enough to answer "is this character standing in one, and does it belong
// to them or their party?" on the damage path.
//
// The channel keeps its own copy rather than querying atlas-maps because the
// alternative is a synchronous REST round trip on the most latency-sensitive
// path in the service. The cost is a restart gap: the mist consumer starts at
// kafka.LastOffset, so a channel that restarts mid-mist never learns about
// mists created before it came up. That is not a regression -- the same
// restart already loses the AffectedAreaCreated broadcast, so those mists are
// invisible to every client on that channel too -- and it is bounded by the
// longest Smokescreen lifetime (60s at level 30).
type Protection struct {
	id        uuid.UUID
	f         field.Model
	ownerId   uint32
	minX      int16
	minY      int16
	maxX      int16
	maxY      int16
	expiresAt time.Time
}

// Id returns the mist id this protection was created from.
func (p Protection) Id() uuid.UUID { return p.id }

// Field returns the field the protection covers.
func (p Protection) Field() field.Model { return p.f }

// OwnerId returns the casting character. The client's smoke lookup accepts an
// area only if its dwOwnerID is the local character or one of their online
// party members (CAffectedAreaPool::IsSmokeAreaByPoint, v95 @0x434f40), so
// the owner is what the party check is evaluated against.
func (p Protection) OwnerId() uint32 { return p.ownerId }

// ExpiresAt returns the absolute expiry.
func (p Protection) ExpiresAt() time.Time { return p.expiresAt }

// Expired reports whether the protection is past its lifetime as of now.
func (p Protection) Expired(now time.Time) bool { return now.After(p.expiresAt) }

// Contains reports whether the world coordinates fall inside the protection's
// axis-aligned bounding box. Edges are INCLUSIVE, matching atlas-maps'
// Mist.Contains and the atlas-monsters in-rect endpoint -- the rect test
// exists on both sides and the two conventions must not drift.
func (p Protection) Contains(x, y int16) bool {
	return x >= p.minX && x <= p.maxX && y >= p.minY && y <= p.maxY
}

// ProtectionBuilder constructs a Protection via fluent setters.
type ProtectionBuilder struct {
	p Protection
}

// NewProtectionBuilder starts a Protection anchored to a mist id and field.
func NewProtectionBuilder(id uuid.UUID, f field.Model) *ProtectionBuilder {
	return &ProtectionBuilder{p: Protection{id: id, f: f}}
}

func (b *ProtectionBuilder) SetOwnerId(v uint32) *ProtectionBuilder {
	b.p.ownerId = v
	return b
}

// SetRect sets the ABSOLUTE world-coordinate bounding box (origin already
// added to the lt/rb offsets).
func (b *ProtectionBuilder) SetRect(minX, minY, maxX, maxY int16) *ProtectionBuilder {
	b.p.minX, b.p.minY, b.p.maxX, b.p.maxY = minX, minY, maxX, maxY
	return b
}

func (b *ProtectionBuilder) SetExpiresAt(v time.Time) *ProtectionBuilder {
	b.p.expiresAt = v
	return b
}

func (b *ProtectionBuilder) Build() Protection { return b.p }

// ProtectionRegistry is a tenant-scoped, in-memory index of live protection
// mists. Safe for concurrent use: the damage path reads it on every hit.
type ProtectionRegistry struct {
	mu        sync.RWMutex
	perTenant map[string]map[uuid.UUID]Protection
}

var (
	protectionRegistryOnce sync.Once
	protectionRegistry     *ProtectionRegistry
)

// GetProtectionRegistry returns the process-wide singleton, lazily built.
func GetProtectionRegistry() *ProtectionRegistry {
	protectionRegistryOnce.Do(func() {
		protectionRegistry = &ProtectionRegistry{perTenant: map[string]map[uuid.UUID]Protection{}}
	})
	return protectionRegistry
}

// NewTestProtectionRegistry returns a fresh, isolated registry so tests do
// not leak state through the singleton. Not used in production paths.
func NewTestProtectionRegistry() *ProtectionRegistry {
	return &ProtectionRegistry{perTenant: map[string]map[uuid.UUID]Protection{}}
}

// Add inserts p and lazily prunes entries that have already expired, so a
// dropped MIST_DESTROYED cannot accumulate stale rectangles.
func (r *ProtectionRegistry) Add(t tenant.Model, p Protection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := t.Id().String()
	b, ok := r.perTenant[key]
	if !ok {
		b = map[uuid.UUID]Protection{}
		r.perTenant[key] = b
	}
	now := time.Now()
	for id, existing := range b {
		if existing.Expired(now) {
			delete(b, id)
		}
	}
	b[p.Id()] = p
}

// Remove drops the protection with the given mist id. No-op if absent.
func (r *ProtectionRegistry) Remove(t tenant.Model, id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := t.Id().String()
	b, ok := r.perTenant[key]
	if !ok {
		return
	}
	delete(b, id)
	if len(b) == 0 {
		delete(r.perTenant, key)
	}
}

// Covering returns every live protection on f that contains (x, y). Expired
// entries are treated as absent regardless of whether they have been pruned:
// a missed MIST_DESTROYED must degrade to "no protection", never to a
// permanently invulnerable rectangle.
func (r *ProtectionRegistry) Covering(t tenant.Model, f field.Model, x, y int16, now time.Time) []Protection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.perTenant[t.Id().String()]
	if !ok {
		return nil
	}
	var out []Protection
	for _, p := range b {
		if p.Expired(now) || !p.Field().Equals(f) || !p.Contains(x, y) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// Len reports how many protections the tenant currently holds. Exposed for
// tests asserting the pruning behaviour.
func (r *ProtectionRegistry) Len(t tenant.Model) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.perTenant[t.Id().String()])
}
