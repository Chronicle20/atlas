package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v79 STORAGE (op 0x11F) family verification — CTrunkDlg::OnPacket @0x73bc53
// (GMS_v79_1_DEVM.exe, port 13340). The dispatcher does Decode1(mode) then:
//
//	 9 -> SetGetItems @0x73921c (RETRIEVE_ASSETS)            data arm
//	10 -> StringPool::GetInstance(853) + Notice             mode-only notice
//	11 -> StringPool::GetInstance(5347) + Notice  [LABEL_47] mode-only notice
//	12 -> StringPool::GetInstance(866) + Notice             mode-only notice
//	13 -> SetGetItems (STORE_ASSETS)                        data arm
//	19 -> SetGetItems (UPDATE_MESO)                         data arm
//	22 -> trunk open (SetGetItems, SHOW)                    data arm
//	23 -> Decode1(flag); if flag DecodeStr(msg) + Notice    ERROR_MESSAGE
//
// v79 is GMS, so the jms_v185 -1 dispatcher shift does NOT apply — mode bytes
// equal the gms mode bytes. The notice arms (10/11/12) consume ONLY the mode
// byte (no CInPacket::Decode after the switch), so their body is [mode]. The
// SetGetItems data arms carry no MajorVersion gate in the atlas codec, so each
// v79 encode is byte-equal to the IDA-verified v83 encode (cross-version
// equality, the door/SpawnDoor discipline).

// packet-audit:verify packet=storage/clientbound/StorageErrorInventoryFull version=gms_v79 ida=0x73bc53
// packet-audit:verify packet=storage/clientbound/StorageErrorNotEnoughMesos version=gms_v79 ida=0x73bc53
// packet-audit:verify packet=storage/clientbound/StorageErrorOneOfAKind version=gms_v79 ida=0x73bc53
func TestStorageNoticeArmsV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	cases := map[string][]byte{
		"InventoryFull":  NewStorageErrorInventoryFull(10).Encode(l, ctx)(nil),
		"NotEnoughMesos": NewStorageErrorNotEnoughMesos(11).Encode(l, ctx)(nil),
		"OneOfAKind":     NewStorageErrorOneOfAKind(12).Encode(l, ctx)(nil),
	}
	want := map[string]byte{"InventoryFull": 10, "NotEnoughMesos": 11, "OneOfAKind": 12}
	for name, got := range cases {
		if !bytes.Equal(got, []byte{want[name]}) {
			t.Errorf("v79 %s: got % x want %02x", name, got, want[name])
		}
	}
}

// packet-audit:verify packet=storage/clientbound/StorageErrorMessage version=gms_v79 ida=0x73bc53
// packet-audit:verify packet=storage/clientbound/StorageStoreAssets version=gms_v79 ida=0x73bc53
// packet-audit:verify packet=storage/clientbound/StorageRetrieveAssets version=gms_v79 ida=0x73bc53
// packet-audit:verify packet=storage/clientbound/StorageUpdateMeso version=gms_v79 ida=0x73bc53
// packet-audit:verify packet=storage/clientbound/StorageShow version=gms_v79 ida=0x73bc53
func TestStorageDataArmsV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	v79 := pt.CreateContext("GMS", 79, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	assets := []model.Asset{testAsset(), testAsset()}
	type arm struct {
		name string
		v79  []byte
		v83  []byte
	}
	arms := []arm{
		{"ErrorMessage", NewStorageErrorMessage(23, "Test error message").Encode(l, v79)(nil), NewStorageErrorMessage(23, "Test error message").Encode(l, v83)(nil)},
		{"StoreAssets", NewStorageStoreAssets(13, 16, 8, assets).Encode(l, v79)(nil), NewStorageStoreAssets(13, 16, 8, assets).Encode(l, v83)(nil)},
		{"RetrieveAssets", NewStorageRetrieveAssets(9, 16, 8, assets).Encode(l, v79)(nil), NewStorageRetrieveAssets(9, 16, 8, assets).Encode(l, v83)(nil)},
		{"UpdateMeso", NewStorageUpdateMeso(19, 24, 5000000).Encode(l, v79)(nil), NewStorageUpdateMeso(19, 24, 5000000).Encode(l, v83)(nil)},
		{"Show", NewStorageShow(22, 9200000, 16, 2|8|32, 50000, assets).Encode(l, v79)(nil), NewStorageShow(22, 9200000, 16, 2|8|32, 50000, assets).Encode(l, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v79, a.v83) {
			t.Errorf("%s v79 != v83\n v79: % x\n v83: % x", a.name, a.v79, a.v83)
		}
	}
}
