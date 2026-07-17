package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const TeleportRockAddMapHandle = "TeleportRockAddMapHandle"

// AddMap - CWvsContext::SendMapTransferRequest (TROCK_ADD_MAP).
// Layout (design task-124 §1 Q1, version-invariant):
//
//	byte nType                 // 1 = register, 0 = delete
//	byte bCanTransferContinent // 0 = regular list (5), 1 = VIP list (10)
//	if nType == 0: int dwTargetField
//
// On register the client sends NO map id: the current map comes from
// server-side session state, never from the packet.
type AddMap struct {
	register bool
	vip      bool
	mapId    uint32
}

func NewAddMap(register bool, vip bool, mapId uint32) AddMap {
	return AddMap{register: register, vip: vip, mapId: mapId}
}

func (m AddMap) Register() bool    { return m.register }
func (m AddMap) Vip() bool         { return m.vip }
func (m AddMap) MapId() uint32     { return m.mapId }
func (m AddMap) Operation() string { return TeleportRockAddMapHandle }

func (m AddMap) String() string {
	return fmt.Sprintf("AddMap{register=%v vip=%v mapId=%d}", m.register, m.vip, m.mapId)
}

func (m AddMap) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.register)
		w.WriteBool(m.vip)
		if !m.register {
			w.WriteInt(m.mapId)
		}
		return w.Bytes()
	}
}

func (m *AddMap) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.register = r.ReadBool()
		m.vip = r.ReadBool()
		if !m.register && r.Available() >= 4 {
			m.mapId = r.ReadUint32()
		}
	}
}
