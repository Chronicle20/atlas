package clientbound

import (
	"context"
	"fmt"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// SetItcWriter is the registry writer name (Operation()) for the SET_ITC
// clientbound packet — the MTS (ITC) scene-transition packet. CStage::OnSetITC
// reads the full migrate-in CharacterData block, then the account name, then
// five ITC config int32s, then an 8-byte FILETIME contract date, and pushes a
// CITC stage so the in-game MTS view opens.
//
// IDA-verified read order (identical in all five versions):
//
//	CStage::OnSetITC -> CharacterData::Decode(a2)
//	  v83 0x7774d1 (MapleStory_dump.exe)        -> body reader sub_59EF9D 0x59ef9d
//	  v84 0x799e7a (GMS_v84.1_U_DEVM.exe)        -> CITC ctor sub_5AE011 -> LoadData sub_5AF339 0x5af339
//	  v87 0x7c57d0 (GMSv87_4GB.exe)              -> CITC::CITC 0x5cd970 -> CITC::LoadData 0x5ced61
//	  v95 0x71af60 (GMS_v95.0_U_DEVM.exe)        -> CITC::CITC 0x574d00 -> CITC::LoadData 0x574a60
//	  jms 0x7ef6fa (MapleStory_dump_SCY.exe)     -> CITC::CITC 0x60311a -> CITC::LoadData 0x60448e
//
// The body reader (CITC::LoadData / sub_59EF9D / sub_5AF339) performs, AFTER the
// CharacterData block:
//
//	DecodeStr  m_sNexonClubID    (account name)
//	Decode4    m_nRegisterFeeMeso    (listing/registration fee)
//	Decode4    m_nCommissionRate     (sale commission rate %)
//	Decode4    m_nCommissionBase     (commission base)
//	Decode4    m_nAuctionDurationMin (auction min hours)
//	Decode4    m_nAuctionDurationMax (auction max hours)
//	DecodeBuffer(8)  ftITCDateExpired (FILETIME contract/expiry date)
//
// The named struct fields are from the v95 CITC::LoadData decompile (0x574a60),
// which carries Hex-Rays member names; v83/v84/v87/jms read the same five
// Decode4 + DecodeBuffer(8) in the same order (the per-version client-side
// account-name display formatting around the single DecodeStr does NOT change
// the wire read order — the five int32s + 8-byte buffer are unconditional).
//
// This is an ENVELOPE writer: the inner CharacterData shape is the same block
// already encoded by CashShopOpen (CStage::OnSetCashShop), reused verbatim via
// the charpkt.CharacterData codec (audited under the character domain).
const SetItcWriter = "SetItc"

// ITC config defaults (Cosmic-faithful; the five Decode4 values the client
// reads after the account name). Sourced from Cosmic's
// MapleClient/PacketCreator.openCashShop(c, true) / EnterMTSHandler, which is
// the only faithful MTS reference available. listingFee/auctionMin/auctionMax
// map cleanly to MTS config concepts (registration fee, auction duration
// floor/ceiling, in hours); commissionRate/commissionBase (7 / 500) are client
// display constants with no clean config analogue and are sent as documented
// constants. These are used when no per-tenant MTS config override is supplied.
const (
	DefaultItcListingFee      uint32 = 5000 // m_nRegisterFeeMeso  (0x1388)
	DefaultItcCommissionRate  uint32 = 7    // m_nCommissionRate   (client constant)
	DefaultItcCommissionBase  uint32 = 500  // m_nCommissionBase   (client constant, 0x1F4)
	DefaultItcAuctionMinHours uint32 = 24   // m_nAuctionDurationMin (0x18)
	DefaultItcAuctionMaxHours uint32 = 168  // m_nAuctionDurationMax (0xA8)
)

// DefaultItcContractDate is the 8-byte FILETIME contract/expiry date the client
// displays. CITC::LoadData reads it via DecodeBuffer(8) and adds it to the local
// time (CITC::FileTimeAddition). Cosmic sends a fixed value; we mirror those
// bytes as a documented constant (little-endian FILETIME on the wire).
//
//	Cosmic EnterMTSHandler: 70 AA A7 C5 4E C1 CA 01
var DefaultItcContractDate = [8]byte{0x70, 0xAA, 0xA7, 0xC5, 0x4E, 0xC1, 0xCA, 0x01}

// SetItc is the SET_ITC clientbound writer. Body = CharacterData block +
// account name (ZXString) + 5×int32 ITC config + 8-byte FILETIME date.
type SetItc struct {
	characterData   charpkt.CharacterData
	accountName     string
	listingFee      uint32
	commissionRate  uint32
	commissionBase  uint32
	auctionMinHours uint32
	auctionMaxHours uint32
	contractDate    [8]byte
}

// NewSetItc builds a SET_ITC packet with the Cosmic-faithful ITC config
// defaults and contract date. Callers that have per-tenant MTS config can use
// NewSetItcWithConfig instead.
func NewSetItc(characterData charpkt.CharacterData, accountName string) SetItc {
	return SetItc{
		characterData:   characterData,
		accountName:     accountName,
		listingFee:      DefaultItcListingFee,
		commissionRate:  DefaultItcCommissionRate,
		commissionBase:  DefaultItcCommissionBase,
		auctionMinHours: DefaultItcAuctionMinHours,
		auctionMaxHours: DefaultItcAuctionMaxHours,
		contractDate:    DefaultItcContractDate,
	}
}

// NewSetItcWithConfig builds a SET_ITC packet with explicit ITC config values
// (e.g. from an MTS config resource) and contract date.
func NewSetItcWithConfig(characterData charpkt.CharacterData, accountName string, listingFee, commissionRate, commissionBase, auctionMinHours, auctionMaxHours uint32, contractDate [8]byte) SetItc {
	return SetItc{
		characterData:   characterData,
		accountName:     accountName,
		listingFee:      listingFee,
		commissionRate:  commissionRate,
		commissionBase:  commissionBase,
		auctionMinHours: auctionMinHours,
		auctionMaxHours: auctionMaxHours,
		contractDate:    contractDate,
	}
}

func (m SetItc) Operation() string { return SetItcWriter }
func (m SetItc) String() string {
	return fmt.Sprintf("set itc account [%s] listingFee [%d] auction [%d..%d]", m.accountName, m.listingFee, m.auctionMinHours, m.auctionMaxHours)
}

func (m SetItc) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByteArray(m.characterData.Encode(l, ctx)(options)) // CharacterData::Decode block
		w.WriteAsciiString(m.accountName)                         // DecodeStr m_sNexonClubID
		w.WriteInt(m.listingFee)                                  // Decode4 m_nRegisterFeeMeso
		w.WriteInt(m.commissionRate)                              // Decode4 m_nCommissionRate
		w.WriteInt(m.commissionBase)                              // Decode4 m_nCommissionBase
		w.WriteInt(m.auctionMinHours)                             // Decode4 m_nAuctionDurationMin
		w.WriteInt(m.auctionMaxHours)                             // Decode4 m_nAuctionDurationMax
		w.WriteByteArray(m.contractDate[:])                       // DecodeBuffer(8) ftITCDateExpired
		return w.Bytes()
	}
}

func (m *SetItc) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterData.Decode(l, ctx)(r, options)
		m.accountName = r.ReadAsciiString()
		m.listingFee = r.ReadUint32()
		m.commissionRate = r.ReadUint32()
		m.commissionBase = r.ReadUint32()
		m.auctionMinHours = r.ReadUint32()
		m.auctionMaxHours = r.ReadUint32()
		copy(m.contractDate[:], r.ReadBytes(8))
	}
}

func (m SetItc) CharacterData() charpkt.CharacterData { return m.characterData }
func (m SetItc) AccountName() string                  { return m.accountName }
func (m SetItc) ListingFee() uint32                   { return m.listingFee }
func (m SetItc) CommissionRate() uint32               { return m.commissionRate }
func (m SetItc) CommissionBase() uint32               { return m.commissionBase }
func (m SetItc) AuctionMinHours() uint32              { return m.auctionMinHours }
func (m SetItc) AuctionMaxHours() uint32              { return m.auctionMaxHours }
func (m SetItc) ContractDate() [8]byte                { return m.contractDate }
