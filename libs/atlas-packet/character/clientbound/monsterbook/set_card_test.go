package monsterbook

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

// packet-audit:verify packet=character/clientbound/monsterbook/CharacterSetCard version=gms_v83 ida=0xa081b8
// packet-audit:verify packet=character/clientbound/monsterbook/CharacterSetCard version=gms_v84 ida=0xa524a6
// packet-audit:verify packet=character/clientbound/monsterbook/CharacterSetCard version=gms_v87 ida=0xa9d83c
// packet-audit:verify packet=character/clientbound/monsterbook/CharacterSetCard version=gms_v95 ida=0x9ddcb0
// packet-audit:verify packet=character/clientbound/monsterbook/CharacterSetCard version=jms_v185 ida=0xaec797
//
// CWvsContext::OnMonsterBookSetCard reads Decode1 (added flag); when set,
// Decode4 (cardId) + Decode4 (count). The server's add path always writes
// flag + cardId + level. Layout is byte-identical across all 5 versions
// (IDA-verified at each address above).
func TestSetCardEncodeShape(t *testing.T) {
	body := SetCard{CardId: 2380000, Level: 3, Added: true}
	out := body.Encode(logrus.New(), context.Background())(map[string]interface{}{})
	if len(out) != 9 {
		t.Fatalf("expected 9-byte body, got %d", len(out))
	}
	if out[0] != 1 {
		t.Fatalf("expected flag byte=1, got %d", out[0])
	}
}

func TestSetCardEncodeShapeNotAdded(t *testing.T) {
	body := SetCard{CardId: 2380000, Level: 3, Added: false}
	out := body.Encode(logrus.New(), context.Background())(map[string]interface{}{})
	if out[0] != 0 {
		t.Fatalf("expected flag byte=0, got %d", out[0])
	}
}
