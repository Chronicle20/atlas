package character

import (
	"atlas-rates/rate"
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

// Registry is the Redis-backed cache for character rates
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
		characters: atlas.NewTenantRegistry[uint32, Model](client, "rates", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
	}
}

// Get retrieves a character's rate model
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

// GetOrCreate retrieves a character's rate model, creating one if it doesn't exist
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

// Update replaces a character's rate model
func (r *Registry) Update(ctx context.Context, m Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.characters.Put(ctx, t, m.characterId, m)
}

// AddFactor adds or updates a rate factor for a character
func (r *Registry) AddFactor(ctx context.Context, ch channel.Model, characterId uint32, f rate.Factor) Model {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		m = NewModel(t, ch, characterId)
	}
	m = m.WithFactor(f)
	_ = r.characters.Put(ctx, t, characterId, m)
	return m
}

// RemoveFactor removes a specific rate factor for a character
func (r *Registry) RemoveFactor(ctx context.Context, characterId uint32, source string, rateType rate.Type) (Model, error) {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, ErrNotFound
	}
	m = m.WithoutFactor(source, rateType)
	_ = r.characters.Put(ctx, t, characterId, m)
	return m, nil
}

// RemoveFactorsBySource removes all factors from a specific source for a character
func (r *Registry) RemoveFactorsBySource(ctx context.Context, characterId uint32, source string) (Model, error) {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, ErrNotFound
	}
	m = m.WithoutFactorsBySource(source)
	_ = r.characters.Put(ctx, t, characterId, m)
	return m, nil
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
		if m.worldId == worldId {
			result = append(result, m)
		}
	}
	return result
}

// UpdateWorldRate updates the world rate factor for all characters in that world
func (r *Registry) UpdateWorldRate(ctx context.Context, worldId world.Id, rateType rate.Type, multiplier float64) {
	t := tenant.MustFromContext(ctx)
	models, err := r.characters.GetAllValues(ctx, t)
	if err != nil {
		return
	}

	source := "world"
	f := rate.NewFactor(source, rateType, multiplier)

	for _, m := range models {
		if m.worldId == worldId {
			m = m.WithFactor(f)
			_ = r.characters.Put(ctx, t, m.characterId, m)
		}
	}
}

// Delete removes a character from the registry
func (r *Registry) Delete(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.characters.Remove(ctx, t, characterId)
}
