package model

import (
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
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

func (m *DamageInfo) Decode(_ logrus.FieldLogger, t tenant.Model, _ map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
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
