package character_test

import (
	"atlas-character/character"
	character2 "atlas-character/kafka/message/character"
	dropmsg "atlas-character/kafka/message/drop"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func outboxEvents(t *testing.T, db *gorm.DB) []character2.StatusEvent[json.RawMessage] {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Order("id asc").Find(&rows).Error)
	out := make([]character2.StatusEvent[json.RawMessage], 0, len(rows))
	for _, r := range rows {
		var e character2.StatusEvent[json.RawMessage]
		require.NoError(t, json.Unmarshal(r.MessageValue, &e))
		out = append(out, e)
	}
	return out
}

func pickUpCommands(t *testing.T, capture *producertest.Capture) []dropmsg.Command[dropmsg.RequestPickUpCommandBody] {
	t.Helper()
	var out []dropmsg.Command[dropmsg.RequestPickUpCommandBody]
	for _, m := range capture.Messages(dropmsg.EnvCommandTopic) {
		var c dropmsg.Command[dropmsg.RequestPickUpCommandBody]
		require.NoError(t, json.Unmarshal(m.Value, &c))
		if c.Type == dropmsg.CommandTypeRequestPickUp {
			out = append(out, c)
		}
	}
	return out
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.MustParse("00000000-0000-0000-0000-00000000000a")).Build()
}

func TestAwardPickedUpMeso_CreditsAndEmitsMesoChangedAndStatChanged(t *testing.T) {
	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)

	c := createTestCharacter(t, tctx, db, 0)
	p := character.NewProcessor(testLogger(), tctx, db)

	before := outboxRowCount(t, db)
	txId := uuid.New()
	f := testField()

	err := p.AwardPickedUpMeso(txId, f, c.Id(), 4242, 33, false)
	require.NoError(t, err)

	got, gerr := p.GetById()(c.Id())
	require.NoError(t, gerr)
	require.Equal(t, uint32(33), got.Meso())

	require.Equal(t, before+2, outboxRowCount(t, db))

	events := outboxEvents(t, db)
	require.Len(t, events, 2)
	mesoChanged := events[len(events)-2]
	statChanged := events[len(events)-1]
	require.Equal(t, character2.StatusEventTypeMesoChanged, mesoChanged.Type)
	require.Equal(t, character2.StatusEventTypeStatChanged, statChanged.Type)
	require.Equal(t, txId, mesoChanged.TransactionId)
	require.Equal(t, txId, statChanged.TransactionId)

	var mesoBody character2.MesoChangedStatusEventBody
	require.NoError(t, json.Unmarshal(mesoChanged.Body, &mesoBody))
	require.Equal(t, int32(33), mesoBody.Amount)
	require.True(t, mesoBody.ShowEffect)
	require.Equal(t, uint32(4242), mesoBody.ActorId)
	require.Equal(t, "DROP", mesoBody.ActorType)

	require.Empty(t, pickUpCommands(t, capture))
}

func TestAwardPickedUpMeso_PickerCompletesThePickUp(t *testing.T) {
	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)

	c := createTestCharacter(t, tctx, db, 0)
	p := character.NewProcessor(testLogger(), tctx, db)

	txId := uuid.New()
	f := testField()

	err := p.AwardPickedUpMeso(txId, f, c.Id(), 4242, 33, true)
	require.NoError(t, err)

	cmds := pickUpCommands(t, capture)
	require.Len(t, cmds, 1)
	require.Equal(t, uint32(4242), cmds[0].Body.DropId)
	require.Equal(t, c.Id(), cmds[0].Body.CharacterId)
	require.Equal(t, world.Id(0), cmds[0].WorldId)
	require.Equal(t, channel.Id(1), cmds[0].ChannelId)
	require.Equal(t, _map.Id(100000000), cmds[0].MapId)
	require.Equal(t, f.Instance(), cmds[0].Instance)
}

func TestAwardPickedUpMeso_NonPickerDoesNotCompleteThePickUp(t *testing.T) {
	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)

	c := createTestCharacter(t, tctx, db, 0)
	p := character.NewProcessor(testLogger(), tctx, db)

	txId := uuid.New()
	f := testField()

	err := p.AwardPickedUpMeso(txId, f, c.Id(), 4242, 33, false)
	require.NoError(t, err)

	got, gerr := p.GetById()(c.Id())
	require.NoError(t, gerr)
	require.Equal(t, uint32(33), got.Meso())
	require.Empty(t, pickUpCommands(t, capture))
}

func TestAwardPickedUpMeso_ZeroAmountRunsNoTransactionButCompletesThePickUp(t *testing.T) {
	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)

	c := createTestCharacter(t, tctx, db, 500)
	p := character.NewProcessor(testLogger(), tctx, db)

	before := outboxRowCount(t, db)
	txId := uuid.New()
	f := testField()

	err := p.AwardPickedUpMeso(txId, f, c.Id(), 4242, 0, true)
	require.NoError(t, err)

	got, gerr := p.GetById()(c.Id())
	require.NoError(t, gerr)
	require.Equal(t, uint32(500), got.Meso())
	require.Equal(t, before, outboxRowCount(t, db))

	cmds := pickUpCommands(t, capture)
	require.Len(t, cmds, 1)
}

func TestAwardPickedUpMeso_OverflowSkipsTheCreditButStillCompletesThePickUp(t *testing.T) {
	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)

	c := createTestCharacter(t, tctx, db, 1000000000)
	p := character.NewProcessor(testLogger(), tctx, db)
	require.NoError(t, p.RequestChangeMeso(uuid.New(), c.Id(), 2147483647, 0, "SYSTEM", false))
	capture.Reset()

	before := outboxRowCount(t, db)
	txId := uuid.New()
	f := testField()

	err := p.AwardPickedUpMeso(txId, f, c.Id(), 4242, 2147483647, true)
	require.ErrorIs(t, err, character.ErrMesoOverflow)

	got, gerr := p.GetById()(c.Id())
	require.NoError(t, gerr)
	require.Equal(t, uint32(3147483647), got.Meso())
	require.Equal(t, before, outboxRowCount(t, db))

	cmds := pickUpCommands(t, capture)
	require.Len(t, cmds, 1)
}

func TestAwardPickedUpMeso_AmountAboveInt32IsRejected(t *testing.T) {
	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)

	c := createTestCharacter(t, tctx, db, 0)
	p := character.NewProcessor(testLogger(), tctx, db)

	before := outboxRowCount(t, db)
	txId := uuid.New()
	f := testField()

	err := p.AwardPickedUpMeso(txId, f, c.Id(), 4242, 2147483648, true)
	require.ErrorIs(t, err, character.ErrMesoOverflow)

	got, gerr := p.GetById()(c.Id())
	require.NoError(t, gerr)
	require.Equal(t, uint32(0), got.Meso())
	require.Equal(t, before, outboxRowCount(t, db))

	cmds := pickUpCommands(t, capture)
	require.Len(t, cmds, 1)
}
