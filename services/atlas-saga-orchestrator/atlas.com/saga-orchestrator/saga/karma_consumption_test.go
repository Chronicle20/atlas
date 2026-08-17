package saga

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
)

func TestClearKarmaFromSnapshotEquip(t *testing.T) {
	in := AssetSnapshot{TemplateId: 1002357, Flag: uint16(af.FlagUntradeable) | uint16(af.FlagKarmaEquip)}
	out := clearKarmaFromSnapshot(in)
	if af.HasFlag(out.Flag, af.FlagKarmaEquip) {
		t.Fatal("the equip karma bit survived the transfer")
	}
	if !af.HasFlag(out.Flag, af.FlagUntradeable) {
		t.Fatal("clearing karma disturbed FlagUntradeable; the item must arrive UNTRADEABLE")
	}
}

func TestClearKarmaFromSnapshotBundle(t *testing.T) {
	in := AssetSnapshot{TemplateId: 2280000, Flag: uint16(af.FlagUntradeable) | uint16(af.FlagKarmaUse)}
	out := clearKarmaFromSnapshot(in)
	if af.HasFlag(out.Flag, af.FlagKarmaUse) {
		t.Fatal("the bundle karma bit survived the transfer")
	}
}

// TestClearKarmaFromSnapshotLeavesSpikesAlone: 0x02 on an EQUIP is FlagSpikes,
// and a traded spiked equip must arrive spiked.
func TestClearKarmaFromSnapshotLeavesSpikesAlone(t *testing.T) {
	in := AssetSnapshot{TemplateId: 1002357, Flag: uint16(af.FlagSpikes)}
	out := clearKarmaFromSnapshot(in)
	if !af.HasFlag(out.Flag, af.FlagSpikes) {
		t.Fatal("a spiked equip lost its spikes in transfer")
	}
}

// TestClearKarmaFromSnapshotSkipsPets: KarmaFlagFor reports no bit for a pet,
// and the pet bit is 0x01 = FlagLock. A pet passing through a trade must be
// untouched.
func TestClearKarmaFromSnapshotSkipsPets(t *testing.T) {
	in := AssetSnapshot{TemplateId: 5000000, Flag: uint16(af.FlagLock)}
	out := clearKarmaFromSnapshot(in)
	if out.Flag != in.Flag {
		t.Fatalf("a pet's flag changed in transfer: %#x -> %#x", in.Flag, out.Flag)
	}
}

// TestSettlementClearsTheMarkAndUnwindDoesNot is FR-7.4 + FR-7.6 together.
//
// It builds a settlement payload whose side-1 item (templateId 1302000, an
// EQUIP per asset.KarmaFlagFor's 1xxxxxx rule) carries FlagUntradeable |
// FlagKarmaEquip, expands it, and asserts the AcceptToCharacter step that
// delivers that item to side 0 arrives with the karma bit CLEAR (and
// FlagUntradeable still SET, so the item is genuinely untradeable for its new
// owner).
//
// It then builds the equivalent unwind payload — same owner, same
// karma-marked snapshot — expands it, and asserts the returned item's karma
// bit is still SET. That is the divergence the whole feature depends on: a
// completed transfer spends the mark, a cancelled one does not, and both
// expanders share assetDataFromSnapshot, so this test is the only thing that
// proves the settlement call site is the one that changed.
func TestSettlementClearsTheMarkAndUnwindDoesNot(t *testing.T) {
	const markedTemplateId = uint32(1302000)
	markedFlag := uint16(af.FlagUntradeable) | uint16(af.FlagKarmaEquip)

	settlement := tradeSettlementFixture()
	settlement.Sides[1].Items[0].Snapshot.TemplateId = markedTemplateId
	settlement.Sides[1].Items[0].Snapshot.Flag = markedFlag

	settlementSteps := expandSettlement(t, settlement)

	var settlementDelivery *AcceptToCharacterPayload
	for _, s := range settlementSteps {
		if s.Action() != AcceptToCharacter {
			continue
		}
		pl, ok := s.Payload().(AcceptToCharacterPayload)
		require.True(t, ok)
		if pl.TemplateId == markedTemplateId {
			settlementDelivery = &pl
		}
	}
	require.NotNil(t, settlementDelivery, "settlement never delivered the marked item")
	require.False(t, af.HasFlag(settlementDelivery.AssetData.Flag, af.FlagKarmaEquip),
		"a COMPLETED trade must consume the karma mark, but the delivered item still carries it")
	require.True(t, af.HasFlag(settlementDelivery.AssetData.Flag, af.FlagUntradeable),
		"clearing the karma mark must not disturb FlagUntradeable")

	unwind := TradeUnwindPayload{
		TransactionId: uuid.New(),
		Items: []TradeUnwindItem{
			{OwnerId: settlement.Sides[1].CharacterId, Item: settlement.Sides[1].Items[0]},
		},
	}
	unwindSteps := expandUnwind(t, unwind)

	var unwindReturn *AcceptToCharacterPayload
	for _, s := range unwindSteps {
		if s.Action() != AcceptToCharacter {
			continue
		}
		pl, ok := s.Payload().(AcceptToCharacterPayload)
		require.True(t, ok)
		if pl.TemplateId == markedTemplateId {
			unwindReturn = &pl
		}
	}
	require.NotNil(t, unwindReturn, "unwind never returned the marked item")
	require.True(t, af.HasFlag(unwindReturn.AssetData.Flag, af.FlagKarmaEquip),
		"an UNWOUND (cancelled) trade must preserve the karma mark, but the returned item lost it")
}
