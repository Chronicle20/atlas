package handler

import (
	"atlas-channel/cashshop/inventory/asset"
	"atlas-channel/cashshop/item"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	"github.com/google/uuid"

	opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestAsset(t *testing.T, cashId int64, templateId uint32, giftFrom string, giftMessage string) asset.Model {
	t.Helper()
	return newTestGiftAsset(t, cashId, templateId, giftFrom, giftMessage, false)
}

func newTestGiftAsset(t *testing.T, cashId int64, templateId uint32, giftFrom string, giftMessage string, giftAcknowledged bool) asset.Model {
	t.Helper()
	i, err := item.NewBuilder().SetId(1).SetCashId(cashId).SetTemplateId(templateId).Build()
	if err != nil {
		t.Fatalf("item build: %v", err)
	}
	b := asset.NewBuilder(1, uuid.New(), i)
	if giftFrom != "" {
		b = b.SetGiftFrom(giftFrom)
	}
	if giftMessage != "" {
		b = b.SetGiftMessage(giftMessage)
	}
	b = b.SetGiftAcknowledged(giftAcknowledged)
	m, err := b.Build()
	if err != nil {
		t.Fatalf("asset build: %v", err)
	}
	return m
}

// TestBuildGiftListEntries pins F2's filtering rule: only locker assets with
// a non-empty GiftFrom produce a LOAD_GIFT_SUCCESS entry, carrying the
// sender name and message; assets the character bought for itself (no
// sender) are omitted entirely.
func TestBuildGiftListEntries(t *testing.T) {
	purchased := newTestAsset(t, 5000000, 1302000, "", "")
	gifted := newTestAsset(t, 5000001, 1032001, "Sender1", "Happy birthday!")

	got := buildGiftListEntries([]asset.Model{purchased, gifted})

	if len(got) != 1 {
		t.Fatalf("len(entries) = %d, want 1 (only the gifted asset)", len(got))
	}
	want := cashcb.GiftListEntry{
		SN:               5000001,
		ItemId:           1032001,
		BuyCharacterName: "Sender1",
		Text:             "Happy birthday!",
	}
	if got[0] != want {
		t.Errorf("entries[0] = %+v, want %+v", got[0], want)
	}
}

// TestBuildGiftListEntriesNoGifts confirms an all-purchased locker announces
// zero entries rather than a slice of empty-sender junk.
func TestBuildGiftListEntriesNoGifts(t *testing.T) {
	purchased := newTestAsset(t, 5000000, 1302000, "", "")

	got := buildGiftListEntries([]asset.Model{purchased})

	if len(got) != 0 {
		t.Fatalf("len(entries) = %d, want 0", len(got))
	}
}

// TestBuildGiftListEntriesSkipsAcknowledged pins task-240 Defect H: a gift
// asset already marked GiftAcknowledged has been presented in a prior cash
// shop entry and must not be re-announced, while an unacknowledged gift
// asset is still included.
func TestBuildGiftListEntriesSkipsAcknowledged(t *testing.T) {
	acknowledged := newTestGiftAsset(t, 5000001, 1032001, "Sender1", "Happy birthday!", true)
	unacknowledged := newTestGiftAsset(t, 5000002, 1032002, "Sender2", "Congrats!", false)

	got := buildGiftListEntries([]asset.Model{acknowledged, unacknowledged})

	if len(got) != 1 {
		t.Fatalf("len(entries) = %d, want 1 (only the unacknowledged gift)", len(got))
	}
	if got[0].SN != 5000002 {
		t.Errorf("entries[0].SN = %d, want 5000002 (the unacknowledged gift)", got[0].SN)
	}
}

func mustEntryTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// TestLoadGiftDoneConfigured pins the version guard: LOAD_GIFT_SUCCESS is
// only announced when the tenant's CashShopOperation writer table actually
// binds it (gms_v61+ per template survey), never unconditionally.
func TestLoadGiftDoneConfigured(t *testing.T) {
	t.Run("bound", func(t *testing.T) {
		ten := mustEntryTestTenant(t)
		ctx := tenant.WithContext(context.Background(), ten)
		writer.RegisterTenantWriterOptions(ten.Id(), []opcodes.WriterConfig{
			{OpCode: "0x9F", Writer: cashcb.CashShopOperationWriter, Options: map[string]interface{}{
				"operations": map[string]interface{}{cashcb.CashShopOperationLoadGiftDone: float64(77)},
			}},
		})
		t.Cleanup(func() { writer.EvictTenantWriterOptions(ten.Id()) })

		if !loadGiftDoneConfigured(discardLogger(), ctx) {
			t.Errorf("loadGiftDoneConfigured() = false, want true when LOAD_GIFT_SUCCESS is bound")
		}
	})

	t.Run("unbound (template_gms_12_1 / template_gms_48_1 shape)", func(t *testing.T) {
		ten := mustEntryTestTenant(t)
		ctx := tenant.WithContext(context.Background(), ten)
		writer.RegisterTenantWriterOptions(ten.Id(), []opcodes.WriterConfig{
			{OpCode: "0x9F", Writer: cashcb.CashShopOperationWriter, Options: map[string]interface{}{
				"operations": map[string]interface{}{cashcb.CashShopOperationPurchaseSuccess: float64(3)},
			}},
		})
		t.Cleanup(func() { writer.EvictTenantWriterOptions(ten.Id()) })

		if loadGiftDoneConfigured(discardLogger(), ctx) {
			t.Errorf("loadGiftDoneConfigured() = true, want false when LOAD_GIFT_SUCCESS is absent")
		}
	})

	t.Run("no writer options registered at all", func(t *testing.T) {
		ten := mustEntryTestTenant(t)
		ctx := tenant.WithContext(context.Background(), ten)

		if loadGiftDoneConfigured(discardLogger(), ctx) {
			t.Errorf("loadGiftDoneConfigured() = true, want false when the tenant has no registered writer options")
		}
	})
}
