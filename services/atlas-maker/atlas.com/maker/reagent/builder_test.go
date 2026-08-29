package reagent_test

import (
	"atlas-maker/reagent"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// TestBuilderAcceptsEveryDerivedStat walks the package-level valid-name set so
// the seed of docs/tasks/task-285-maker-skill-crafting/reagent-derivation.md
// §3.2 can round-trip in full. randOption and randStat are in the set: they are
// the equip random-option / random-stat variance keys the client stores in the
// same 15-field block, not additive equip stats.
func TestBuilderAcceptsEveryDerivedStat(t *testing.T) {
	tenantId := uuid.New()
	for _, stat := range reagent.ValidStats {
		t.Run(stat, func(t *testing.T) {
			m, err := reagent.NewBuilder(tenantId, item.Id(4250000)).
				SetStat(stat).
				SetValue(1).
				Build()
			require.NoError(t, err)
			assert.Equal(t, stat, m.Stat())
		})
	}
}

func TestBuilderRejectsUnknownStat(t *testing.T) {
	tests := []string{
		"",
		"incWATK",
		"incpad",   // the archive spelling is case-sensitive
		"incPAD ",  // trailing whitespace is not the archive spelling
		"strength", // not a client field name at all
	}
	for _, stat := range tests {
		t.Run(stat, func(t *testing.T) {
			_, err := reagent.NewBuilder(uuid.New(), item.Id(4250000)).
				SetStat(stat).
				SetValue(1).
				Build()
			assert.Error(t, err, "stat %q is outside the derived set and must be rejected", stat)
		})
	}
}

func TestBuilderRejectsMissingIdentity(t *testing.T) {
	t.Run("NilTenant", func(t *testing.T) {
		_, err := reagent.NewBuilder(uuid.Nil, item.Id(4250000)).SetStat("incPAD").SetValue(1).Build()
		assert.Error(t, err)
	})
	t.Run("ZeroItemId", func(t *testing.T) {
		_, err := reagent.NewBuilder(uuid.New(), item.Id(0)).SetStat("incPAD").SetValue(1).Build()
		assert.Error(t, err)
	})
}

func TestBuilderRoundTrip(t *testing.T) {
	tenantId := uuid.New()

	// Row 39 of the derived table: 0425.img/04251202/info/incReqLevel = -3.
	// The negative value is the reason the model's width is signed.
	m, err := reagent.NewBuilder(tenantId, item.Id(4251202)).
		SetStat("incReqLevel").
		SetValue(-3).
		Build()
	require.NoError(t, err)

	assert.Equal(t, tenantId, m.TenantId())
	assert.Equal(t, item.Id(4251202), m.ReagentItemId())
	assert.Equal(t, "incReqLevel", m.Stat())
	assert.Equal(t, int16(-3), m.Value())
}

// TestBuilderKeepsTheFullItemId guards the derivation's finding 1: the gem key
// is the full item id (4250000 .. 4251402), never a truncated byte.
func TestBuilderKeepsTheFullItemId(t *testing.T) {
	m, err := reagent.NewBuilder(uuid.New(), item.Id(4251402)).
		SetStat("randStat").
		SetValue(5).
		Build()
	require.NoError(t, err)
	assert.Equal(t, item.Id(4251402), m.ReagentItemId())
}
