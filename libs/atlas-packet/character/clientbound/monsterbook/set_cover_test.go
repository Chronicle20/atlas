package monsterbook

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

// packet-audit:verify packet=character/clientbound/monsterbook/CharacterSetCover version=gms_v83 ida=0xa082d5
// packet-audit:verify packet=character/clientbound/monsterbook/CharacterSetCover version=gms_v84 ida=0xa525c3
// packet-audit:verify packet=character/clientbound/monsterbook/CharacterSetCover version=gms_v87 ida=0xa9d959
// packet-audit:verify packet=character/clientbound/monsterbook/CharacterSetCover version=gms_v95 ida=0x9cfa70
// packet-audit:verify packet=character/clientbound/monsterbook/CharacterSetCover version=jms_v185 ida=0xaec8b5
//
// CWvsContext::OnMonsterBookSetCover reads a single Decode4 (cover cardId).
// Layout is byte-identical across all 5 versions (IDA-verified at each address
// above).
func TestSetCoverEncodeShape(t *testing.T) {
	body := SetCover{CardId: 2380000}
	out := body.Encode(logrus.New(), context.Background())(map[string]interface{}{})
	if len(out) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(out))
	}
}
