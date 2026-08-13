package clientbound

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// packet-audit:fname CStage::OnSetField
type SetField struct {
	channelId     channel.Id
	characterData charpkt.CharacterData
	damageSeeds   []uint32
	timestamp     int64
}

func NewSetField(channelId channel.Id, characterData charpkt.CharacterData) SetField {
	seeds := make([]uint32, 4)
	for i := range seeds {
		seeds[i] = rand.Uint32()
	}
	return SetField{
		channelId:     channelId,
		characterData: characterData,
		damageSeeds:   seeds,
		timestamp:     fieldMsTime(time.Now()),
	}
}

func (m SetField) Operation() string { return SetFieldWriter }
func (m SetField) String() string {
	return fmt.Sprintf("set field channelId [%d]", m.channelId)
}

func (m SetField) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" {
			// v87+ decode-opt header (CClientOptMan::DecodeOpt); v84..86 == v83 (off-by-one fix). delta §3.1.6
			w.WriteShort(0) // decode opt
		}
		w.WriteInt(uint32(m.channelId))
		if t.IsRegion("GMS") && t.MajorAtLeast(95) {
			w.WriteInt(0) // m_dwOldDriverID: GMS reads Decode4 after channelId (v95+); v83/v87 omit it (verified CStage::OnSetField v83 @0x776020 and v87 @0x7c429c — both read sNotifierMessage (Decode1) immediately after channelId, no Decode4 in between; field introduced between v87 and v95)
		}
		if t.Region() == "JMS" {
			w.WriteByte(0)
			w.WriteInt(0)
		}
		w.WriteByte(1) // sNotifierMessage
		w.WriteByte(1) // bCharacterData

		// nNotifierCheck was introduced BETWEEN v48 and v61 and is independent of the
		// seed count, so the two are gated separately. IDA: v48 CStage::OnSetField
		// @0x5c4616 reads Decode4(channelId) @0x659... then Decode1, Decode1 and goes
		// STRAIGHT to the three seed Decode4s (fed to sub_5CD911/sub_5A49F8) — there is
		// no Decode2 in between. v61 @0x659fd3 reads the same three, then Decode2
		// @0x65a046 (nNotifierCheck, gating a DecodeStr list), then its three seeds
		// @0x65a0ea/0x65a0f4/0x65a109. Writing the short to v48 desynced every
		// subsequent byte, including the whole character payload.
		if (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS" {
			w.WriteShort(0) // nNotifierCheck
		}
		if (t.IsRegion("GMS") && t.MajorAtLeast(29)) || t.IsRegion("JMS") {
			// 3 damage seeds
			for i := 0; i < 3; i++ {
				w.WriteInt(m.damageSeeds[i])
			}
		} else {
			// 4 damage seeds
			for i := 0; i < 4; i++ {
				w.WriteInt(m.damageSeeds[i])
			}
		}

		w.WriteByteArray(m.characterData.Encode(l, ctx)(options))

		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" {
			// v87+ logout-gift block (OnSetLogoutGiftConfig reads 4 ints); v84..86 == v83 (off-by-one fix). delta §3.1.6
			w.WriteInt(0) // logout gifts
			w.WriteInt(0)
			w.WriteInt(0)
			w.WriteInt(0)
		}
		w.WriteInt64(m.timestamp)
		return w.Bytes()
	}
}

func (m SetField) CharacterData() charpkt.CharacterData { return m.characterData }

func (m *SetField) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)

		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" {
			// v87+ decode-opt header; v84..86 == v83 (off-by-one fix). delta §3.1.6
			_ = r.ReadUint16() // decode opt
		}
		m.channelId = channel.Id(r.ReadUint32())
		if t.IsRegion("GMS") && t.MajorAtLeast(95) {
			_ = r.ReadUint32() // m_dwOldDriverID (GMS v95+; v83 @0x776020 and v87 @0x7c429c omit it)
		}
		if t.Region() == "JMS" {
			_ = r.ReadByte()
			_ = r.ReadUint32()
		}
		_ = r.ReadByte() // sNotifierMessage
		_ = r.ReadByte() // bCharacterData

		// mirrors Encode: nNotifierCheck is gated separately from the seed count
		// because it appears between v48 and v61 (see the Encode comment).
		if (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS" {
			_ = r.ReadUint16() // nNotifierCheck
		}
		if (t.IsRegion("GMS") && t.MajorAtLeast(29)) || t.IsRegion("JMS") {
			m.damageSeeds = make([]uint32, 4)
			for i := 0; i < 3; i++ {
				m.damageSeeds[i] = r.ReadUint32()
			}
		} else {
			m.damageSeeds = make([]uint32, 4)
			for i := 0; i < 4; i++ {
				m.damageSeeds[i] = r.ReadUint32()
			}
		}

		m.characterData.Decode(l, ctx)(r, options)

		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" {
			// v87+ logout-gift block; v84..86 == v83 (off-by-one fix). delta §3.1.6
			_ = r.ReadUint32() // logout gifts
			_ = r.ReadUint32()
			_ = r.ReadUint32()
			_ = r.ReadUint32()
		}
		m.timestamp = r.ReadInt64()
	}
}
