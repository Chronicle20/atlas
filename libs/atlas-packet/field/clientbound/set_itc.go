package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// SetItcWriter is the registry writer name (Operation()) for the SET_ITC
// clientbound packet — the MTS (ITC) scene-transition packet. CStage::OnSetITC
// reads the full migrate-in CharacterData block, then the account name, then
// five ITC config int32s, then an 8-byte FILETIME that is the SERVER'S CURRENT
// TIME (a clock-sync value, NOT an expiry date), and pushes a CITC stage so the
// in-game MTS view opens.
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
//	DecodeBuffer(8)  ftSvr (server-now FILETIME; used to set m_ftRel — see below)
//
// CRITICAL — the trailing 8-byte FILETIME is the server clock, not a date.
// The body reader (v83 sub_59EF9D 0x59ef9d) does, with the 8 bytes it reads:
//
//	GetLocalTime()                 -> ftLocal   (the client's wall clock)
//	DecodeBuffer(8)                -> ftSvr     (this field)
//	m_ftRel = ftSvr - ftLocal      (CITC+52/+56; sub_59FD49 subtract)
//
// and thereafter CITC::GetCorrectTime() returns ftLocal + m_ftRel == ftSvr, so
// EVERY ITC countdown the client renders relative to "now" (the bid dialog's
// TIME LEFT — CITCBidAuctionDlg::Draw v83 sub_5C309A / v95 0x58e050 — and the
// item tooltip's remaining-time line) is computed as (ftITCDateExpired - ftSvr).
// If ftSvr is stale, every countdown is off by (realNow - ftSvr): sending a
// fixed ~2010 constant here made a 24h auction render as
// "143160 hr" (≈16.3 years). The absolute "Sold Until" DATE column is unaffected
// because it prints ftITCDateExpired directly and never consults m_ftRel — which
// is why the bug shows only in the countdowns. Send the real server time
// (MsTimeBytes(time.Now())) so m_ftRel ≈ 0 and countdowns are correct regardless
// of the client's local clock.
//
// This is an ENVELOPE writer: the inner CharacterData shape is the same block
// already encoded by CashShopOpen (CStage::OnSetCashShop), reused verbatim via
// the charpkt.CharacterData codec (audited under the character domain).
const SetItcWriter = "SetItc"

// Interim fallback values for the five ITC config int32s the client reads after
// the account name. These are the last-resort defaults used only when the caller
// supplies no per-tenant MTS configuration (NewSetItcWithConfig); the intended
// source is the tenant mts-configs resource. listingFee/auctionMin/auctionMax map
// to MTS config concepts (registration fee, auction duration floor/ceiling, in
// hours); commissionRate/commissionBase are the client-side commission display
// parameters (Your Bid = commissionBase + (commissionRate+100)*bid/100).
const (
	DefaultItcListingFee      uint32 = 5000 // m_nRegisterFeeMeso  (0x1388)
	DefaultItcCommissionRate  uint32 = 7    // m_nCommissionRate
	DefaultItcCommissionBase  uint32 = 500  // m_nCommissionBase   (0x1F4)
	DefaultItcAuctionMinHours uint32 = 24   // m_nAuctionDurationMin (0x18)
	DefaultItcAuctionMaxHours uint32 = 168  // m_nAuctionDurationMax (0xA8)
)

// SetItc is the SET_ITC clientbound writer. Body = CharacterData block +
// account name (ZXString) + 5×int32 ITC config + 8-byte server-now FILETIME.
type SetItc struct {
	characterData   charpkt.CharacterData
	accountName     string
	listingFee      uint32
	commissionRate  uint32
	commissionBase  uint32
	auctionMinHours uint32
	auctionMaxHours uint32
	serverTime      [8]byte
}

// NewSetItc builds a SET_ITC packet with the interim ITC config fallback
// defaults. serverTime is the server's current time as an 8-byte FILETIME
// (MsTimeBytes(time.Now())) — it seeds the client's ITC clock so auction
// countdowns are correct (see the SetItcWriter doc). Callers with per-tenant MTS
// config can use NewSetItcWithConfig instead.
func NewSetItc(characterData charpkt.CharacterData, accountName string, serverTime [8]byte) SetItc {
	return SetItc{
		characterData:   characterData,
		accountName:     accountName,
		listingFee:      DefaultItcListingFee,
		commissionRate:  DefaultItcCommissionRate,
		commissionBase:  DefaultItcCommissionBase,
		auctionMinHours: DefaultItcAuctionMinHours,
		auctionMaxHours: DefaultItcAuctionMaxHours,
		serverTime:      serverTime,
	}
}

// NewSetItcWithConfig builds a SET_ITC packet with explicit ITC config values
// (e.g. from an MTS config resource) and the server-now FILETIME.
func NewSetItcWithConfig(characterData charpkt.CharacterData, accountName string, listingFee, commissionRate, commissionBase, auctionMinHours, auctionMaxHours uint32, serverTime [8]byte) SetItc {
	return SetItc{
		characterData:   characterData,
		accountName:     accountName,
		listingFee:      listingFee,
		commissionRate:  commissionRate,
		commissionBase:  commissionBase,
		auctionMinHours: auctionMinHours,
		auctionMaxHours: auctionMaxHours,
		serverTime:      serverTime,
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
		w.WriteByteArray(m.serverTime[:])                         // DecodeBuffer(8) ftSvr (server-now)
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
		copy(m.serverTime[:], r.ReadBytes(8))
	}
}

func (m SetItc) CharacterData() charpkt.CharacterData { return m.characterData }
func (m SetItc) AccountName() string                  { return m.accountName }
func (m SetItc) ListingFee() uint32                   { return m.listingFee }
func (m SetItc) CommissionRate() uint32               { return m.commissionRate }
func (m SetItc) CommissionBase() uint32               { return m.commissionBase }
func (m SetItc) AuctionMinHours() uint32              { return m.auctionMinHours }
func (m SetItc) AuctionMaxHours() uint32              { return m.auctionMaxHours }
func (m SetItc) ServerTime() [8]byte                  { return m.serverTime }
