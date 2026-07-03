package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterItemUseLotteryHandle = "CharacterItemUseLotteryHandle"

// LotteryItemUse - reward-box ("lottery") use request.
// packet-audit:fname CWvsContext::SendLotteryItemUseRequest
// Body is invariant across GMS v83-v95: slot int16, itemId int32. There is no
// leading updateTime (unlike CUser::SendStatChangeItemUseRequest). IDA-verified
// v83 fn 0xa1249f, v95 fn 0x9d6c50 (design task-131 §2.1).
type LotteryItemUse struct {
	source int16
	itemId uint32
}

func NewLotteryItemUse() LotteryItemUse {
	return LotteryItemUse{}
}

func (m LotteryItemUse) Source() int16  { return m.source }
func (m LotteryItemUse) ItemId() uint32 { return m.itemId }

func (m LotteryItemUse) Operation() string {
	return CharacterItemUseLotteryHandle
}

func (m LotteryItemUse) String() string {
	return fmt.Sprintf("source [%d], itemId [%d]", m.source, m.itemId)
}

func (m LotteryItemUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *LotteryItemUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}
