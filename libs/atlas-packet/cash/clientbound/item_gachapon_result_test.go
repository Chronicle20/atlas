package clientbound

import (
	"context"
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// Per-version operations tables, IDA-derived per design.md §1.1
// (CCashShop::OnCashItemGachaponResult dispatch). Template JSON numbers
// decode as float64, so the fixtures use float64 exactly as the runtime does.
func gachaponOps(success byte, failed byte) map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			CashItemGachaponModeSuccess: float64(success),
			CashItemGachaponModeFailed:  float64(failed),
		},
	}
}

func gachaponOpsByVersion() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"gms_v83":  gachaponOps(0xE5, 0xE4),
		"gms_v84":  gachaponOps(0xEE, 0xED),
		"gms_v87":  gachaponOps(0xF4, 0xF3),
		"gms_v92":  gachaponOps(0xBE, 0xBD),
		"gms_v95":  gachaponOps(0xC1, 0xC0),
		"jms_v185": gachaponOps(0xEB, 0xEA),
	}
}

func sampleGachaponItem() CashInventoryItem {
	return CashInventoryItem{
		CashId:      1234567890,
		AccountId:   10,
		CharacterId: 20,
		TemplateId:  5222000,
		CommodityId: 40000,
		Quantity:    1,
		GiftFrom:    "",
		Expiration:  0,
	}
}

// The 55-byte GW_CashItemInfo blob is unconditional on this packet — unlike
// GachaponOpenDone there is no isCashItem gate byte (design.md §1.3).
// packet-audit:verify packet=cash/clientbound/CashItemGachaponResult version=gms_v83 ida=0x47c75e
// packet-audit:verify packet=cash/clientbound/CashItemGachaponResult version=gms_v84 ida=0x47f8fc
// packet-audit:verify packet=cash/clientbound/CashItemGachaponResult version=gms_v87 ida=0x487f5a
// packet-audit:verify packet=cash/clientbound/CashItemGachaponResult version=gms_v92 ida=0x491530
// packet-audit:verify packet=cash/clientbound/CashItemGachaponResult version=gms_v95 ida=0x495820
// packet-audit:verify packet=cash/clientbound/CashItemGachaponResult version=jms_v185 ida=0x48f840
//
// Correction (task-207): the addresses originally carried here (v83
// 0x478e2b, v84 0x47bf59, v87 0x4844a4, v92 0x495770, v95 0x4997e0, jms
// 0x48b21d) were CCashShop::OnPacket — the opcode dispatcher switch, not the
// CCashShop::OnCashItemGachaponResult leaf handler. Live-IDB decompile
// (idb_list sessions 41f13e0d/5881cf84/d51ecbd3/acdfccff/79906a1e/b6864e54)
// confirmed the dispatcher's case 0x14D/0x154/0x15E/0x180/0x188/0x16D calls
// CCashShop::OnCashItemGachaponResult at the addresses now cited above; the
// read order (mode, then guarded sn/remain/blob/delegate) matches this
// file's fixtures exactly on every version.
func TestCashItemGachaponSuccessEncodePerVersion(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for version, opts := range gachaponOpsByVersion() {
		out := CashItemGachaponSuccessBody(1234567890, 2, sampleGachaponItem(), 5222001, 1, 0)(l, context.Background())(opts)
		wantMode := opts["operations"].(map[string]interface{})[CashItemGachaponModeSuccess].(float64)
		if out[0] != byte(wantMode) {
			t.Fatalf("%s: mode byte = %#x, want %#x", version, out[0], byte(wantMode))
		}
		// 1 mode + 8 sn + 4 remain + 55 blob + 4 itemId + 1 count + 1 jackpot
		if len(out) != 74 {
			t.Fatalf("%s: length = %d, want 74 (hex %s)", version, len(out), hex.EncodeToString(out))
		}
	}
}

func TestCashItemGachaponFailedEncodePerVersion(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for version, opts := range gachaponOpsByVersion() {
		out := CashItemGachaponFailedBody()(l, context.Background())(opts)
		wantMode := opts["operations"].(map[string]interface{})[CashItemGachaponModeFailed].(float64)
		if len(out) != 1 {
			t.Fatalf("%s: FAILED arm must be mode-only, got %d bytes", version, len(out))
		}
		if out[0] != byte(wantMode) {
			t.Fatalf("%s: mode byte = %#x, want %#x", version, out[0], byte(wantMode))
		}
	}
}

// A missing operations table must NOT silently encode a plausible byte —
// ResolveCode returns the loud 99 sentinel. This is the DOM-25 guard.
func TestCashItemGachaponModeIsNotHardCoded(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	out := CashItemGachaponFailedBody()(l, context.Background())(map[string]interface{}{})
	if out[0] != 99 {
		t.Fatalf("unconfigured mode = %d, want the 99 sentinel — the byte is hard-coded", out[0])
	}
}

func TestCashItemGachaponSuccessRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	original := NewCashItemGachaponSuccess(0xE5, 1234567890, 2, sampleGachaponItem(), 5222001, 3, 1)
	out := original.Encode(l, context.Background())(map[string]interface{}{})
	req := request.Request(out)
	r := request.NewRequestReader(&req, 0)
	var back CashItemGachaponSuccess
	back.Decode(l, context.Background())(&r, map[string]interface{}{})
	if back.Mode() != 0xE5 || back.SN() != 1234567890 || back.Remain() != 2 {
		t.Fatalf("header round-trip failed: %+v", back)
	}
	if back.ItemId() != 5222001 || back.Count() != 3 || back.Jackpot() != 1 {
		t.Fatalf("trailing UI fields round-trip failed: %+v", back)
	}
	if back.NewItem().TemplateId != 5222000 {
		t.Fatalf("blob round-trip failed: %+v", back.NewItem())
	}
}
