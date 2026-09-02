package npc

import (
	"atlas-data/xml"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newReaderTestTenant(t *testing.T) tenant.Model {
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tn
}

// TestNpcReadImitateFlag drives Read against hand-built xml.Node fixtures
// (no XML parsing needed; xml.Node is a plain struct) to verify that the
// info/imitate flag is parsed into RestModel.Imitate.
func TestNpcReadImitateFlag(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), newReaderTestTenant(t))

	t.Run("imitate 1", func(t *testing.T) {
		n := xml.Node{
			Name: "2005.img",
			ChildNodes: []xml.Node{
				{Name: "info", IntegerNodes: []xml.IntegerNode{{Name: "imitate", Value: "1"}}},
			},
		}
		np := model.FixedProvider(n)
		rm, err := Read(l)(ctx)(np)()
		require.NoError(t, err)
		assert.True(t, rm.Imitate)
	})

	t.Run("imitate 0", func(t *testing.T) {
		n := xml.Node{
			Name: "2005.img",
			ChildNodes: []xml.Node{
				{Name: "info", IntegerNodes: []xml.IntegerNode{{Name: "imitate", Value: "0"}}},
			},
		}
		np := model.FixedProvider(n)
		rm, err := Read(l)(ctx)(np)()
		require.NoError(t, err)
		assert.False(t, rm.Imitate)
	})

	t.Run("absent", func(t *testing.T) {
		n := xml.Node{
			Name: "2005.img",
			ChildNodes: []xml.Node{
				{Name: "info", IntegerNodes: []xml.IntegerNode{{Name: "trunkPut", Value: "1"}}},
			},
		}
		np := model.FixedProvider(n)
		rm, err := Read(l)(ctx)(np)()
		require.NoError(t, err)
		assert.False(t, rm.Imitate)
	})

	t.Run("no info section", func(t *testing.T) {
		n := xml.Node{
			Name:       "2005.img",
			ChildNodes: []xml.Node{},
		}
		np := model.FixedProvider(n)
		rm, err := Read(l)(ctx)(np)()
		require.NoError(t, err)
		assert.False(t, rm.Imitate)
	})
}
