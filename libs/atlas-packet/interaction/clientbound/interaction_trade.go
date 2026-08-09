// Package clientbound — trade-family (CTradingRoomDlg / CCashTradingRoomDlg)
// mode arms. Kept out of interaction_body.go, which is already at the size
// limit for a single family file (task-205 design.md §4.2).
package clientbound

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// InteractionTradePutItem is the mode-15 arm of CTradingRoomDlg's dispatcher
// (v83 CTradingRoomDlg::OnPacket @0x7c1f6d -> CTradingRoomDlg::OnPutItem
// @0x7c1fb7): Decode1 side, Decode1
// trade slot, then GW_ItemSlotBase. side is the recipient-relative room side
// (0 = the receiving client's own side, 1 = the counterparty); tradeSlot is the
// 1..9 slot within that side's grid.
//
// packet-audit:fname CTradingRoomDlg::OnPutItem
type InteractionTradePutItem struct {
	mode      byte
	side      byte
	tradeSlot byte
	asset     model.Asset
}

func NewInteractionTradePutItem(mode byte, side byte, tradeSlot byte, a model.Asset) InteractionTradePutItem {
	return InteractionTradePutItem{mode: mode, side: side, tradeSlot: tradeSlot, asset: a}
}

func (m InteractionTradePutItem) Operation() string  { return CharacterInteractionWriter }
func (m InteractionTradePutItem) Mode() byte         { return m.mode }
func (m InteractionTradePutItem) Side() byte         { return m.side }
func (m InteractionTradePutItem) TradeSlot() byte    { return m.tradeSlot }
func (m InteractionTradePutItem) Asset() model.Asset { return m.asset }

func (m InteractionTradePutItem) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.side)
		w.WriteByte(m.tradeSlot)
		w.WriteByteArray(m.asset.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *InteractionTradePutItem) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.side = r.ReadByte()
		m.tradeSlot = r.ReadByte()
		m.asset = model.NewAsset(true, 0, 0, time.Time{})
		m.asset.Decode(l, ctx)(r, options)
	}
}

// InteractionTradeAddMeso is the mode-16 arm of CTradingRoomDlg's dispatcher
// (v83 CTradingRoomDlg::OnPutMoney @0x7c208e): Decode1 side, Decode4 amount. The client ASSIGNS
// the value (this[v3+115] = Decode4) rather than accumulating, so re-sending
// the last valid amount is an authoritative correction the client's view snaps
// back to.
//
// packet-audit:fname CTradingRoomDlg::OnPutMoney
type InteractionTradeAddMeso struct {
	mode   byte
	side   byte
	amount uint32
}

func NewInteractionTradeAddMeso(mode byte, side byte, amount uint32) InteractionTradeAddMeso {
	return InteractionTradeAddMeso{mode: mode, side: side, amount: amount}
}

func (m InteractionTradeAddMeso) Operation() string { return CharacterInteractionWriter }
func (m InteractionTradeAddMeso) Mode() byte        { return m.mode }
func (m InteractionTradeAddMeso) Side() byte        { return m.side }
func (m InteractionTradeAddMeso) Amount() uint32    { return m.amount }

func (m InteractionTradeAddMeso) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.side)
		w.WriteInt(m.amount)
		return w.Bytes()
	}
}

func (m *InteractionTradeAddMeso) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.side = r.ReadByte()
		m.amount = r.ReadUint32()
	}
}

// InteractionTradeConfirm is the mode-17 arm (v83 CTradingRoomDlg::OnTrade
// @0x7c20bc). It carries NO body: the client sets its confirmed flag, redraws,
// and immediately sends serverbound PLAYER_INTERACTION mode 0x14 (TRANSACTION)
// with its own {itemId, itemCRC} attestation list.
//
// Because receipt auto-replies, this arm is broadcast ONLY after BOTH sides
// have confirmed — sending it on the first confirm would drive the
// counterparty's attestation without its owner ever pressing Trade
// (design §6.2).
//
// packet-audit:fname CTradingRoomDlg::OnTrade
type InteractionTradeConfirm struct {
	mode byte
}

func NewInteractionTradeConfirm(mode byte) InteractionTradeConfirm {
	return InteractionTradeConfirm{mode: mode}
}

func (m InteractionTradeConfirm) Operation() string { return CharacterInteractionWriter }
func (m InteractionTradeConfirm) Mode() byte        { return m.mode }

func (m InteractionTradeConfirm) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *InteractionTradeConfirm) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// InteractionTradeMesoLimit is the mode-21 arm (v83
// CTradingRoomDlg::OnMesoLimitRefused @0x7c21bd), the
// server-side twin of CTradingRoomDlg::PutMoney's client-side daily-meso guard.
// Bodyless: the client shows SP_3977, clears its local confirm flag
// (this[111] = 0) and re-enables both confirm buttons.
//
// CCashTradingRoomDlg::OnPacket (@0x4833b4) dispatches 15/16/17 only — there is
// no mode-21 arm in the cash room. Where the arm is absent, meso rejection
// degrades to the authoritative TRADE_ADD_MESO re-echo alone (design §4.2).
//
// packet-audit:fname CTradingRoomDlg::OnMesoLimitRefused
type InteractionTradeMesoLimit struct {
	mode byte
}

func NewInteractionTradeMesoLimit(mode byte) InteractionTradeMesoLimit {
	return InteractionTradeMesoLimit{mode: mode}
}

func (m InteractionTradeMesoLimit) Operation() string { return CharacterInteractionWriter }
func (m InteractionTradeMesoLimit) Mode() byte        { return m.mode }

func (m InteractionTradeMesoLimit) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *InteractionTradeMesoLimit) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
