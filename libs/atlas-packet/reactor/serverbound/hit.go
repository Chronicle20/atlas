package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const ReactorHitHandle = "ReactorHitHandle"

// HitRequest - CReactorPool::OnHitReactor
// packet-audit:fname CReactorPool::FindHitReactor
type HitRequest struct {
	oid          uint32
	isSkill      bool
	dwHitOption  uint32
	delay        uint16
	skillId      uint32
}

func (m HitRequest) Oid() uint32         { return m.oid }
func (m HitRequest) IsSkill() bool       { return m.isSkill }
func (m HitRequest) DwHitOption() uint32 { return m.dwHitOption }
func (m HitRequest) Delay() uint16       { return m.delay }
func (m HitRequest) SkillId() uint32     { return m.skillId }

func (m HitRequest) Operation() string {
	return ReactorHitHandle
}

func (m HitRequest) String() string {
	return fmt.Sprintf("oid [%d], isSkill [%t], dwHitOption [%d], delay [%d], skillId [%d]", m.oid, m.isSkill, m.dwHitOption, m.delay, m.skillId)
}

func (m HitRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.oid)
		if (t.IsRegion("GMS") && t.MajorAtLeast(72)) || t.Region() == "JMS" { // isSkill added between v61 and v72: v48 CReactorPool::FindHitReactor @0x5a5d1a and v61 @0x633ac7 go oid->dwHitOption directly; v72 @0x6928bc inserts Encode4(0). Legacy (<72) omits.
			if m.isSkill {
				w.WriteInt(1)
			} else {
				w.WriteInt(0)
			}
		}
		w.WriteInt(m.dwHitOption)
		w.WriteShort(m.delay)
		if (t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS" { // trailing skillId added between v72 and v79: v48/v61/v72 omit it; v79 @0x6b8077 and v83 @0x7356c7 append Encode4(0). Legacy (<79) omits.
			w.WriteInt(m.skillId)
		}
		return w.Bytes()
	}
}

func (m *HitRequest) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.oid = r.ReadUint32()
		if (t.IsRegion("GMS") && t.MajorAtLeast(72)) || t.Region() == "JMS" { // mirror of Encode: isSkill added between v61 and v72
			m.isSkill = r.ReadUint32() == 1
		}
		m.dwHitOption = r.ReadUint32()
		m.delay = r.ReadUint16()
		if (t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS" { // mirror of Encode: skillId added between v72 and v79
			m.skillId = r.ReadUint32()
		}
	}
}
