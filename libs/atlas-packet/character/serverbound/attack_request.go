package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// The four serverbound attack requests the client sends from CUserLocal:
//
//	CLOSE_RANGE_ATTACK     CUserLocal::TryDoingMeleeAttack   (0x2C v83)
//	RANGED_ATTACK          CUserLocal::TryDoingShootAttack   (0x2D v83)
//	MAGIC_ATTACK           CUserLocal::TryDoingMagicAttack   (0x2E v83)
//	TOUCH_MONSTER_ATTACK   CUserLocal::TryDoingBodyAttack    (0x2F v83)
//
// All four share the one wire structure decoded by model.AttackInfo (the
// type-specific fields branch on AttackType inside AttackInfo.Decode). These
// thin per-op wrappers exist so each registry op links to a distinct codec for
// the coverage matrix (one packet/evidence per op), exactly as the verified
// clientbound character/clientbound/Attack wraps the same model for the four
// CUserRemote::OnAttack ops. The wrapper embeds an AttackInfo and delegates
// Encode/Decode; the analyzer recurses into AttackInfo for the read order.
//
// These are the canonical per-op codec representation for the packet-audit matrix
// (exercised by attack_request_test.go). The atlas-channel handlers
// (CharacterMelee/Ranged/Magic/TouchAttackHandle) decode the same model.AttackInfo
// directly — the wire structure is identical, so the wrappers verify the same bytes.
const (
	AttackMeleeRequestHandle  = "AttackMeleeRequest"
	AttackRangedRequestHandle = "AttackRangedRequest"
	AttackMagicRequestHandle  = "AttackMagicRequest"
	AttackTouchRequestHandle  = "AttackTouchRequest"
)

// AttackMeleeRequest is CLOSE_RANGE_ATTACK (CUserLocal::TryDoingMeleeAttack).
// packet-audit:fname CUserLocal::TryDoingNormalAttack
type AttackMeleeRequest struct {
	attackInfo model.AttackInfo
}

func NewAttackMeleeRequest() AttackMeleeRequest {
	return AttackMeleeRequest{attackInfo: *model.NewAttackInfo(model.AttackTypeMelee)}
}
func (m AttackMeleeRequest) AttackInfo() model.AttackInfo { return m.attackInfo }
func (m AttackMeleeRequest) Operation() string           { return AttackMeleeRequestHandle }
func (m AttackMeleeRequest) String() string              { return attackString(m.attackInfo) }
func (m AttackMeleeRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return m.attackInfo.Encode(l, ctx)
}
func (m *AttackMeleeRequest) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return m.attackInfo.Decode(l, ctx)
}

// AttackRangedRequest is RANGED_ATTACK (CUserLocal::TryDoingShootAttack).
// packet-audit:fname CUserLocal::TryDoingShootAttack
type AttackRangedRequest struct {
	attackInfo model.AttackInfo
}

func NewAttackRangedRequest() AttackRangedRequest {
	return AttackRangedRequest{attackInfo: *model.NewAttackInfo(model.AttackTypeRanged)}
}
func (m AttackRangedRequest) AttackInfo() model.AttackInfo { return m.attackInfo }
func (m AttackRangedRequest) Operation() string           { return AttackRangedRequestHandle }
func (m AttackRangedRequest) String() string              { return attackString(m.attackInfo) }
func (m AttackRangedRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return m.attackInfo.Encode(l, ctx)
}
func (m *AttackRangedRequest) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return m.attackInfo.Decode(l, ctx)
}

// AttackMagicRequest is MAGIC_ATTACK (CUserLocal::TryDoingMagicAttack).
// packet-audit:fname CUserLocal::TryDoingMagicAttack
type AttackMagicRequest struct {
	attackInfo model.AttackInfo
}

func NewAttackMagicRequest() AttackMagicRequest {
	return AttackMagicRequest{attackInfo: *model.NewAttackInfo(model.AttackTypeMagic)}
}
func (m AttackMagicRequest) AttackInfo() model.AttackInfo { return m.attackInfo }
func (m AttackMagicRequest) Operation() string           { return AttackMagicRequestHandle }
func (m AttackMagicRequest) String() string              { return attackString(m.attackInfo) }
func (m AttackMagicRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return m.attackInfo.Encode(l, ctx)
}
func (m *AttackMagicRequest) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return m.attackInfo.Decode(l, ctx)
}

// AttackTouchRequest is TOUCH_MONSTER_ATTACK (CUserLocal::TryDoingBodyAttack).
// packet-audit:fname CUserLocal::TryDoingBodyAttack
type AttackTouchRequest struct {
	attackInfo model.AttackInfo
}

func NewAttackTouchRequest() AttackTouchRequest {
	return AttackTouchRequest{attackInfo: *model.NewAttackInfo(model.AttackTypeEnergy)}
}
func (m AttackTouchRequest) AttackInfo() model.AttackInfo { return m.attackInfo }
func (m AttackTouchRequest) Operation() string           { return AttackTouchRequestHandle }
func (m AttackTouchRequest) String() string              { return attackString(m.attackInfo) }
func (m AttackTouchRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return m.attackInfo.Encode(l, ctx)
}
func (m *AttackTouchRequest) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return m.attackInfo.Decode(l, ctx)
}

func attackString(ai model.AttackInfo) string {
	return fmt.Sprintf("skillId [%d], hits [%d], damage [%d]", ai.SkillId(), ai.Hits(), ai.Damage())
}
