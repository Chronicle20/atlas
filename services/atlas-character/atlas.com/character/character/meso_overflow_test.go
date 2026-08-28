package character_test

import (
	"atlas-character/character"
	character2 "atlas-character/kafka/message/character"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// RequestChangeMeso's overflow guard used to return ErrMesoOverflow with no
// emission at all, which stranded the award_mesos step of a meso_sack_use saga
// until the orchestrator's timeout backstop fired. It must now emit the same
// non-generic meso-error body the NOT_ENOUGH_MESO path uses, with a
// MESO_OVERFLOW code, so the step fails fast. The error return is unchanged —
// rejection, never clamping.
func TestRequestChangeMeso_OverflowEmitsMesoOverflowErrorEvent(t *testing.T) {
	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)

	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)
	// Starting balance chosen so that a max-int32 credit succeeds first
	// (1,000,000,000 + 2,147,483,647 = 3,147,483,647, well under MaxUint32)
	// and a second max-int32 credit from that new balance is guaranteed to
	// cross MaxUint32 (4,294,967,295): 3,147,483,647 + 2,147,483,647 =
	// 5,294,967,294. Doubling a max-int32 credit from a zero base, as
	// originally sketched, tops out at 4,294,967,294 — one below MaxUint32 —
	// and can never overflow, since amount is capped at int32's max; a
	// nonzero starting balance is required to actually cross the ceiling.
	c := createTestCharacter(t, tctx, db, 1000000000)

	p := character.NewProcessor(testLogger(), tctx, db)
	require.NoError(t, p.RequestChangeMeso(uuid.New(), c.Id(), 2147483647, 0, "SYSTEM", false))
	capture.Reset()

	before := outboxRowCount(t, db)
	err := p.RequestChangeMeso(uuid.New(), c.Id(), 2147483647, 0, "SYSTEM", false)
	require.ErrorIs(t, err, character.ErrMesoOverflow)

	// The rejection commits nothing: no MESO_CHANGED / STAT_CHANGED outbox rows.
	require.Equal(t, before, outboxRowCount(t, db))

	msgs := capture.Messages(string(character2.EnvEventTopicCharacterStatus))
	require.Len(t, msgs, 1, "overflow must emit exactly one status event")

	var e character2.StatusEvent[character2.StatusEventMesoErrorBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &e))
	require.Equal(t, character2.StatusEventTypeError, e.Type)
	require.Equal(t, c.Id(), e.CharacterId)
	require.Equal(t, character2.StatusEventErrorTypeMesoOverflow, e.Body.Error)
	require.Equal(t, int32(2147483647), e.Body.Amount)

	// And the balance is untouched (still the post-first-call balance).
	got, gerr := p.GetById()(c.Id())
	require.NoError(t, gerr)
	require.Equal(t, uint32(3147483647), got.Meso())
}
