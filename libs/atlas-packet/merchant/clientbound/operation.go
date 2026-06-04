package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const HiredMerchantOperationWriter = "HiredMerchantOperation"

// EntrustedShopOperationMode11 is a defined case of CWvsContext::OnEntrustedShopCheckResult
// (JMS185 @ 0xb0ee59) that carries no body. The client displays a notice string from its
// StringPool (JMS185 entry 3638; v95 plan context cites 3508). No server path currently
// raises it, so it is registered as a named constant only — a body-less mode is fully
// expressed by NewMerchantErrorSimple(EntrustedShopOperationMode11) if one ever needs it.
const EntrustedShopOperationMode11 byte = 11

// OpenShop - mode only
type OpenShop struct {
	mode byte
}

func NewOpenShop(mode byte) OpenShop {
	return OpenShop{mode: mode}
}

func (m OpenShop) Operation() string { return HiredMerchantOperationWriter }
func (m OpenShop) String() string    { return fmt.Sprintf("open shop mode [%d]", m.mode) }

func (m OpenShop) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *OpenShop) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ErrorSimple - mode only (covers ErrorRetrieveFromFredrick, ErrorAnotherCharacter, ErrorUnableToOpen, ErrorRetrieveFromFredrick2)
type ErrorSimple struct {
	mode byte
}

func NewMerchantErrorSimple(mode byte) ErrorSimple {
	return ErrorSimple{mode: mode}
}

func (m ErrorSimple) Operation() string { return HiredMerchantOperationWriter }
func (m ErrorSimple) String() string    { return fmt.Sprintf("merchant error mode [%d]", m.mode) }

func (m ErrorSimple) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ErrorSimple) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ShopSearch - mode, shopId
type ShopSearch struct {
	mode   byte
	shopId uint32
}

func NewShopSearch(mode byte, shopId uint32) ShopSearch {
	return ShopSearch{mode: mode, shopId: shopId}
}

func (m ShopSearch) Operation() string { return HiredMerchantOperationWriter }
func (m ShopSearch) String() string    { return fmt.Sprintf("shop search shopId [%d]", m.shopId) }

func (m ShopSearch) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.shopId)
		return w.Bytes()
	}
}

func (m *ShopSearch) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.shopId = r.ReadUint32()
	}
}

// ShopRename - mode, success
type ShopRename struct {
	mode    byte
	success bool
}

func NewShopRename(mode byte, success bool) ShopRename {
	return ShopRename{mode: mode, success: success}
}

func (m ShopRename) Operation() string { return HiredMerchantOperationWriter }
func (m ShopRename) String() string    { return fmt.Sprintf("shop rename success [%v]", m.success) }

func (m ShopRename) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.success)
		return w.Bytes()
	}
}

func (m *ShopRename) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.success = r.ReadBool()
	}
}

// RemoteShopWarp - mode, shopId, channelId
type RemoteShopWarp struct {
	mode      byte
	shopId    uint32
	channelId byte
}

func NewRemoteShopWarp(mode byte, shopId uint32, channelId byte) RemoteShopWarp {
	return RemoteShopWarp{mode: mode, shopId: shopId, channelId: channelId}
}

func (m RemoteShopWarp) Operation() string { return HiredMerchantOperationWriter }
func (m RemoteShopWarp) String() string {
	return fmt.Sprintf("remote shop warp shopId [%d] channelId [%d]", m.shopId, m.channelId)
}

func (m RemoteShopWarp) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.shopId)
		w.WriteByte(m.channelId)
		return w.Bytes()
	}
}

func (m *RemoteShopWarp) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.shopId = r.ReadUint32()
		m.channelId = r.ReadByte()
	}
}

// EntrustedShopUnknownChannel - mode(8), shopId, channelId.
// Mode 8 of CWvsContext::OnEntrustedShopCheckResult (JMS185 @ 0xb0ee59): the "unknown
// channel" notice. The body is Decode4(shopId) + Decode1(channelId); the client uses them
// to redirect the player toward the channel where the shop actually lives. Identical body
// layout to RemoteShopWarp but a distinct, fixed mode.
type EntrustedShopUnknownChannel struct {
	mode      byte
	shopId    uint32
	channelId byte
}

func NewEntrustedShopUnknownChannel(shopId uint32, channelId byte) EntrustedShopUnknownChannel {
	return EntrustedShopUnknownChannel{mode: 8, shopId: shopId, channelId: channelId}
}

func (m EntrustedShopUnknownChannel) Operation() string { return HiredMerchantOperationWriter }
func (m EntrustedShopUnknownChannel) String() string {
	return fmt.Sprintf("entrusted shop unknown channel shopId [%d] channelId [%d]", m.shopId, m.channelId)
}

func (m EntrustedShopUnknownChannel) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.shopId)
		w.WriteByte(m.channelId)
		return w.Bytes()
	}
}

func (m *EntrustedShopUnknownChannel) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.shopId = r.ReadUint32()
		m.channelId = r.ReadByte()
	}
}

// ConfirmManage - mode, shopId, position, serialNumber
type ConfirmManage struct {
	mode         byte
	shopId       uint32
	position     uint16
	serialNumber uint64
}

func NewConfirmManage(mode byte, shopId uint32, position uint16, serialNumber uint64) ConfirmManage {
	return ConfirmManage{mode: mode, shopId: shopId, position: position, serialNumber: serialNumber}
}

func (m ConfirmManage) Operation() string { return HiredMerchantOperationWriter }
func (m ConfirmManage) String() string {
	return fmt.Sprintf("confirm manage shopId [%d] position [%d]", m.shopId, m.position)
}

func (m ConfirmManage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.shopId)
		w.WriteShort(m.position)
		w.WriteLong(m.serialNumber)
		return w.Bytes()
	}
}

func (m *ConfirmManage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.shopId = r.ReadUint32()
		m.position = r.ReadUint16()
		m.serialNumber = r.ReadUint64()
	}
}

// FreeFormNotice - mode, message
type FreeFormNotice struct {
	mode    byte
	message string
}

func NewFreeFormNotice(mode byte, message string) FreeFormNotice {
	return FreeFormNotice{mode: mode, message: message}
}

func (m FreeFormNotice) Operation() string { return HiredMerchantOperationWriter }
func (m FreeFormNotice) String() string    { return fmt.Sprintf("free form notice") }

func (m FreeFormNotice) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(true)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *FreeFormNotice) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadBool() // always true
		m.message = r.ReadAsciiString()
	}
}
