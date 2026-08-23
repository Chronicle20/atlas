package parcel

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// newTestProcessorImpl mirrors newTestProcessor but returns the concrete
// *ProcessorImpl, since AcceptCustody is additive to the Processor interface
// (see kafka/consumer/custody/consumer.go's processor() helper).
func newTestProcessorImpl(t *testing.T, db *gorm.DB, tenantId uuid.UUID, now time.Time) *ProcessorImpl {
	t.Helper()
	p := newTestProcessor(t, db, tenantId, now)
	impl, ok := p.(*ProcessorImpl)
	require.True(t, ok)
	return impl
}

// TestProcessorAcceptCustody asserts AcceptCustody persists the caller-supplied
// ItemType onto the created row when HasItem is true, and leaves it zero for
// a meso-only accept.
func TestProcessorAcceptCustody(t *testing.T) {
	t.Run("item parcel persists ItemType", func(t *testing.T) {
		db, tid := newProcessorTestDB(t)
		p := newTestProcessorImpl(t, db, tid, fixedClock)

		parcelId := uuid.New()
		m, err := p.AcceptCustody(AcceptParams{
			ParcelId:           parcelId,
			CharacterId:        1001,
			WorldId:            world.Id(0),
			SenderAccountId:    10,
			SenderName:         "Sender",
			RecipientId:        2002,
			RecipientAccountId: 20,
			RecipientName:      "Bob",
			ReceivableAt:       fixedClock,
			ExpiresAt:          fixedClock.Add(ExpiryWindow),
			HasItem:            true,
			ItemType:           2,
			TemplateId:         2000004,
			Quantity:           5,
		})
		require.NoError(t, err)
		assert.Equal(t, byte(2), m.ItemType())

		reread, err := p.GetById(parcelId)
		require.NoError(t, err)
		assert.Equal(t, byte(2), reread.ItemType())
	})

	t.Run("meso only leaves ItemType zero", func(t *testing.T) {
		db, tid := newProcessorTestDB(t)
		p := newTestProcessorImpl(t, db, tid, fixedClock)

		parcelId := uuid.New()
		m, err := p.AcceptCustody(AcceptParams{
			ParcelId:           parcelId,
			CharacterId:        1001,
			WorldId:            world.Id(0),
			SenderAccountId:    10,
			SenderName:         "Sender",
			RecipientId:        2002,
			RecipientAccountId: 20,
			RecipientName:      "Bob",
			ReceivableAt:       fixedClock,
			ExpiresAt:          fixedClock.Add(ExpiryWindow),
			HasItem:            false,
			MesoAmount:         500,
		})
		require.NoError(t, err)
		assert.Zero(t, m.ItemType())
	})
}
