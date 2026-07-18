package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v72 STORAGE family verification — CTrunkDlg::OnPacket @0x704175
// (GMS_v72.1_U_DEVM.exe, port 13339). The dispatcher does Decode1(mode) then
// switches, byte-identical mode table to v79/v83:
//
//	 9 -> SetGetItems (RETRIEVE_ASSETS)            data arm
//	10 -> StringPool::GetInstance + Notice         mode-only notice (InventoryFull)
//	11 -> Notice                                   mode-only notice (NotEnoughMesos)
//	12 -> Notice                                   mode-only notice (OneOfAKind)
//	13 -> SetGetItems (STORE_ASSETS)               data arm
//	15/19 -> SetGetItems (UPDATE_MESO)             data arm
//	22 -> trunk open (SetGetItems, SHOW)           data arm
//	23 -> Decode1(flag); if flag DecodeStr(msg)     ERROR_MESSAGE
//
// The notice arms (10/11/12) consume ONLY the mode byte (no CInPacket::Decode after
// the switch), so their body is [mode]. The SetGetItems data arms carry no
// MajorVersion gate in the atlas codec, so each v72 encode is byte-equal to the
// IDA-verified v83 encode (cross-version equality).

// packet-audit:verify packet=storage/clientbound/StorageErrorInventoryFull version=gms_v72 ida=0x704175
// packet-audit:verify packet=storage/clientbound/StorageErrorNotEnoughMesos version=gms_v72 ida=0x704175
// packet-audit:verify packet=storage/clientbound/StorageErrorOneOfAKind version=gms_v72 ida=0x704175
func TestStorageNoticeArmsV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	cases := map[string][]byte{
		"InventoryFull":  NewStorageErrorInventoryFull(10).Encode(l, ctx)(nil),
		"NotEnoughMesos": NewStorageErrorNotEnoughMesos(11).Encode(l, ctx)(nil),
		"OneOfAKind":     NewStorageErrorOneOfAKind(12).Encode(l, ctx)(nil),
	}
	want := map[string]byte{"InventoryFull": 10, "NotEnoughMesos": 11, "OneOfAKind": 12}
	for name, got := range cases {
		if !bytes.Equal(got, []byte{want[name]}) {
			t.Errorf("v72 %s: got % x want %02x", name, got, want[name])
		}
	}
}

// packet-audit:verify packet=storage/clientbound/StorageErrorMessage version=gms_v72 ida=0x704175
// packet-audit:verify packet=storage/clientbound/StorageStoreAssets version=gms_v72 ida=0x704175
// packet-audit:verify packet=storage/clientbound/StorageRetrieveAssets version=gms_v72 ida=0x704175
// packet-audit:verify packet=storage/clientbound/StorageUpdateMeso version=gms_v72 ida=0x704175
// packet-audit:verify packet=storage/clientbound/StorageShow version=gms_v72 ida=0x704175
func TestStorageDataArmsV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	v72 := pt.CreateContext("GMS", 72, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	assets := []model.Asset{testAsset(), testAsset()}
	type arm struct {
		name string
		v72  []byte
		v83  []byte
	}
	arms := []arm{
		{"ErrorMessage", NewStorageErrorMessage(23, "Test error message").Encode(l, v72)(nil), NewStorageErrorMessage(23, "Test error message").Encode(l, v83)(nil)},
		{"StoreAssets", NewStorageStoreAssets(13, 16, 8, assets).Encode(l, v72)(nil), NewStorageStoreAssets(13, 16, 8, assets).Encode(l, v83)(nil)},
		{"RetrieveAssets", NewStorageRetrieveAssets(9, 16, 8, assets).Encode(l, v72)(nil), NewStorageRetrieveAssets(9, 16, 8, assets).Encode(l, v83)(nil)},
		{"UpdateMeso", NewStorageUpdateMeso(19, 24, 5000000).Encode(l, v72)(nil), NewStorageUpdateMeso(19, 24, 5000000).Encode(l, v83)(nil)},
		{"Show", NewStorageShow(22, 9200000, 16, 2|8|32, 50000, assets).Encode(l, v72)(nil), NewStorageShow(22, 9200000, 16, 2|8|32, 50000, assets).Encode(l, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v72, a.v83) {
			t.Errorf("%s v72 != v83\n v72: % x\n v83: % x", a.name, a.v72, a.v83)
		}
	}
}
