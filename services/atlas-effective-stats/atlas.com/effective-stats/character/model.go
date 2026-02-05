package character

import (
	"atlas-effective-stats/stat"
	"math"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

// Model holds all stat bonuses and computed effective stats for a character
type Model struct {
	tenant      tenant.Model
	ch          channel.Model
	characterId uint32

	// Base stats from character service
	baseStats stat.Base

	// Bonuses by source type
	bonuses []stat.Bonus

	// Cached computed totals
	computed    stat.Computed
	lastUpdated time.Time
	initialized bool
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}

func (m Model) WorldId() world.Id {
	return m.ch.WorldId()
}

func (m Model) ChannelId() channel.Id {
	return m.ch.Id()
}

func (m Model) Channel() channel.Model {
	return m.ch
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) BaseStats() stat.Base {
	return m.baseStats
}

func (m Model) Bonuses() []stat.Bonus {
	// Return defensive copy
	result := make([]stat.Bonus, len(m.bonuses))
	copy(result, m.bonuses)
	return result
}

func (m Model) Computed() stat.Computed {
	return m.computed
}

func (m Model) LastUpdated() time.Time {
	return m.lastUpdated
}

func (m Model) Initialized() bool {
	return m.initialized
}

// NewModel creates a new character effective stats model
func NewModel(t tenant.Model, ch channel.Model, characterId uint32) Model {
	return Model{
		tenant:      t,
		ch:          ch,
		characterId: characterId,
		bonuses:     make([]stat.Bonus, 0),
		initialized: false,
	}
}

// WithBaseStats returns a new model with updated base stats
func (m Model) WithBaseStats(base stat.Base) Model {
	return Model{
		tenant:      m.tenant,
		ch:          m.ch,
		characterId: m.characterId,
		baseStats:   base,
		bonuses:     m.bonuses,
		computed:    m.computed,
		lastUpdated: m.lastUpdated,
		initialized: m.initialized,
	}
}

// WithBonus returns a new model with the bonus added/updated
func (m Model) WithBonus(b stat.Bonus) Model {
	// Remove existing bonus with same source and stat type, then add new one
	newBonuses := make([]stat.Bonus, 0, len(m.bonuses)+1)
	for _, existing := range m.bonuses {
		if existing.Source() != b.Source() || existing.StatType() != b.StatType() {
			newBonuses = append(newBonuses, existing)
		}
	}
	newBonuses = append(newBonuses, b)

	return Model{
		tenant:      m.tenant,
		ch:          m.ch,
		characterId: m.characterId,
		baseStats:   m.baseStats,
		bonuses:     newBonuses,
		computed:    m.computed,
		lastUpdated: m.lastUpdated,
		initialized: m.initialized,
	}
}

// WithBonuses returns a new model with multiple bonuses added/updated
func (m Model) WithBonuses(bonuses []stat.Bonus) Model {
	result := m
	for _, b := range bonuses {
		result = result.WithBonus(b)
	}
	return result
}

// WithoutBonus returns a new model with the bonus removed
func (m Model) WithoutBonus(source string, statType stat.Type) Model {
	newBonuses := make([]stat.Bonus, 0, len(m.bonuses))
	for _, existing := range m.bonuses {
		if existing.Source() != source || existing.StatType() != statType {
			newBonuses = append(newBonuses, existing)
		}
	}

	return Model{
		tenant:      m.tenant,
		ch:          m.ch,
		characterId: m.characterId,
		baseStats:   m.baseStats,
		bonuses:     newBonuses,
		computed:    m.computed,
		lastUpdated: m.lastUpdated,
		initialized: m.initialized,
	}
}

// WithoutBonusesBySource removes all bonuses from a specific source
func (m Model) WithoutBonusesBySource(source string) Model {
	newBonuses := make([]stat.Bonus, 0, len(m.bonuses))
	for _, existing := range m.bonuses {
		if existing.Source() != source {
			newBonuses = append(newBonuses, existing)
		}
	}

	return Model{
		tenant:      m.tenant,
		ch:          m.ch,
		characterId: m.characterId,
		baseStats:   m.baseStats,
		bonuses:     newBonuses,
		computed:    m.computed,
		lastUpdated: m.lastUpdated,
		initialized: m.initialized,
	}
}

// WithComputed returns a new model with updated computed stats
func (m Model) WithComputed(computed stat.Computed) Model {
	return Model{
		tenant:      m.tenant,
		ch:          m.ch,
		characterId: m.characterId,
		baseStats:   m.baseStats,
		bonuses:     m.bonuses,
		computed:    computed,
		lastUpdated: time.Now(),
		initialized: m.initialized,
	}
}

// WithInitialized returns a new model marked as initialized
func (m Model) WithInitialized() Model {
	return Model{
		tenant:      m.tenant,
		ch:          m.ch,
		characterId: m.characterId,
		baseStats:   m.baseStats,
		bonuses:     m.bonuses,
		computed:    m.computed,
		lastUpdated: m.lastUpdated,
		initialized: true,
	}
}

// ComputeEffectiveStats calculates effective stats from base stats and bonuses
// Formula: effective = floor((base + flat_bonuses) * (1.0 + multiplier_bonuses))
func (m Model) ComputeEffectiveStats() stat.Computed {
	// Initialize with base stats
	baseValues := map[stat.Type]int32{
		stat.TypeStrength:      int32(m.baseStats.Strength()),
		stat.TypeDexterity:     int32(m.baseStats.Dexterity()),
		stat.TypeLuck:          int32(m.baseStats.Luck()),
		stat.TypeIntelligence:  int32(m.baseStats.Intelligence()),
		stat.TypeMaxHP:         int32(m.baseStats.MaxHP()),
		stat.TypeMaxMP:         int32(m.baseStats.MaxMP()),
		stat.TypeWeaponAttack:  0,
		stat.TypeWeaponDefense: 0,
		stat.TypeMagicAttack:   0,
		stat.TypeMagicDefense:  0,
		stat.TypeAccuracy:      0,
		stat.TypeAvoidability:  0,
		stat.TypeSpeed:         0,
		stat.TypeJump:          0,
	}

	// Sum flat bonuses and multipliers for each stat type
	flatBonuses := make(map[stat.Type]int32)
	multipliers := make(map[stat.Type]float64)

	for _, statType := range stat.AllTypes() {
		flatBonuses[statType] = 0
		multipliers[statType] = 0.0
	}

	for _, b := range m.bonuses {
		flatBonuses[b.StatType()] += b.Amount()
		multipliers[b.StatType()] += b.Multiplier()
	}

	// Calculate effective values
	// effective = floor((base + flat) * (1.0 + multiplier))
	computeEffective := func(statType stat.Type) uint32 {
		base := baseValues[statType]
		flat := flatBonuses[statType]
		mult := multipliers[statType]

		effective := float64(base+flat) * (1.0 + mult)
		if effective < 0 {
			return 0
		}
		return uint32(math.Floor(effective))
	}

	return stat.NewComputed(
		computeEffective(stat.TypeStrength),
		computeEffective(stat.TypeDexterity),
		computeEffective(stat.TypeLuck),
		computeEffective(stat.TypeIntelligence),
		computeEffective(stat.TypeMaxHP),
		computeEffective(stat.TypeMaxMP),
		computeEffective(stat.TypeWeaponAttack),
		computeEffective(stat.TypeWeaponDefense),
		computeEffective(stat.TypeMagicAttack),
		computeEffective(stat.TypeMagicDefense),
		computeEffective(stat.TypeAccuracy),
		computeEffective(stat.TypeAvoidability),
		computeEffective(stat.TypeSpeed),
		computeEffective(stat.TypeJump),
	)
}

// Recompute returns a new model with freshly computed effective stats
func (m Model) Recompute() Model {
	computed := m.ComputeEffectiveStats()
	return m.WithComputed(computed)
}
