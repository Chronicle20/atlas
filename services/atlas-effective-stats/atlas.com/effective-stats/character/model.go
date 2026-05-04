package character

import (
	"atlas-effective-stats/stat"
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// MaxHpMpCap is the per-character ceiling on effective MaxHp / MaxMp.
// The classic v83 client and the legacy serializers cap these stats at
// 30000; bonuses are still summed but the published effective value is
// clamped here so downstream services (atlas-character HP/MP clamps,
// stat-changed broadcasts, heal handlers) operate on a value the wire
// protocol can represent.
const MaxHpMpCap uint32 = 30000

// Model holds all stat bonuses and computed effective stats for a character
type Model struct {
	tenant      tenant.Model
	ch          channel.Model
	characterId uint32

	// Base stats from character service
	baseStats stat.Base

	// Bonuses by source. After this task lands and the equipment migration in
	// Task 17 / 18 completes, this slice holds only buff:* and passive:*
	// entries; equipment lives in the equipped map below.
	bonuses []stat.Bonus

	// Wearer profile (level + jobId) — inputs to reqLevel / reqJob.
	wearer WearerProfile

	// Equipped-asset snapshot map keyed by assetId. Source of truth for
	// equipment bonuses.
	equipped map[uint32]EquippedAsset

	// Cached set of qualifying asset ids from the most recent
	// RecomputeWith. Read by Bonuses() to avoid re-running the iterator.
	// Always treated as read-only after construction.
	qualifiedSnapshot map[uint32]bool

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

func (m Model) Wearer() WearerProfile {
	return m.wearer
}

// Equipped returns a copy of the equipped-asset snapshot map.
func (m Model) Equipped() map[uint32]EquippedAsset {
	out := make(map[uint32]EquippedAsset, len(m.equipped))
	for k, v := range m.equipped {
		out[k] = v
	}
	return out
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
		tenant:            t,
		ch:                ch,
		characterId:       characterId,
		bonuses:           make([]stat.Bonus, 0),
		equipped:          make(map[uint32]EquippedAsset),
		qualifiedSnapshot: make(map[uint32]bool),
		initialized:       false,
	}
}

// WithBaseStats returns a new model with updated base stats
func (m Model) WithBaseStats(base stat.Base) Model {
	out := m.shallowCopy()
	out.baseStats = base
	return out
}

// WithBonus returns a new model with the bonus added/updated
func (m Model) WithBonus(b stat.Bonus) Model {
	newBonuses := make([]stat.Bonus, 0, len(m.bonuses)+1)
	for _, existing := range m.bonuses {
		if existing.Source() != b.Source() || existing.StatType() != b.StatType() {
			newBonuses = append(newBonuses, existing)
		}
	}
	newBonuses = append(newBonuses, b)
	out := m.shallowCopy()
	out.bonuses = newBonuses
	return out
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
	out := m.shallowCopy()
	out.bonuses = newBonuses
	return out
}

// WithoutBonusesBySource removes all bonuses from a specific source
func (m Model) WithoutBonusesBySource(source string) Model {
	newBonuses := make([]stat.Bonus, 0, len(m.bonuses))
	for _, existing := range m.bonuses {
		if existing.Source() != source {
			newBonuses = append(newBonuses, existing)
		}
	}
	out := m.shallowCopy()
	out.bonuses = newBonuses
	return out
}

// WithComputed returns a new model with updated computed stats
func (m Model) WithComputed(computed stat.Computed) Model {
	out := m.shallowCopy()
	out.computed = computed
	out.lastUpdated = time.Now()
	return out
}

// WithInitialized returns a new model marked as initialized
func (m Model) WithInitialized() Model {
	out := m.shallowCopy()
	out.initialized = true
	return out
}

// WithWearer returns a new model with an updated wearer profile.
func (m Model) WithWearer(p WearerProfile) Model {
	out := m.shallowCopy()
	out.wearer = p
	return out
}

// WithEquippedAsset overwrites (or inserts) the snapshot keyed by asset id.
func (m Model) WithEquippedAsset(a EquippedAsset) Model {
	out := m.shallowCopy()
	out.equipped = copyEquipped(m.equipped)
	out.equipped[a.AssetId()] = a
	return out
}

// WithoutEquippedAsset removes the snapshot for the given asset id.
func (m Model) WithoutEquippedAsset(assetId uint32) Model {
	out := m.shallowCopy()
	out.equipped = copyEquipped(m.equipped)
	delete(out.equipped, assetId)
	return out
}

// withQualifiedSnapshot is package-private — only RecomputeWith should call it.
func (m Model) withQualifiedSnapshot(q map[uint32]bool) Model {
	out := m.shallowCopy()
	out.qualifiedSnapshot = q
	return out
}

func (m Model) shallowCopy() Model {
	return Model{
		tenant:            m.tenant,
		ch:                m.ch,
		characterId:       m.characterId,
		baseStats:         m.baseStats,
		bonuses:           m.bonuses,
		wearer:            m.wearer,
		equipped:          m.equipped,
		qualifiedSnapshot: m.qualifiedSnapshot,
		computed:          m.computed,
		lastUpdated:       m.lastUpdated,
		initialized:       m.initialized,
	}
}

func copyEquipped(src map[uint32]EquippedAsset) map[uint32]EquippedAsset {
	out := make(map[uint32]EquippedAsset, len(src)+1)
	for k, v := range src {
		out[k] = v
	}
	return out
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
		stat.TypeMaxHp:         int32(m.baseStats.MaxHp()),
		stat.TypeMaxMp:         int32(m.baseStats.MaxMp()),
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
	// MaxHp / MaxMp are clamped to MaxHpMpCap (30000) — see the
	// MaxHpMpCap doc comment for the rationale.
	computeEffective := func(statType stat.Type) uint32 {
		base := baseValues[statType]
		flat := flatBonuses[statType]
		mult := multipliers[statType]

		effective := float64(base+flat) * (1.0 + mult)
		if effective < 0 {
			return 0
		}
		v := uint32(math.Floor(effective))
		if statType == stat.TypeMaxHp || statType == stat.TypeMaxMp {
			if v > MaxHpMpCap {
				v = MaxHpMpCap
			}
		}
		return v
	}

	return stat.NewComputed(
		computeEffective(stat.TypeStrength),
		computeEffective(stat.TypeDexterity),
		computeEffective(stat.TypeLuck),
		computeEffective(stat.TypeIntelligence),
		computeEffective(stat.TypeMaxHp),
		computeEffective(stat.TypeMaxMp),
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

type wearerJSON struct {
	Level byte   `json:"level"`
	JobId job.Id `json:"jobId"`
}

type equippedAssetJSON struct {
	AssetId    uint32       `json:"assetId"`
	TemplateId uint32       `json:"templateId"`
	Bonuses    []stat.Bonus `json:"bonuses"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	eq := make(map[string]equippedAssetJSON, len(m.equipped))
	for id, snap := range m.equipped {
		eq[strconv.FormatUint(uint64(id), 10)] = equippedAssetJSON{
			AssetId:    snap.assetId,
			TemplateId: snap.templateId,
			Bonuses:    append([]stat.Bonus(nil), snap.bonuses...),
		}
	}
	qs := make(map[string]bool, len(m.qualifiedSnapshot))
	for id, ok := range m.qualifiedSnapshot {
		qs[strconv.FormatUint(uint64(id), 10)] = ok
	}
	return json.Marshal(struct {
		WorldId           world.Id                     `json:"worldId"`
		ChannelId         channel.Id                   `json:"channelId"`
		CharacterId       uint32                       `json:"characterId"`
		BaseStats         stat.Base                    `json:"baseStats"`
		Bonuses           []stat.Bonus                 `json:"bonuses"`
		Wearer            wearerJSON                   `json:"wearer"`
		Equipped          map[string]equippedAssetJSON `json:"equipped"`
		QualifiedSnapshot map[string]bool              `json:"qualifiedSnapshot"`
		Computed          stat.Computed                `json:"computed"`
		LastUpdated       time.Time                    `json:"lastUpdated"`
		Initialized       bool                         `json:"initialized"`
	}{
		WorldId:           m.ch.WorldId(),
		ChannelId:         m.ch.Id(),
		CharacterId:       m.characterId,
		BaseStats:         m.baseStats,
		Bonuses:           m.bonuses,
		Wearer:            wearerJSON{Level: m.wearer.level, JobId: m.wearer.jobId},
		Equipped:          eq,
		QualifiedSnapshot: qs,
		Computed:          m.computed,
		LastUpdated:       m.lastUpdated,
		Initialized:       m.initialized,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux struct {
		WorldId           world.Id                     `json:"worldId"`
		ChannelId         channel.Id                   `json:"channelId"`
		CharacterId       uint32                       `json:"characterId"`
		BaseStats         stat.Base                    `json:"baseStats"`
		Bonuses           []stat.Bonus                 `json:"bonuses"`
		Wearer            wearerJSON                   `json:"wearer"`
		Equipped          map[string]equippedAssetJSON `json:"equipped"`
		QualifiedSnapshot map[string]bool              `json:"qualifiedSnapshot"`
		Computed          stat.Computed                `json:"computed"`
		LastUpdated       time.Time                    `json:"lastUpdated"`
		Initialized       bool                         `json:"initialized"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.ch = channel.NewModel(aux.WorldId, aux.ChannelId)
	m.characterId = aux.CharacterId
	m.baseStats = aux.BaseStats
	if aux.Bonuses == nil {
		m.bonuses = make([]stat.Bonus, 0)
	} else {
		m.bonuses = aux.Bonuses
	}
	m.wearer = WearerProfile{level: aux.Wearer.Level, jobId: aux.Wearer.JobId}
	m.equipped = make(map[uint32]EquippedAsset, len(aux.Equipped))
	for _, snap := range aux.Equipped {
		m.equipped[snap.AssetId] = EquippedAsset{
			assetId:    snap.AssetId,
			templateId: snap.TemplateId,
			bonuses:    append([]stat.Bonus(nil), snap.Bonuses...),
		}
	}
	m.qualifiedSnapshot = make(map[uint32]bool, len(aux.QualifiedSnapshot))
	for k, v := range aux.QualifiedSnapshot {
		id, err := strconv.ParseUint(k, 10, 32)
		if err != nil {
			continue
		}
		m.qualifiedSnapshot[uint32(id)] = v
	}
	m.computed = aux.Computed
	m.lastUpdated = aux.LastUpdated
	m.initialized = aux.Initialized
	return nil
}
