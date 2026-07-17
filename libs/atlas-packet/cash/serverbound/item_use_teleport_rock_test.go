package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Sub-payload of SendConsumeCashItemUseRequest for teleport rocks (design §1
// Q1): shared RunMapTransferItem target payload + trailing int updateTime on
// ALL versions (v83 tail 0xA0EA53, v95 case 0x9EE059).
//
// packet-audit:verify packet=cash/serverbound/ItemUseTeleportRock version=gms_v83 ida=0xA0EA53
// packet-audit:verify packet=cash/serverbound/ItemUseTeleportRock version=gms_v95 ida=0x9EE059
func TestItemUseTeleportRockByMap(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	b := []byte{
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
		0x2A, 0x00, 0x00, 0x00, // trailing updateTime = 42
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := NewItemUseTeleportRock(false)
	p.Decode(l, ctx)(&r, nil)
	if !p.Target().Valid() || p.Target().TargetMap() != 100000000 || p.UpdateTime() != 42 {
		t.Fatalf("decode: target=%+v updateTime=%d", p.Target(), p.UpdateTime())
	}
}

func TestItemUseTeleportRockByName(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	b := []byte{
		0x01,       // byName = 1
		0x05, 0x00, // name length
		'A', 'd', 'e', 'l', 'e',
		0x00, 0x00, 0x00, 0x00,
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := NewItemUseTeleportRock(true)
	p.Decode(l, ctx)(&r, nil)
	if !p.Target().Valid() || p.Target().TargetName() != "Adele" {
		t.Fatalf("decode: %+v", p.Target())
	}
}

func TestItemUseTeleportRockAbsentTarget(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	b := []byte{0x2A, 0x00, 0x00, 0x00} // updateTime only
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := NewItemUseTeleportRock(false)
	p.Decode(l, ctx)(&r, nil)
	if p.Target().Valid() {
		t.Fatalf("absent target payload must be invalid")
	}
}
