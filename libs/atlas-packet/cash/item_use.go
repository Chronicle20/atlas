package cash

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterCashItemUseHandle = "CharacterCashItemUseHandle"

// ItemUse - CUser::SendCashItemUseRequest (partial decode: common prefix only)
type ItemUse struct {
	updateTime uint32
	source     int16
	itemId     uint32
}

func (m ItemUse) UpdateTime() uint32 { return m.updateTime }
func (m ItemUse) Source() int16      { return m.source }
func (m ItemUse) ItemId() uint32     { return m.itemId }

func (m ItemUse) Operation() string {
	return CharacterCashItemUseHandle
}

func (m ItemUse) String() string {
	return fmt.Sprintf("updateTime [%d], source [%d], itemId [%d]", m.updateTime, m.source, m.itemId)
}

func (m ItemUse) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			w.WriteInt(m.updateTime)
		}
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *ItemUse) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.updateTime = r.ReadUint32()
		}
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}
