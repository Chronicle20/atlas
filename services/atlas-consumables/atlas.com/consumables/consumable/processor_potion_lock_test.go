package consumable

import (
	"atlas-consumables/character/buff"
	"atlas-consumables/character/buff/stat"
	"atlas-consumables/compartment"
	"atlas-consumables/kafka/message/consumable"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	buffmock "atlas-consumables/character/buff/mock"
	compmock "atlas-consumables/compartment/mock"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// lockedBuffs builds a single unexpired STOP_PORTION buff using the package's
// own exported constructor (CLAUDE.md test-helper rule: no test-only
// constructors).
func lockedBuffsFixture() []buff.Model {
	return []buff.Model{
		buff.NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeStopPortion, Amount: 1}}, time.Now(), time.Now().Add(time.Minute), false),
	}
}

func TestRequestItemConsume_LockedInScopeRejects(t *testing.T) {
	tests := []struct {
		name   string
		itemId int
	}{
		{"standard potion", 2000000},
		{"transformation potion", 2210000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emitted.Reset()

			reserved := false
			bp := &buffmock.ProcessorMock{
				GetByCharacterIdFunc: func(characterId uint32) ([]buff.Model, error) {
					return lockedBuffsFixture(), nil
				},
			}
			cpp := &compmock.ProcessorMock{
				RequestReserveFunc: func(transactionId uuid.UUID, characterId uint32, it inventory2.Type, expiry time.Duration, reserves []compartment.Reserves) error {
					reserved = true
					return nil
				},
			}
			p := &ProcessorImpl{l: logrus.New(), ctx: context.Background(), bp: bp, cpp: cpp}

			err := p.RequestItemConsume(channel.Model{}, 555, 3, item2.Id(tt.itemId), 1, 0)

			assert.True(t, errors.Is(err, ErrPotionLocked))
			assert.False(t, reserved, "RequestReserve must not be called when the consume is rejected pre-reservation")
		})
	}

	// Assert the emitted event shape once, on a fresh capture, rather than per
	// subtest -- the two subtests above already pin the return value and the
	// absence of a reservation call.
	emitted.Reset()
	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: context.Background(),
		bp: &buffmock.ProcessorMock{
			GetByCharacterIdFunc: func(characterId uint32) ([]buff.Model, error) {
				return lockedBuffsFixture(), nil
			},
		},
		cpp: &compmock.ProcessorMock{},
	}
	_ = p.RequestItemConsume(channel.Model{}, 555, 3, item2.Id(2000000), 1, 0)

	msgs := emitted.Messages(string(consumable.EnvEventTopic))
	if assert.Len(t, msgs, 1) {
		var evt consumable.Event[consumable.ErrorBody]
		assert.NoError(t, json.Unmarshal(msgs[0].Value, &evt))
		assert.Equal(t, "ERROR", evt.Type)
		assert.Equal(t, "POTION_LOCKED", evt.Body.Error)
	}
}

func TestRequestItemConsume_OutOfScopeIssuesNoBuffRead(t *testing.T) {
	tests := []struct {
		name   string
		itemId int
	}{
		{"town warp scroll", 2030000},
		{"pet food", 2120000},
		{"monster card", 2380000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			read := false
			bp := &buffmock.ProcessorMock{
				GetByCharacterIdFunc: func(characterId uint32) ([]buff.Model, error) {
					read = true
					return lockedBuffsFixture(), nil
				},
			}
			cpp := &compmock.ProcessorMock{}
			p := &ProcessorImpl{l: logrus.New(), ctx: context.Background(), bp: bp, cpp: cpp}

			_ = p.RequestItemConsume(channel.Model{}, 555, 3, item2.Id(tt.itemId), 1, 0)

			assert.False(t, read, "out-of-scope items must issue no buffs read at all (FR-2)")
		})
	}
}

func TestRequestItemConsume_UnlockedInScopeReserves(t *testing.T) {
	bp := &buffmock.ProcessorMock{
		GetByCharacterIdFunc: func(characterId uint32) ([]buff.Model, error) {
			return []buff.Model{}, nil
		},
	}

	var calls int
	var gotIt inventory2.Type
	var gotReserves []compartment.Reserves
	var gotExpiry time.Duration
	cpp := &compmock.ProcessorMock{
		RequestReserveFunc: func(transactionId uuid.UUID, characterId uint32, it inventory2.Type, expiry time.Duration, reserves []compartment.Reserves) error {
			calls++
			gotIt = it
			gotReserves = reserves
			gotExpiry = expiry
			return nil
		},
	}
	p := &ProcessorImpl{l: logrus.New(), ctx: context.Background(), bp: bp, cpp: cpp}

	err := p.RequestItemConsume(channel.Model{}, 555, 3, item2.Id(2000000), 1, 0)

	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Equal(t, inventory2.TypeValueUse, gotIt)
	assert.Equal(t, []compartment.Reserves{{Slot: 3, ItemId: 2000000, Quantity: 1}}, gotReserves)
	assert.Equal(t, 30*time.Second, gotExpiry)
}

func TestRequestItemConsume_BuffReadErrorFailsOpen(t *testing.T) {
	logger, hook := test.NewNullLogger()

	bp := &buffmock.ProcessorMock{
		GetByCharacterIdFunc: func(characterId uint32) ([]buff.Model, error) {
			return nil, errors.New("buffs down")
		},
	}
	reserveCalled := false
	cpp := &compmock.ProcessorMock{
		RequestReserveFunc: func(transactionId uuid.UUID, characterId uint32, it inventory2.Type, expiry time.Duration, reserves []compartment.Reserves) error {
			reserveCalled = true
			return nil
		},
	}
	p := &ProcessorImpl{l: logger, ctx: context.Background(), bp: bp, cpp: cpp}

	err := p.RequestItemConsume(channel.Model{}, 555, 3, item2.Id(2000000), 1, 0)

	assert.NoError(t, err)
	assert.True(t, reserveCalled)

	// Filter to the buff-read Warn specifically: the consume path can log
	// other Warns (consumer registration, topic resolution) that are
	// unrelated to the fail-open path.
	var warnEntries []string
	var degradeEntries []string
	for _, e := range hook.AllEntries() {
		if e.Level != logrus.WarnLevel {
			continue
		}
		if strings.Contains(e.Message, "Unable to read buffs") {
			warnEntries = append(warnEntries, e.Message)
		}
		if strings.Contains(e.Message, "Enrichment degraded") {
			degradeEntries = append(degradeEntries, e.Message)
		}
	}
	if assert.Len(t, warnEntries, 1) {
		assert.Contains(t, warnEntries[0], "555")
	}

	// degrade.Observe must fire on the fail-open branch so
	// atlas_enrichment_degraded_total increments (DOM-28).
	if assert.Len(t, degradeEntries, 1) {
		assert.Contains(t, degradeEntries[0], "consumable.potion-lock.buffs")
	}
}

func TestResolvePotionLocked(t *testing.T) {
	tests := []struct {
		name string
		mock func() ([]buff.Model, error)
		want bool
	}{
		{"locked", func() ([]buff.Model, error) { return lockedBuffsFixture(), nil }, true},
		{"unlocked", func() ([]buff.Model, error) { return []buff.Model{}, nil }, false},
		{"read error", func() ([]buff.Model, error) { return nil, errors.New("boom") }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bp := &buffmock.ProcessorMock{
				GetByCharacterIdFunc: func(characterId uint32) ([]buff.Model, error) {
					return tt.mock()
				},
			}
			got := resolvePotionLocked(logrus.New(), bp, 555)
			assert.Equal(t, tt.want, got)
		})
	}
}
