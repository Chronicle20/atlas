package clientbound

import (
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SetFieldWriter = "SetField"

type WarpToMap struct {
	channelId channel.Id
	mapId     _map.Id
	portalId  byte
	hp        uint16
	timestamp int64
}

func NewWarpToMap(channelId channel.Id, mapId _map.Id, portalId byte, hp uint16) WarpToMap {
	return WarpToMap{
		channelId: channelId,
		mapId:     mapId,
		portalId:  portalId,
		hp:        hp,
		timestamp: fieldMsTime(time.Now()),
	}
}

func (m WarpToMap) ChannelId() channel.Id { return m.channelId }
func (m WarpToMap) MapId() _map.Id        { return m.mapId }
func (m WarpToMap) PortalId() byte        { return m.portalId }
func (m WarpToMap) Hp() uint16            { return m.hp }
func (m WarpToMap) Operation() string     { return SetFieldWriter }
func (m WarpToMap) String() string {
	return fmt.Sprintf("channelId [%d], mapId [%d], portalId [%d]", m.channelId, m.mapId, m.portalId)
}

func fieldMsTime(t time.Time) int64 {
	if t.IsZero() {
		return -1
	}
	return t.Unix()*int64(10000000) + int64(116444736000000000)
}

func (m WarpToMap) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteShort(0) // decode opt
		}
		w.WriteInt(uint32(m.channelId))
		if t.Region() == "JMS" {
			w.WriteByte(0)
			w.WriteInt(0)
		}
		w.WriteByte(0) // sNotifierMessage
		w.WriteByte(0) // bCharacterData
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			w.WriteShort(0) // nNotifierCheck
			w.WriteByte(0)  // revive
		}
		w.WriteInt(uint32(m.mapId))
		w.WriteByte(m.portalId)
		w.WriteShort(m.hp)
		if t.Region() == "GMS" && t.MajorVersion() > 28 {
			w.WriteBool(false) // Chasing
		}
		w.WriteInt64(m.timestamp)
		return w.Bytes()
	}
}

func (m *WarpToMap) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			_ = r.ReadUint16() // decode opt
		}
		m.channelId = channel.Id(r.ReadUint32())
		if t.Region() == "JMS" {
			_ = r.ReadByte()
			_ = r.ReadUint32()
		}
		_ = r.ReadByte() // sNotifierMessage
		_ = r.ReadByte() // bCharacterData
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			_ = r.ReadUint16() // nNotifierCheck
			_ = r.ReadByte()   // revive
		}
		m.mapId = _map.Id(r.ReadUint32())
		m.portalId = r.ReadByte()
		m.hp = r.ReadUint16()
		if t.Region() == "GMS" && t.MajorVersion() > 28 {
			_ = r.ReadBool() // Chasing
		}
		m.timestamp = r.ReadInt64()
	}
}
