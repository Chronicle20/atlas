package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationCashTradeOpen struct {
	nProc             byte
	roomType          byte
	targetCharacterId uint32
	spw               uint32
	dwSN              uint32
	shopId            uint32
	unk2              byte
	position          uint16
	serialNumber      uint64
	birthday          uint32
}

func (m OperationCashTradeOpen) NProc() byte                { return m.nProc }
func (m OperationCashTradeOpen) RoomType() byte             { return m.roomType }
func (m OperationCashTradeOpen) TargetCharacterId() uint32  { return m.targetCharacterId }
func (m OperationCashTradeOpen) Spw() uint32                { return m.spw }
func (m OperationCashTradeOpen) DwSN() uint32               { return m.dwSN }
func (m OperationCashTradeOpen) ShopId() uint32             { return m.shopId }
func (m OperationCashTradeOpen) Unk2() byte                 { return m.unk2 }
func (m OperationCashTradeOpen) Position() uint16           { return m.position }
func (m OperationCashTradeOpen) SerialNumber() uint64       { return m.serialNumber }
func (m OperationCashTradeOpen) Birthday() uint32           { return m.birthday }

func (m OperationCashTradeOpen) Operation() string { return "OperationCashTradeOpen" }

func (m OperationCashTradeOpen) String() string {
	return fmt.Sprintf("nProc [%d] roomType [%d] targetCharacterId [%d] spw [%d] dwSN [%d] shopId [%d] unk2 [%d] position [%d] serialNumber [%d] birthday [%d]", m.nProc, m.roomType, m.targetCharacterId, m.spw, m.dwSN, m.shopId, m.unk2, m.position, m.serialNumber, m.birthday)
}

func (m OperationCashTradeOpen) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.nProc)
		w.WriteByte(m.roomType)
		if m.nProc == 0 && m.roomType == 6 {
			w.WriteInt(m.targetCharacterId)
		}
		if m.nProc == 4 && m.roomType == 6 {
			w.WriteInt(m.spw)
			w.WriteInt(m.dwSN)
			w.WriteByte(m.unk2)
		}
		if m.nProc == 4 && m.roomType == 5 {
			w.WriteInt(m.spw)
			w.WriteInt(m.shopId)
			w.WriteByte(m.unk2)
			w.WriteShort(m.position)
			w.WriteLong(m.serialNumber)
		}
		if m.nProc == 11 && (m.roomType == 4 || m.roomType == 5) {
			w.WriteInt(m.birthday)
		}
		return w.Bytes()
	}
}

func (m *OperationCashTradeOpen) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.nProc = r.ReadByte()
		m.roomType = r.ReadByte()
		if m.nProc == 0 && m.roomType == 6 {
			m.targetCharacterId = r.ReadUint32()
		}
		if m.nProc == 4 && m.roomType == 6 {
			m.spw = r.ReadUint32()
			m.dwSN = r.ReadUint32()
			m.unk2 = r.ReadByte()
		}
		if m.nProc == 4 && m.roomType == 5 {
			m.spw = r.ReadUint32()
			m.shopId = r.ReadUint32()
			m.unk2 = r.ReadByte()
			m.position = r.ReadUint16()
			m.serialNumber = r.ReadUint64()
		}
		if m.nProc == 11 && (m.roomType == 4 || m.roomType == 5) {
			m.birthday = r.ReadUint32()
		}
	}
}
