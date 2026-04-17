package model

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func NewDamageInfo(hits byte) *DamageInfo {
	return &DamageInfo{hits: hits}
}

type DamageInfo struct {
	hits                byte
	monsterId           uint32
	hitAction           byte
	forceAction         byte
	frameIdx            byte
	calcDamageStatIndex byte
	hitPositionX        uint16
	hitPositionY        uint16
	previousPositionX   uint16
	previousPositionY   uint16
	delay               uint16
	damages             []uint32
	crc                 uint32
}

func (m *DamageInfo) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.monsterId = r.ReadUint32()
		m.hitAction = r.ReadByte()
		m.forceAction = r.ReadByte()
		m.frameIdx = r.ReadByte()
		m.calcDamageStatIndex = r.ReadByte()
		m.hitPositionX = r.ReadUint16()
		m.hitPositionY = r.ReadUint16()
		m.previousPositionX = r.ReadUint16()
		m.previousPositionY = r.ReadUint16()
		m.delay = r.ReadUint16()
		for range m.hits {
			m.damages = append(m.damages, r.ReadUint32())
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 83 {
			m.crc = r.ReadUint32()
		}
	}
}

func (m *DamageInfo) Damages() []uint32 {
	return m.damages
}

func (m *DamageInfo) MonsterId() uint32 {
	return m.monsterId
}

func (m *DamageInfo) HitAction() byte {
	return m.hitAction
}

// Builder methods for constructing DamageInfo in the server-send path.

func (m *DamageInfo) SetMonsterId(monsterId uint32) *DamageInfo {
	m.monsterId = monsterId
	return m
}

func (m *DamageInfo) SetHitAction(hitAction byte) *DamageInfo {
	m.hitAction = hitAction
	return m
}

func (m *DamageInfo) SetDamages(damages []uint32) *DamageInfo {
	m.damages = damages
	return m
}
