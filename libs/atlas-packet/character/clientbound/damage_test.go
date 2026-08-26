package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterDamage version=gms_v83 ida=0x9832e3
// packet-audit:verify packet=character/clientbound/CharacterDamage version=gms_v87 ida=0xa08d57
// packet-audit:verify packet=character/clientbound/CharacterDamage version=gms_v95 ida=0x954c50
// packet-audit:verify packet=character/clientbound/CharacterDamage version=gms_v84 ida=0x9c3681
// packet-audit:verify packet=character/clientbound/CharacterDamage version=jms_v185 ida=0xa56e2e
func TestCharacterDamagePhysical(t *testing.T) {
	input := NewCharacterDamage(1234, model.DamageTypePhysical, 500, 100100, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestCharacterDamageMobAttackIndexIncludesBlock covers the regression fixed
// in docs/tasks/task-268-character-damage-attack-index/
// bug-character-damage-attack-index-truncation.md: the client's
// CUserRemote::OnHit gates the monsterTemplateId/left/stance/stanceRelated
// block on `nAttackIdx > -2`, i.e. every mob attack index >= 1 (not only the
// -1/0 magic/physical sentinels). The predicate must be
// `>= model.DamageTypePhysical`, matching the serverbound decoder at
// libs/atlas-packet/model/damage_taken_info.go:122/174.
func TestCharacterDamageMobAttackIndexIncludesBlock(t *testing.T) {
	input := NewCharacterDamage(1234, model.DamageType(1), 500, 100100, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}

	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0xd2, 0x04, 0x00, 0x00, // characterId 1234
		0x01,                   // attackIdx 1 (mob attack index, not -1/0)
		0xf4, 0x01, 0x00, 0x00, // damage 500
		0x04, 0x87, 0x01, 0x00, // monsterTemplateId 100100
		0x01,                   // left
		0x00,                   // stance
		0x00,                   // stanceRelated
		0xf4, 0x01, 0x00, 0x00, // damage repeated
	}
	if !bytes.Equal(got, want) {
		t.Errorf("CharacterDamage v83 mob-attack-index wire:\n got %x\nwant %x", got, want)
	}
}

// TestCharacterDamageCounterOmitsBlock covers the other side of the same
// boundary: DamageTypeCounter (-2) and below must still omit the
// monsterTemplateId/left/stance/stanceRelated block, since the client's
// guard is strictly `nAttackIdx > -2`.
func TestCharacterDamageCounterOmitsBlock(t *testing.T) {
	input := NewCharacterDamage(1234, model.DamageTypeCounter, 500, 100100, true)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0xd2, 0x04, 0x00, 0x00, // characterId 1234
		0xfe,                   // attackIdx -2 (DamageTypeCounter; block omitted)
		0xf4, 0x01, 0x00, 0x00, // damage 500
		0xf4, 0x01, 0x00, 0x00, // damage repeated
	}
	if !bytes.Equal(got, want) {
		t.Errorf("CharacterDamage v83 counter wire:\n got %x\nwant %x", got, want)
	}
}
