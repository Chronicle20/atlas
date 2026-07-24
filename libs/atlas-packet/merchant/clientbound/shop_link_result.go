package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const ShopLinkResultWriter = "ShopLinkResult"

// packet-audit:fname CWvsContext::OnShopLinkResult
// ShopLinkResult carries the owl-warp/enter outcome as a single code byte.
// Code set identical in v83 (0x8a4e7a) and v95 (0x847d60): 0 success,
// 1 closed, 2 full, 3 busy, 4 dead, 7 no-trade, 17 denied, 18 maintenance,
// 23 FM-only; anything else = "This character is unable to do it".
type ShopLinkResult struct {
	code byte
}

func NewShopLinkResult(code byte) ShopLinkResult {
	return ShopLinkResult{code: code}
}

func (m ShopLinkResult) Code() byte {
	return m.code
}

func (m ShopLinkResult) Operation() string {
	return ShopLinkResultWriter
}

func (m ShopLinkResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.code)
		return w.Bytes()
	}
}

func (m *ShopLinkResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadByte()
	}
}
