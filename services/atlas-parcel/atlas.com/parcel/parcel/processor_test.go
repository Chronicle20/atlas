package parcel

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// fixedClock — T = 2026-01-02T00:00:00Z, the reference instant every
// TestProcessor* case below is written against.
var fixedClock = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

func newProcessorTestDB(t *testing.T) (*gorm.DB, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	// The "missing" TestProcessorReceive subtest deliberately provokes a
	// "record not found" GORM read, which the library's default logger
	// prints to stdout unless silenced.
	db.Logger = logger.Default.LogMode(logger.Silent)
	return db, uuid.New()
}

func newTestProcessor(t *testing.T, db *gorm.DB, tenantId uuid.UUID, now time.Time) Processor {
	t.Helper()
	l, _ := test.NewNullLogger()
	ctx := databasetest.TenantContext(tenantId)
	p := NewProcessor(l, ctx, db).(*ProcessorImpl)
	return p.withClock(func() time.Time { return now })
}

// seedParcel inserts a pending-by-default parcel via NewBuilder + Create,
// scoped to tenantId, and returns its id.
func seedParcel(t *testing.T, db *gorm.DB, tenantId uuid.UUID, mutate func(b *Builder)) uuid.UUID {
	t.Helper()
	id := uuid.New()
	b := NewBuilder().
		SetId(id).
		SetWorldId(0).
		SetSenderId(200).
		SetSenderAccountId(1).
		SetSenderName("Sender").
		SetRecipientId(100).
		SetRecipientAccountId(2).
		SetStatus(StatusPending).
		SetCreatedAt(fixedClock).
		SetReceivableAt(fixedClock).
		SetExpiresAt(fixedClock.Add(ExpiryWindow))
	if mutate != nil {
		mutate(b)
	}
	m, err := b.Build()
	require.NoError(t, err)

	ctx := databasetest.TenantContext(tenantId)
	_, err = Create(db.WithContext(ctx))(m)
	require.NoError(t, err)
	return id
}

// TestProcessorReceive is table-driven over run: each row exercises a
// distinct pre-flight/state combination for Processor.Receive with a
// distinct assertion set (success shape, sentinel error, or a post-call
// re-read), so the scenario itself is carried as a closure rather than
// shared data fields.
func TestProcessorReceive(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{name: "happy path", run: func(t *testing.T) {
			db, tid := newProcessorTestDB(t)
			id := seedParcel(t, db, tid, func(b *Builder) {
				b.SetReceivableAt(fixedClock.Add(-time.Hour))
			})
			p := newTestProcessor(t, db, tid, fixedClock)

			m, err := p.Receive(id, 100)
			require.NoError(t, err)
			assert.Equal(t, StatusReceived, m.Status())
			require.NotNil(t, m.ResolvedAt())
			assert.True(t, m.ResolvedAt().Equal(fixedClock))
		}},
		{name: "wrong recipient", run: func(t *testing.T) {
			db, tid := newProcessorTestDB(t)
			id := seedParcel(t, db, tid, nil)
			p := newTestProcessor(t, db, tid, fixedClock)

			_, err := p.Receive(id, 999)
			assert.ErrorIs(t, err, ErrNotRecipient)

			m, gerr := p.GetById(id)
			require.NoError(t, gerr)
			assert.Equal(t, StatusPending, m.Status())
		}},
		{name: "not yet receivable", run: func(t *testing.T) {
			db, tid := newProcessorTestDB(t)
			id := seedParcel(t, db, tid, func(b *Builder) {
				b.SetReceivableAt(fixedClock.Add(time.Hour))
			})
			p := newTestProcessor(t, db, tid, fixedClock)

			_, err := p.Receive(id, 100)
			assert.ErrorIs(t, err, ErrNotYetReceivable)

			m, gerr := p.GetById(id)
			require.NoError(t, gerr)
			assert.Equal(t, StatusPending, m.Status())
		}},
		{name: "already received", run: func(t *testing.T) {
			db, tid := newProcessorTestDB(t)
			id := seedParcel(t, db, tid, func(b *Builder) {
				b.SetStatus(StatusReceived).SetReceivableAt(fixedClock.Add(-time.Hour))
			})
			p := newTestProcessor(t, db, tid, fixedClock)

			_, err := p.Receive(id, 100)
			assert.ErrorIs(t, err, ErrNotPending)
		}},
		{name: "missing", run: func(t *testing.T) {
			db, tid := newProcessorTestDB(t)
			p := newTestProcessor(t, db, tid, fixedClock)

			_, err := p.Receive(uuid.New(), 100)
			assert.ErrorIs(t, err, ErrNotFound)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

// TestProcessorDiscard is table-driven over run: each row exercises a
// distinct pre-flight/state combination for Processor.Discard, mirroring
// TestProcessorReceive's shape.
func TestProcessorDiscard(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{name: "happy path", run: func(t *testing.T) {
			db, tid := newProcessorTestDB(t)
			id := seedParcel(t, db, tid, func(b *Builder) {
				b.SetReceivableAt(fixedClock.Add(-time.Hour))
			})
			p := newTestProcessor(t, db, tid, fixedClock)

			m, err := p.Discard(id, 100)
			require.NoError(t, err)
			assert.Equal(t, StatusDiscarded, m.Status())
			require.NotNil(t, m.ResolvedAt())
			assert.True(t, m.ResolvedAt().Equal(fixedClock))
		}},
		{name: "wrong recipient", run: func(t *testing.T) {
			db, tid := newProcessorTestDB(t)
			id := seedParcel(t, db, tid, nil)
			p := newTestProcessor(t, db, tid, fixedClock)

			_, err := p.Discard(id, 999)
			assert.ErrorIs(t, err, ErrNotRecipient)
		}},
		{name: "already discarded", run: func(t *testing.T) {
			db, tid := newProcessorTestDB(t)
			id := seedParcel(t, db, tid, func(b *Builder) {
				b.SetStatus(StatusDiscarded)
			})
			p := newTestProcessor(t, db, tid, fixedClock)

			_, err := p.Discard(id, 100)
			assert.ErrorIs(t, err, ErrNotPending)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

// TestProcessorHasInFlight is a true data table: every case shares the same
// shape (seed zero or more parcels, call HasInFlight(100), assert a bool),
// so seed is a setup closure but the expectation itself is plain data.
func TestProcessorHasInFlight(t *testing.T) {
	tests := []struct {
		name string
		seed func(t *testing.T, db *gorm.DB, tid uuid.UUID)
		want bool
	}{
		{
			name: "outbound pending",
			seed: func(t *testing.T, db *gorm.DB, tid uuid.UUID) {
				seedParcel(t, db, tid, func(b *Builder) {
					b.SetSenderId(100).SetRecipientId(200)
				})
			},
			want: true,
		},
		{
			name: "inbound receivable",
			seed: func(t *testing.T, db *gorm.DB, tid uuid.UUID) {
				seedParcel(t, db, tid, func(b *Builder) {
					b.SetRecipientId(100).SetReceivableAt(fixedClock.Add(-time.Hour))
				})
			},
			want: true,
		},
		{
			// Regression for the reviewer's BLOCKING 1: HasInFlight's inbound
			// check must not be hardcoded to world 0 — a parcel sent within
			// any world must still be found. Under the old
			// ReceivableByRecipient(characterId, world.Id(0), now) code this
			// case fails (false negative) because the seeded parcel's
			// WorldId is 7, not 0.
			name: "inbound receivable in a non-zero world",
			seed: func(t *testing.T, db *gorm.DB, tid uuid.UUID) {
				seedParcel(t, db, tid, func(b *Builder) {
					b.SetWorldId(world.Id(7)).SetRecipientId(100).SetReceivableAt(fixedClock.Add(-time.Hour))
				})
			},
			want: true,
		},
		{
			name: "inbound not yet receivable",
			seed: func(t *testing.T, db *gorm.DB, tid uuid.UUID) {
				seedParcel(t, db, tid, func(b *Builder) {
					b.SetRecipientId(100).SetReceivableAt(fixedClock.Add(time.Hour))
				})
			},
			want: false,
		},
		{
			name: "resolved only",
			seed: func(t *testing.T, db *gorm.DB, tid uuid.UUID) {
				seedParcel(t, db, tid, func(b *Builder) {
					b.SetSenderId(100).SetRecipientId(300).SetStatus(StatusReceived)
				})
				seedParcel(t, db, tid, func(b *Builder) {
					b.SetSenderId(300).SetRecipientId(100).SetStatus(StatusDiscarded).SetReceivableAt(fixedClock.Add(-time.Hour))
				})
			},
			want: false,
		},
		{
			name: "no parcels",
			seed: func(t *testing.T, db *gorm.DB, tid uuid.UUID) {},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, tid := newProcessorTestDB(t)
			tt.seed(t, db, tid)
			p := newTestProcessor(t, db, tid, fixedClock)

			has, err := p.HasInFlight(100)
			require.NoError(t, err)
			assert.Equal(t, tt.want, has)
		})
	}
}

// TestProcessorGetById is table-driven over run: the happy path checks the
// returned Model's identity, while "missing" checks the sentinel-error
// mapping — different enough shapes to keep as closures.
func TestProcessorGetById(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{name: "happy path", run: func(t *testing.T) {
			db, tid := newProcessorTestDB(t)
			id := seedParcel(t, db, tid, nil)
			p := newTestProcessor(t, db, tid, fixedClock)

			m, err := p.GetById(id)
			require.NoError(t, err)
			assert.Equal(t, id, m.Id())
		}},
		{name: "missing", run: func(t *testing.T) {
			// Regression for the reviewer's BLOCKING 2: GetById must map a
			// missing row to the package's own ErrNotFound sentinel, not
			// forward gorm.ErrRecordNotFound unchanged, so a caller can
			// errors.Is(err, parcel.ErrNotFound) uniformly.
			db, tid := newProcessorTestDB(t)
			p := newTestProcessor(t, db, tid, fixedClock)

			_, err := p.GetById(uuid.New())
			assert.ErrorIs(t, err, ErrNotFound)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
