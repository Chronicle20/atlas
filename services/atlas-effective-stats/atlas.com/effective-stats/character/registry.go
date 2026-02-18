package character

import (
	"atlas-effective-stats/stat"
	"context"
	"errors"
	"strconv"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("not found")

// Registry is the Redis-backed cache for character effective stats
type Registry struct {
	characters *atlas.TenantRegistry[uint32, Model]
}

var registry *Registry

// GetRegistry returns the singleton registry instance
func GetRegistry() *Registry {
	return registry
}

// InitRegistry initializes the registry with a Redis client
func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		characters: atlas.NewTenantRegistry[uint32, Model](client, "effective-stats", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
	}
}

// Get retrieves a character's effective stats model
func (r *Registry) Get(ctx context.Context, characterId uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		if errors.Is(err, atlas.ErrNotFound) {
			return Model{}, ErrNotFound
		}
		return Model{}, err
	}
	return m, nil
}

// GetOrCreate retrieves a character's effective stats model, creating one if it doesn't exist
func (r *Registry) GetOrCreate(ctx context.Context, ch channel.Model, characterId uint32) Model {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err == nil {
		return m
	}
	m = NewModel(t, ch, characterId)
	_ = r.characters.Put(ctx, t, characterId, m)
	return m
}

// Update replaces a character's effective stats model
func (r *Registry) Update(ctx context.Context, m Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.characters.Put(ctx, t, m.characterId, m)
}

// AddBonus adds or updates a stat bonus for a character
func (r *Registry) AddBonus(ctx context.Context, ch channel.Model, characterId uint32, b stat.Bonus) Model {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		m = NewModel(t, ch, characterId)
	}
	m = m.WithBonus(b).Recompute()
	_ = r.characters.Put(ctx, t, characterId, m)
	return m
}

// AddBonuses adds or updates multiple stat bonuses for a character
func (r *Registry) AddBonuses(ctx context.Context, ch channel.Model, characterId uint32, bonuses []stat.Bonus) Model {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		m = NewModel(t, ch, characterId)
	}
	m = m.WithBonuses(bonuses).Recompute()
	_ = r.characters.Put(ctx, t, characterId, m)
	return m
}

// RemoveBonus removes a specific stat bonus for a character
func (r *Registry) RemoveBonus(ctx context.Context, characterId uint32, source string, statType stat.Type) (Model, error) {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, ErrNotFound
	}
	m = m.WithoutBonus(source, statType).Recompute()
	_ = r.characters.Put(ctx, t, characterId, m)
	return m, nil
}

// RemoveBonusesBySource removes all bonuses from a specific source for a character
func (r *Registry) RemoveBonusesBySource(ctx context.Context, characterId uint32, source string) (Model, error) {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, ErrNotFound
	}
	m = m.WithoutBonusesBySource(source).Recompute()
	_ = r.characters.Put(ctx, t, characterId, m)
	return m, nil
}

// SetBaseStats sets the base stats for a character and recomputes effective stats
func (r *Registry) SetBaseStats(ctx context.Context, ch channel.Model, characterId uint32, base stat.Base) Model {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		m = NewModel(t, ch, characterId)
	}
	m = m.WithBaseStats(base).Recompute()
	_ = r.characters.Put(ctx, t, characterId, m)
	return m
}

// MarkInitialized marks a character as initialized
func (r *Registry) MarkInitialized(ctx context.Context, characterId uint32) error {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		return ErrNotFound
	}
	m = m.WithInitialized()
	_ = r.characters.Put(ctx, t, characterId, m)
	return nil
}

// IsInitialized checks if a character has been initialized
func (r *Registry) IsInitialized(ctx context.Context, characterId uint32) bool {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		return false
	}
	return m.Initialized()
}

// GetAll returns all characters for a tenant
func (r *Registry) GetAll(ctx context.Context) []Model {
	t := tenant.MustFromContext(ctx)
	models, err := r.characters.GetAllValues(ctx, t)
	if err != nil {
		return nil
	}
	return models
}

// GetAllForWorld returns all characters in a specific world
func (r *Registry) GetAllForWorld(ctx context.Context, worldId world.Id) []Model {
	t := tenant.MustFromContext(ctx)
	models, err := r.characters.GetAllValues(ctx, t)
	if err != nil {
		return nil
	}
	result := make([]Model, 0)
	for _, m := range models {
		if m.WorldId() == worldId {
			result = append(result, m)
		}
	}
	return result
}

// Delete removes a character from the registry
func (r *Registry) Delete(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.characters.Remove(ctx, t, characterId)
}
