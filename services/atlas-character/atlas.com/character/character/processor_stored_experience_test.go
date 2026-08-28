package character_test

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	character2 "atlas-character/kafka/message/character"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// maxOutboxId returns the highest outbox row id currently persisted, or 0 if
// the table is empty. Used to mark a baseline before an action under test so
// only the messages that action enqueued are inspected.
func maxOutboxId(t *testing.T, db *gorm.DB) uint64 {
	t.Helper()
	var id uint64
	require.NoError(t, db.Model(&outbox.Entity{}).Select("COALESCE(MAX(id), 0)").Scan(&id).Error)
	return id
}

// decodeStatChangedSince decodes every STAT_CHANGED event enqueued to the
// outbox for the character-status topic with an id greater than sinceId, in
// enqueue order.
func decodeStatChangedSince(t *testing.T, db *gorm.DB, sinceId uint64) []character2.StatusEvent[character2.StatusEventStatChangedBody] {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Where("topic = ? AND id > ?", character2.EnvEventTopicCharacterStatus, sinceId).Order("id asc").Find(&rows).Error)
	var evts []character2.StatusEvent[character2.StatusEventStatChangedBody]
	for _, r := range rows {
		var evt character2.StatusEvent[character2.StatusEventStatChangedBody]
		if err := json.Unmarshal(r.MessageValue, &evt); err != nil {
			continue
		}
		if evt.Type == character2.StatusEventTypeStatChanged {
			evts = append(evts, evt)
		}
	}
	return evts
}

// TestCreditStoredExperience pins the FR-17 saturating-add contract for the
// CREDIT_STORED_EXPERIENCE command: the gachapon_experience column (the
// client's GW_CharacterStat::nTempEXP) is credited by amount, clamping at
// math.MaxUint32 rather than wrapping, and a zero-amount credit is a total
// no-op (no column write, no emitted message).
func TestCreditStoredExperience(t *testing.T) {
	tests := []struct {
		name            string
		startingBalance uint32
		amount          uint32
		expectAfter     uint32
		expectEmit      bool
	}{
		{
			name:            "credits from zero",
			startingBalance: 0,
			amount:          3000,
			expectAfter:     3000,
			expectEmit:      true,
		},
		{
			name:            "accumulates",
			startingBalance: 1000,
			amount:          500,
			expectAfter:     1500,
			expectEmit:      true,
		},
		{
			name:            "clamps at MaxUint32 instead of wrapping",
			startingBalance: 4294967290,
			amount:          100,
			expectAfter:     4294967295,
			expectEmit:      true,
		},
		{
			name:            "exactly saturates",
			startingBalance: 4294967295,
			amount:          1,
			expectAfter:     4294967295,
			expectEmit:      true,
		},
		{
			name:            "zero amount is a total no-op",
			startingBalance: 1000,
			amount:          0,
			expectAfter:     1000,
			expectEmit:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tctx := tenant.WithContext(context.Background(), testTenant())
			db := testDatabase(t)
			p := character.NewProcessor(testLogger(), tctx, db)

			input, err := character.NewEmptyBuilder().SetAccountId(1000).SetWorldId(0).SetName("StoredExp").SetLevel(1).Build()
			require.NoError(t, err)
			c, err := p.Create(message.NewBuffer())(uuid.New(), input, _map.Id(0))
			require.NoError(t, err)

			cha := channel.NewModel(1, 2)

			// Create does not persist gachaponExperience (task-055's `create`
			// helper predates this column's use), so the starting balance is
			// seeded through the processor under test itself: crediting from
			// a freshly-created zero balance never clamps for any fixture
			// value used here.
			if tt.startingBalance > 0 {
				require.NoError(t, p.CreditStoredExperienceAndEmit(uuid.New(), cha, c.Id(), tt.startingBalance, "seed"))
			}
			baseline := maxOutboxId(t, db)

			require.NoError(t, p.CreditStoredExperienceAndEmit(uuid.New(), cha, c.Id(), tt.amount, "test reason"))

			got, err := p.GetById()(c.Id())
			require.NoError(t, err)
			require.Equal(t, tt.expectAfter, got.GachaponExperience())

			evts := decodeStatChangedSince(t, db, baseline)
			if !tt.expectEmit {
				require.Empty(t, evts, "zero-amount credit must not emit any message")
				return
			}
			require.Len(t, evts, 1)
			require.Equal(t, []stat.Type{stat.TypeGachaponExperience}, evts[0].Body.Updates)
			require.Equal(t, float64(tt.expectAfter), evts[0].Body.Values["gachapon_experience"])
		})
	}
}

// TestCreditStoredExperienceUnknownCharacter asserts CreditStoredExperienceAndEmit
// fails closed for a character id that was never created, and emits nothing.
func TestCreditStoredExperienceUnknownCharacter(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	p := character.NewProcessor(testLogger(), tctx, db)

	err := p.CreditStoredExperienceAndEmit(uuid.New(), channel.NewModel(1, 2), 999999, 100, "test reason")
	require.Error(t, err)

	evts := decodeStatChangedSince(t, db, 0)
	require.Empty(t, evts)
}

// decodeExperienceChangedSince decodes every EXPERIENCE_CHANGED event enqueued
// to the outbox for the character-status topic with an id greater than
// sinceId, in enqueue order.
func decodeExperienceChangedSince(t *testing.T, db *gorm.DB, sinceId uint64) []character2.StatusEvent[character2.ExperienceChangedStatusEventBody] {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Where("topic = ? AND id > ?", character2.EnvEventTopicCharacterStatus, sinceId).Order("id asc").Find(&rows).Error)
	var evts []character2.StatusEvent[character2.ExperienceChangedStatusEventBody]
	for _, r := range rows {
		var evt character2.StatusEvent[character2.ExperienceChangedStatusEventBody]
		if err := json.Unmarshal(r.MessageValue, &evt); err != nil {
			continue
		}
		if evt.Type == character2.StatusEventTypeExperienceChanged {
			evts = append(evts, evt)
		}
	}
	return evts
}

// TestRedeemStoredExperience pins FR-11: redeeming the stored-EXP counter
// awards it as real character experience and zeroes the counter, unless the
// character's level exceeds the client's cap (CUIStatusBar::TryUseTempExp,
// characterLevel <= 0x32) or the balance is already zero, in which case it is
// a total no-op — no write, no event, no error.
func TestRedeemStoredExperience(t *testing.T) {
	tests := []struct {
		name           string
		balance        uint32
		level          byte
		expectExpDelta uint32
		expectBalance  uint32
		expectMessages bool
	}{
		{
			name:           "redeems the whole balance",
			balance:        5000,
			level:          30,
			expectExpDelta: 5000,
			expectBalance:  0,
			expectMessages: true,
		},
		{
			name:           "at the level bound",
			balance:        5000,
			level:          50,
			expectExpDelta: 5000,
			expectBalance:  0,
			expectMessages: true,
		},
		{
			name:           "above the level bound is a no-op",
			balance:        5000,
			level:          51,
			expectExpDelta: 0,
			expectBalance:  5000,
			expectMessages: false,
		},
		{
			name:           "zero balance is a no-op",
			balance:        0,
			level:          30,
			expectExpDelta: 0,
			expectBalance:  0,
			expectMessages: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tctx := tenant.WithContext(context.Background(), testTenant())
			db := testDatabase(t)
			p := character.NewProcessor(testLogger(), tctx, db)

			input, err := character.NewEmptyBuilder().SetAccountId(1000).SetWorldId(0).SetName("SExpRedeem").SetLevel(tt.level).Build()
			require.NoError(t, err)
			c, err := p.Create(message.NewBuffer())(uuid.New(), input, _map.Id(0))
			require.NoError(t, err)

			cha := channel.NewModel(1, 2)

			if tt.balance > 0 {
				require.NoError(t, p.CreditStoredExperienceAndEmit(uuid.New(), cha, c.Id(), tt.balance, "seed"))
			}

			startingExperience := c.Experience()
			baseline := maxOutboxId(t, db)

			require.NoError(t, p.RedeemStoredExperienceAndEmit(uuid.New(), cha, c.Id()))

			got, err := p.GetById()(c.Id())
			require.NoError(t, err)
			require.Equal(t, tt.expectBalance, got.GachaponExperience())
			require.Equal(t, startingExperience+tt.expectExpDelta, got.Experience())

			statEvts := decodeStatChangedSince(t, db, baseline)
			expEvts := decodeExperienceChangedSince(t, db, baseline)
			if !tt.expectMessages {
				require.Empty(t, statEvts, "no-op redeem must not emit STAT_CHANGED")
				require.Empty(t, expEvts, "no-op redeem must not emit EXPERIENCE_CHANGED")
				return
			}

			require.Len(t, expEvts, 1)
			require.Len(t, statEvts, 2)
			require.Equal(t, []stat.Type{stat.TypeExperience}, statEvts[0].Body.Updates)
			require.Equal(t, []stat.Type{stat.TypeGachaponExperience}, statEvts[1].Body.Updates)
			require.Equal(t, float64(0), statEvts[1].Body.Values["gachapon_experience"])
		})
	}
}

// TestRedeemStoredExperienceIsExactlyOnce pins FR-13: a replayed redeem must
// not double-award. The second call reads a zero balance inside its own
// transaction and short-circuits with no write and no emitted message.
func TestRedeemStoredExperienceIsExactlyOnce(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	p := character.NewProcessor(testLogger(), tctx, db)

	input, err := character.NewEmptyBuilder().SetAccountId(1000).SetWorldId(0).SetName("SExpOnce").SetLevel(30).Build()
	require.NoError(t, err)
	c, err := p.Create(message.NewBuffer())(uuid.New(), input, _map.Id(0))
	require.NoError(t, err)

	cha := channel.NewModel(1, 2)
	require.NoError(t, p.CreditStoredExperienceAndEmit(uuid.New(), cha, c.Id(), 5000, "seed"))
	startingExperience := c.Experience()

	require.NoError(t, p.RedeemStoredExperienceAndEmit(uuid.New(), cha, c.Id()))

	baseline := maxOutboxId(t, db)
	require.NoError(t, p.RedeemStoredExperienceAndEmit(uuid.New(), cha, c.Id()))

	got, err := p.GetById()(c.Id())
	require.NoError(t, err)
	require.Equal(t, uint32(0), got.GachaponExperience())
	require.Equal(t, startingExperience+5000, got.Experience())

	statEvts := decodeStatChangedSince(t, db, baseline)
	expEvts := decodeExperienceChangedSince(t, db, baseline)
	require.Empty(t, statEvts, "replayed redeem must not emit STAT_CHANGED")
	require.Empty(t, expEvts, "replayed redeem must not emit EXPERIENCE_CHANGED")
}

// TestRedeemStoredExperienceDistributionIsItem asserts the EXPERIENCE_CHANGED
// event emitted by a redeem carries exactly one distribution tagged ITEM, so
// downstream consumers can distinguish a stored-EXP redeem from other EXP
// sources.
func TestRedeemStoredExperienceDistributionIsItem(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	p := character.NewProcessor(testLogger(), tctx, db)

	input, err := character.NewEmptyBuilder().SetAccountId(1000).SetWorldId(0).SetName("SExpDist").SetLevel(30).Build()
	require.NoError(t, err)
	c, err := p.Create(message.NewBuffer())(uuid.New(), input, _map.Id(0))
	require.NoError(t, err)

	cha := channel.NewModel(1, 2)
	require.NoError(t, p.CreditStoredExperienceAndEmit(uuid.New(), cha, c.Id(), 5000, "seed"))

	baseline := maxOutboxId(t, db)
	require.NoError(t, p.RedeemStoredExperienceAndEmit(uuid.New(), cha, c.Id()))

	expEvts := decodeExperienceChangedSince(t, db, baseline)
	require.Len(t, expEvts, 1)
	require.Len(t, expEvts[0].Body.Distributions, 1)
	require.Equal(t, character2.ExperienceDistributionTypeItem, expEvts[0].Body.Distributions[0].ExperienceType)
	require.Equal(t, uint32(5000), expEvts[0].Body.Distributions[0].Amount)
}
