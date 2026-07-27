package character

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type CharacterStats struct {
	Id         uint32
	Name       string // max 13 chars, padded with zeros
	Gender     byte
	SkinColor  byte
	Face       uint32
	Hair       uint32
	PetIds     [3]uint64
	Level      byte
	JobId      uint16
	Str        uint16
	Dex        uint16
	Int        uint16
	Luk        uint16
	Hp         uint16
	MaxHp      uint16
	Mp         uint16
	MaxMp      uint16
	Ap         uint16
	Sp         uint16
	Exp        uint32
	Fame       int16
	GachaExp   uint32
	MapId      uint32
	SpawnPoint byte
}

type InventoryData struct {
	EquipCapacity byte
	UseCapacity   byte
	SetupCapacity byte
	EtcCapacity   byte
	CashCapacity  byte
	Timestamp     int64
	RegularEquip  []model.Asset
	CashEquip     []model.Asset
	EquipInv      []model.Asset
	UseInv        []model.Asset
	SetupInv      []model.Asset
	EtcInv        []model.Asset
	CashInv       []model.Asset
}

type SkillEntry struct {
	Id          uint32
	Level       uint32
	Expiration  int64
	MasterLevel uint32
	FourthJob   bool
}

type CooldownEntry struct {
	SkillId   uint32
	Remaining uint16
}

type QuestProgress struct {
	QuestId  uint16
	Progress string
}

type QuestCompleted struct {
	QuestId     uint16
	CompletedAt int64
}

// MonsterBookCard is a single owned monster-book card and its level, as
// carried in the CharacterData login packet.
type MonsterBookCard struct {
	CardId item.Id
	Level  byte
}

// MonsterBookData is the player's monster-book state for the login window:
// the chosen cover (full item id, 0 if none) and the full owned-card list.
type MonsterBookData struct {
	CoverCardId item.Id
	Cards       []MonsterBookCard
}

type CharacterData struct {
	Stats           CharacterStats
	BuddyCapacity   byte
	Meso            uint32
	Inventory       InventoryData
	Skills          []SkillEntry
	Cooldowns       []CooldownEntry
	StartedQuests   []QuestProgress
	CompletedQuests []QuestCompleted
	MonsterBook     MonsterBookData
	// TeleportMaps / VipTeleportMaps are the saved teleport-rock lists
	// (regular: 5 slots, VIP: 10 slots). Encoding pads with EmptyMapId;
	// decoding strips the padding.
	TeleportMaps    []_map.Id
	VipTeleportMaps []_map.Id
}

func (m CharacterData) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		// dbcharFlag: widened from a 16-bit mask to a 64-bit mask in the v61
		// protocol revision. v48's CharacterData::Decode reads Decode2 (verified
		// @0x49d341, max bit 0x8000 → no monster-book/new-year/area sections);
		// v61+ read DecodeBuffer(8) (verified v61 @0x4b656d, v72 @0x4d1c80).
		if (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS" {
			w.WriteInt64(-1)
		} else {
			w.WriteInt16(-1)
		}
		// SN list size: added in the v79 protocol revision. Absent in v48/v61/v72,
		// whose CharacterData::Decode reads the 8-byte flag then goes straight to the
		// stat section (verified CStage::OnSetField v72 @0x6c0c9b / CharacterData::Decode
		// v72 @0x4d1c60 vs v79 @0x4d9b85, v83 @0x4e592d). Writing it pre-79 shifts the
		// whole stream by 1 byte (consumed as the first byte of the character id).
		if (t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS" {
			w.WriteByte(0) // SN list size
		}

		m.encodeStats(w, t)
		w.WriteByte(m.BuddyCapacity)

		// linked name: added in the v79 protocol revision (absent v48/v61/v72; v72
		// reads buddyCap then meso with no linked-name byte in between — DecodeMoney
		// v72 @0x4cf30d reads only meso). IDA-verified.
		if (t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS" {
			w.WriteByte(0) // not linked
		}
		w.WriteInt(m.Meso)

		// JMS extra
		if t.Region() == "JMS" {
			w.WriteInt(m.Stats.Id)
			w.WriteInt(0) // dama
			w.WriteInt(0)
		}

		m.encodeInventory(l, ctx, options, w)
		m.encodeSkills(w, t)
		m.encodeQuests(w, t)
		m.encodeMiniGame(w)
		m.encodeRings(w, t)
		m.encodeTeleports(w, t)
		if t.Region() == "JMS" {
			w.WriteShort(0)
		}

		// Monster book. The COVER (flag 0x20000) arrived in the v61 revision; the
		// CARD list (flag 0x10000) arrived in the v72 revision; both are gone by
		// GMS v95+. v48's 16-bit dbcharFlag cannot even express these bits, so it
		// has no monster book at all (verified v48 CharacterData::Decode @0x49d320
		// ends at teleport rocks; v61 @0x4b654d has cover only @0x4b70fd; v72
		// @0x4d1c60 has cover @0x4d2845 + cards @0x4d2869). Absent in GMS v95+.
		if (t.IsRegion("GMS") && t.MajorAtLeast(61) && t.MajorVersion() <= 87) || t.Region() == "JMS" {
			m.encodeMonsterBookCover(w)
		}
		if (t.IsRegion("GMS") && t.MajorAtLeast(72) && t.MajorVersion() <= 87) || t.Region() == "JMS" {
			m.encodeMonsterBookCards(w)
		}
		// New-year cards / area popup / trailing short — v72 revision (flags
		// 0x40000/0x80000/0x100000). Absent in v48/v61 (verified: v61 ends at the
		// monster-book cover; v48's flag cannot express these bits).
		if (t.IsRegion("GMS") && t.MajorAtLeast(72)) || t.Region() == "JMS" {
			if t.Region() == "GMS" {
				m.encodeNewYear(w)
				m.encodeArea(w)
			} else if t.Region() == "JMS" {
				w.WriteShort(0)
			}
			w.WriteShort(0)
		}
		return w.Bytes()
	}
}

func (m *CharacterData) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)

		// dbcharFlag: Int16 pre-v61, Int64 v61+ (mirror of Encode).
		if (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS" {
			_ = r.ReadInt64()
		} else {
			_ = r.ReadInt16()
		}
		// SN list size: v79+ only (mirror of Encode).
		if (t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS" {
			_ = r.ReadByte() // SN list size
		}

		m.decodeStats(r, t)
		m.BuddyCapacity = r.ReadByte()

		// linked name: v79+ only (mirror of Encode).
		if (t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS" {
			linked := r.ReadByte()
			if linked == 1 {
				_ = r.ReadAsciiString()
			}
		}
		m.Meso = r.ReadUint32()

		// JMS extra
		if t.Region() == "JMS" {
			_ = r.ReadUint32() // characterId
			_ = r.ReadUint32() // dama
			_ = r.ReadUint32()
		}

		m.decodeInventory(l, ctx, r, options)
		m.decodeSkills(r, t)
		m.decodeQuests(r, t)
		m.decodeMiniGame(r)
		m.decodeRings(r, t)
		m.decodeTeleports(r, t)
		if t.Region() == "JMS" {
			_ = r.ReadUint16()
		}

		// Monster book cover (v61+) and cards (v72+); both gone by GMS v95+. Mirror of Encode.
		if (t.IsRegion("GMS") && t.MajorAtLeast(61) && t.MajorVersion() <= 87) || t.Region() == "JMS" {
			m.decodeMonsterBookCover(r)
		}
		if (t.IsRegion("GMS") && t.MajorAtLeast(72) && t.MajorVersion() <= 87) || t.Region() == "JMS" {
			m.decodeMonsterBookCards(r)
		}
		// New-year cards / area popup / trailing short — v72+ (mirror of Encode).
		if (t.IsRegion("GMS") && t.MajorAtLeast(72)) || t.Region() == "JMS" {
			if t.Region() == "GMS" {
				m.decodeNewYear(r)
				m.decodeArea(r)
			} else if t.Region() == "JMS" {
				_ = r.ReadUint16()
			}
			_ = r.ReadUint16()
		}
	}
}

// Stats encoding/decoding

// isEvanJob reports whether the job uses the Evan extended-SP block (a per-master-level
// SP list) instead of a single SP short. Matches the v84 client (GW_CharacterStat::Decode):
// jobId == 2001 (Evan beginner) || jobId/100 == 22 (Evan growths 2200-2299).
func isEvanJob(jobId uint16) bool {
	return jobId == 2001 || jobId/100 == 22
}

func (m *CharacterData) encodeStats(w *response.Writer, t tenant.Model) {
	w.WriteInt(m.Stats.Id)

	name := m.Stats.Name
	if len(name) > 13 {
		name = name[:13]
	}
	padSize := 13 - len(name)
	w.WriteByteArray([]byte(name))
	for i := 0; i < padSize; i++ {
		w.WriteByte(0)
	}

	w.WriteByte(m.Stats.Gender)
	w.WriteByte(m.Stats.SkinColor)
	w.WriteInt(m.Stats.Face)
	w.WriteInt(m.Stats.Hair)

	// The character-stat pet-cash-id array widened from 1 slot to 3 in the v61
	// revision. v48 reads DecodeBuffer(8) = one long (verified GW_CharacterStat
	// @0x49b6bc); v61+ read DecodeBuffer(24) = three (verified v61 @0x4b4116,
	// v72 @0x4cf183).
	if (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS" {
		for i := 0; i < 3; i++ {
			w.WriteLong(m.Stats.PetIds[i])
		}
	} else {
		w.WriteLong(m.Stats.PetIds[0])
	}

	w.WriteByte(m.Stats.Level)
	w.WriteShort(m.Stats.JobId)
	w.WriteShort(m.Stats.Str)
	w.WriteShort(m.Stats.Dex)
	w.WriteShort(m.Stats.Int)
	w.WriteShort(m.Stats.Luk)
	w.WriteShort(m.Stats.Hp)
	w.WriteShort(m.Stats.MaxHp)
	w.WriteShort(m.Stats.Mp)
	w.WriteShort(m.Stats.MaxMp)
	w.WriteShort(m.Stats.Ap)
	if t.IsRegion("GMS") && t.MajorAtLeast(84) && isEvanJob(m.Stats.JobId) {
		// Evan extended SP: byte count + count×(masterLevelIdx, sp) byte-pairs
		// (GW_CharacterStat::DecodeExtendSP). 0 for a freshly-created Evan (no SP allocated).
		w.WriteByte(0)
	} else {
		w.WriteShort(m.Stats.Sp)
	}
	w.WriteInt(m.Stats.Exp)
	w.WriteInt16(m.Stats.Fame)

	// gachaExp: inserted before mapId in the v72 revision. Absent v48/v61, whose
	// stat tail after fame is just mapId(Decode4)+spawnPoint(Decode1) — v61
	// OnSetField @0x659fd3 uses stat+177 (the first post-fame int) as the map id;
	// v72 @0x4cf0ee reads gachaExp @+177 then mapId @+189. IDA-verified.
	if (t.IsRegion("GMS") && t.MajorAtLeast(72)) || t.Region() == "JMS" {
		w.WriteInt(m.Stats.GachaExp)
	}
	w.WriteInt(m.Stats.MapId)
	w.WriteByte(m.Stats.SpawnPoint)

	if t.Region() == "GMS" {
		// Trailing stat int added in the v72 revision (v72 stat ends spawn+Decode4
		// @+207; v48/v61 stat ends at spawnPoint). v12-and-older wrote a wider
		// legacy block. IDA-verified.
		if t.MajorAtLeast(72) {
			w.WriteInt(0)
		} else if t.MajorVersion() <= 12 {
			w.WriteInt64(0)
			w.WriteInt(0)
			w.WriteInt(0)
		}
		if t.MajorVersion() >= 87 {
			w.WriteShort(0) // nSubJob
		}
	} else if t.Region() == "JMS" {
		w.WriteShort(0)
		w.WriteLong(0)
		w.WriteInt(0)
		w.WriteInt(0)
		w.WriteInt(0)
	}
}

func (m *CharacterData) decodeStats(r *request.Reader, t tenant.Model) {
	m.Stats.Id = r.ReadUint32()

	nameBytes := r.ReadBytes(13)
	end := 13
	for i, b := range nameBytes {
		if b == 0 {
			end = i
			break
		}
	}
	m.Stats.Name = string(nameBytes[:end])

	m.Stats.Gender = r.ReadByte()
	m.Stats.SkinColor = r.ReadByte()
	m.Stats.Face = r.ReadUint32()
	m.Stats.Hair = r.ReadUint32()

	// Pet-cash-id array: 1 slot pre-v61, 3 slots v61+ (mirror of Encode).
	if (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS" {
		for i := 0; i < 3; i++ {
			m.Stats.PetIds[i] = r.ReadUint64()
		}
	} else {
		m.Stats.PetIds[0] = r.ReadUint64()
	}

	m.Stats.Level = r.ReadByte()
	m.Stats.JobId = r.ReadUint16()
	m.Stats.Str = r.ReadUint16()
	m.Stats.Dex = r.ReadUint16()
	m.Stats.Int = r.ReadUint16()
	m.Stats.Luk = r.ReadUint16()
	m.Stats.Hp = r.ReadUint16()
	m.Stats.MaxHp = r.ReadUint16()
	m.Stats.Mp = r.ReadUint16()
	m.Stats.MaxMp = r.ReadUint16()
	m.Stats.Ap = r.ReadUint16()
	if t.IsRegion("GMS") && t.MajorAtLeast(84) && isEvanJob(m.Stats.JobId) {
		// Evan extended SP (mirror of Encode): byte count + count×(masterLevelIdx, sp).
		count := r.ReadByte()
		for i := byte(0); i < count; i++ {
			_ = r.ReadByte() // master-level index
			_ = r.ReadByte() // sp
		}
	} else {
		m.Stats.Sp = r.ReadUint16()
	}
	m.Stats.Exp = r.ReadUint32()
	m.Stats.Fame = r.ReadInt16()

	// gachaExp: v72+ only (mirror of Encode).
	if (t.IsRegion("GMS") && t.MajorAtLeast(72)) || t.Region() == "JMS" {
		m.Stats.GachaExp = r.ReadUint32()
	}
	m.Stats.MapId = r.ReadUint32()
	m.Stats.SpawnPoint = r.ReadByte()

	if t.Region() == "GMS" {
		// Trailing stat int: v72+ (mirror of Encode); v12-and-older legacy block.
		if t.MajorAtLeast(72) {
			_ = r.ReadUint32()
		} else if t.MajorVersion() <= 12 {
			_ = r.ReadInt64()
			_ = r.ReadUint32()
			_ = r.ReadUint32()
		}
		if t.MajorVersion() >= 87 {
			_ = r.ReadUint16() // nSubJob
		}
	} else if t.Region() == "JMS" {
		_ = r.ReadUint16()
		_ = r.ReadUint64()
		_ = r.ReadUint32()
		_ = r.ReadUint32()
		_ = r.ReadUint32()
	}
}

// Inventory encoding/decoding

func (m *CharacterData) encodeInventory(l logrus.FieldLogger, ctx context.Context, options map[string]interface{}, w *response.Writer) {
	t := tenant.MustFromContext(ctx)

	if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
		w.WriteByte(m.Inventory.EquipCapacity)
		w.WriteByte(m.Inventory.UseCapacity)
		w.WriteByte(m.Inventory.SetupCapacity)
		w.WriteByte(m.Inventory.EtcCapacity)
		w.WriteByte(m.Inventory.CashCapacity)
	}

	// Inventory-update FILETIME: added in the v79 protocol revision (flag 0x100000,
	// read before the equip section). Absent v48/v61/v72 — v72 has no 0x100000 block
	// before equipment (its only 0x100000 use is the trailing wishlist map). IDA-verified.
	if (t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS" {
		w.WriteInt64(m.Inventory.Timestamp)
	}

	// Regular equipment
	for i := range m.Inventory.RegularEquip {
		w.WriteByteArray(m.Inventory.RegularEquip[i].Encode(l, ctx)(options))
	}
	// Equip-section terminator width tracks the equip slot width: short for
	// GMS>=83/JMS, byte for legacy GMS (<83). See model.Asset.encodeSlot.
	if (t.Region() == "GMS" && t.MajorAtLeast(83)) || t.Region() == "JMS" {
		w.WriteShort(0)
	} else {
		w.WriteByte(0)
	}

	// Cash equipment
	for i := range m.Inventory.CashEquip {
		w.WriteByteArray(m.Inventory.CashEquip[i].Encode(l, ctx)(options))
	}
	if (t.Region() == "GMS" && t.MajorAtLeast(83)) || t.Region() == "JMS" {
		w.WriteShort(0)
	} else {
		w.WriteByte(0)
	}

	// Equipable inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		w.WriteByte(m.Inventory.EquipCapacity)
	}
	for i := range m.Inventory.EquipInv {
		w.WriteByteArray(m.Inventory.EquipInv[i].Encode(l, ctx)(options))
	}
	// GMS>=83/JMS fold the empty 4th (dragon/mechanic) equip loop terminator
	// into this Int(0) (two short terminators). Legacy GMS (<83) has no such
	// loop and terminates the equipable inventory with a single byte.
	if (t.Region() == "GMS" && t.MajorAtLeast(83)) || t.Region() == "JMS" {
		w.WriteInt(0)
	} else {
		w.WriteByte(0)
	}

	// Use inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		w.WriteByte(m.Inventory.UseCapacity)
	}
	for i := range m.Inventory.UseInv {
		w.WriteByteArray(m.Inventory.UseInv[i].Encode(l, ctx)(options))
	}
	w.WriteByte(0)

	// Setup inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		w.WriteByte(m.Inventory.SetupCapacity)
	}
	for i := range m.Inventory.SetupInv {
		w.WriteByteArray(m.Inventory.SetupInv[i].Encode(l, ctx)(options))
	}
	w.WriteByte(0)

	// Etc inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		w.WriteByte(m.Inventory.EtcCapacity)
	}
	for i := range m.Inventory.EtcInv {
		w.WriteByteArray(m.Inventory.EtcInv[i].Encode(l, ctx)(options))
	}
	w.WriteByte(0)

	// Cash inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		w.WriteByte(m.Inventory.CashCapacity)
	}
	for i := range m.Inventory.CashInv {
		w.WriteByteArray(m.Inventory.CashInv[i].Encode(l, ctx)(options))
	}
	w.WriteByte(0)
}

func (m *CharacterData) decodeInventory(l logrus.FieldLogger, ctx context.Context, r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)

	if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
		m.Inventory.EquipCapacity = r.ReadByte()
		m.Inventory.UseCapacity = r.ReadByte()
		m.Inventory.SetupCapacity = r.ReadByte()
		m.Inventory.EtcCapacity = r.ReadByte()
		m.Inventory.CashCapacity = r.ReadByte()
	}

	// Inventory-update FILETIME: v79+ only (mirror of Encode).
	if (t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS" {
		m.Inventory.Timestamp = r.ReadInt64()
	}

	// Regular equipment: slot is negative (equipped items)
	m.Inventory.RegularEquip = decodeEquipmentSection(l, ctx, r, options, t, false)

	// Cash equipment: slot is negative with -100 offset
	m.Inventory.CashEquip = decodeEquipmentSection(l, ctx, r, options, t, true)

	// Equipable inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		m.Inventory.EquipCapacity = r.ReadByte()
	}
	m.Inventory.EquipInv = decodeEquipableInventorySection(l, ctx, r, options, t)

	// Use inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		m.Inventory.UseCapacity = r.ReadByte()
	}
	m.Inventory.UseInv = decodeStackableSection(l, ctx, r, options)

	// Setup inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		m.Inventory.SetupCapacity = r.ReadByte()
	}
	m.Inventory.SetupInv = decodeStackableSection(l, ctx, r, options)

	// Etc inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		m.Inventory.EtcCapacity = r.ReadByte()
	}
	m.Inventory.EtcInv = decodeStackableSection(l, ctx, r, options)

	// Cash inventory
	if t.Region() == "GMS" && t.MajorVersion() < 28 {
		m.Inventory.CashCapacity = r.ReadByte()
	}
	m.Inventory.CashInv = decodeStackableSection(l, ctx, r, options)
}

func decodeEquipmentSection(l logrus.FieldLogger, ctx context.Context, r *request.Reader, options map[string]interface{}, t tenant.Model, cash bool) []model.Asset {
	var assets []model.Asset
	for {
		var wireSlot uint16
		if (t.Region() == "GMS" && t.MajorAtLeast(83)) || t.Region() == "JMS" {
			wireSlot = r.ReadUint16()
		} else {
			wireSlot = uint16(r.ReadByte())
		}
		if wireSlot == 0 {
			break
		}
		var slot int16
		if cash {
			slot = -int16(wireSlot) - 100
		} else {
			slot = -int16(wireSlot)
		}
		a := model.NewAsset(false, slot, 0, model.FromMsTime(0))
		a.Decode(l, ctx)(r, options)
		assets = append(assets, a)
	}
	return assets
}

func decodeEquipableInventorySection(l logrus.FieldLogger, ctx context.Context, r *request.Reader, options map[string]interface{}, t tenant.Model) []model.Asset {
	var assets []model.Asset
	for {
		if (t.Region() == "GMS" && t.MajorAtLeast(83)) || t.Region() == "JMS" {
			wireSlot := r.ReadUint16()
			if wireSlot == 0 {
				_ = r.ReadUint16() // consume remaining 2 bytes of WriteInt(0) terminator
				break
			}
			a := model.NewAsset(false, int16(wireSlot), 0, model.FromMsTime(0))
			a.Decode(l, ctx)(r, options)
			assets = append(assets, a)
		} else {
			wireSlot := r.ReadByte()
			if wireSlot == 0 {
				break
			}
			a := model.NewAsset(false, int16(wireSlot), 0, model.FromMsTime(0))
			a.Decode(l, ctx)(r, options)
			assets = append(assets, a)
		}
	}
	return assets
}

func decodeStackableSection(l logrus.FieldLogger, ctx context.Context, r *request.Reader, options map[string]interface{}) []model.Asset {
	var assets []model.Asset
	for {
		wireSlot := r.ReadInt8()
		if wireSlot == 0 {
			break
		}
		a := model.NewAsset(false, int16(wireSlot), 0, model.FromMsTime(0))
		a.Decode(l, ctx)(r, options)
		assets = append(assets, a)
	}
	return assets
}

// Skills encoding/decoding

func (m *CharacterData) encodeSkills(w *response.Writer, t tenant.Model) {
	w.WriteShort(uint16(len(m.Skills)))
	for _, s := range m.Skills {
		w.WriteInt(s.Id)
		w.WriteInt(s.Level)
		// Per-skill expiration (Int64) was introduced between v79 and v83.
		// v79 client GW skill decode reads id+level(+mastery) only (verified
		// CharacterData::Decode v79 @0x4da2ca); v83 adds DecodeBuffer(8)
		// (verified @0x4e592d). Writing it ungated shifts every later section
		// for v79 and over-reads at GW_CoupleRecord::Decode (error 38).
		if (t.Region() == "GMS" && t.MajorAtLeast(83)) || t.Region() == "JMS" {
			w.WriteInt64(s.Expiration)
		}
		if s.FourthJob {
			w.WriteInt(s.MasterLevel)
		}
	}

	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		w.WriteShort(uint16(len(m.Cooldowns)))
		for _, cd := range m.Cooldowns {
			w.WriteInt(cd.SkillId)
			w.WriteShort(cd.Remaining)
		}
	}
}

func (m *CharacterData) decodeSkills(r *request.Reader, t tenant.Model) {
	skillCount := r.ReadUint16()
	m.Skills = make([]SkillEntry, skillCount)
	for i := uint16(0); i < skillCount; i++ {
		m.Skills[i].Id = r.ReadUint32()
		m.Skills[i].Level = r.ReadUint32()
		// Mirror of encodeSkills: expiration present only for GMS >= 83 / JMS.
		if (t.Region() == "GMS" && t.MajorAtLeast(83)) || t.Region() == "JMS" {
			m.Skills[i].Expiration = r.ReadInt64()
		}
		jobId := job.IdFromSkillId(skill.Id(m.Skills[i].Id))
		m.Skills[i].FourthJob = job.IsFourthJob(jobId)
		if m.Skills[i].FourthJob {
			m.Skills[i].MasterLevel = r.ReadUint32()
		}
	}

	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		cdCount := r.ReadUint16()
		m.Cooldowns = make([]CooldownEntry, cdCount)
		for i := uint16(0); i < cdCount; i++ {
			m.Cooldowns[i].SkillId = r.ReadUint32()
			m.Cooldowns[i].Remaining = r.ReadUint16()
		}
	}
}

// Quest encoding/decoding

func (m *CharacterData) encodeQuests(w *response.Writer, t tenant.Model) {
	w.WriteShort(uint16(len(m.StartedQuests)))
	for _, q := range m.StartedQuests {
		w.WriteShort(q.QuestId)
		w.WriteAsciiString(q.Progress)
	}

	if t.Region() == "JMS" {
		w.WriteShort(0)
	}

	if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
		w.WriteShort(uint16(len(m.CompletedQuests)))
		for _, q := range m.CompletedQuests {
			w.WriteShort(q.QuestId)
			w.WriteInt64(q.CompletedAt)
		}
	}
}

func (m *CharacterData) decodeQuests(r *request.Reader, t tenant.Model) {
	startedCount := r.ReadUint16()
	m.StartedQuests = make([]QuestProgress, startedCount)
	for i := uint16(0); i < startedCount; i++ {
		m.StartedQuests[i].QuestId = r.ReadUint16()
		m.StartedQuests[i].Progress = r.ReadAsciiString()
	}

	if t.Region() == "JMS" {
		_ = r.ReadUint16()
	}

	if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
		completedCount := r.ReadUint16()
		m.CompletedQuests = make([]QuestCompleted, completedCount)
		for i := uint16(0); i < completedCount; i++ {
			m.CompletedQuests[i].QuestId = r.ReadUint16()
			m.CompletedQuests[i].CompletedAt = r.ReadInt64()
		}
	}
}

// Zero-value sections — these always write fixed zero patterns

func (m *CharacterData) encodeMiniGame(w *response.Writer) {
	w.WriteShort(0)
}

func (m *CharacterData) decodeMiniGame(r *request.Reader) {
	_ = r.ReadUint16()
}

func (m *CharacterData) encodeRings(w *response.Writer, t tenant.Model) {
	w.WriteShort(0) // crush rings
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		w.WriteShort(0) // friendship rings
		w.WriteShort(0) // partner
	}
}

func (m *CharacterData) decodeRings(r *request.Reader, t tenant.Model) {
	_ = r.ReadUint16() // crush rings
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		_ = r.ReadUint16() // friendship rings
		_ = r.ReadUint16() // partner
	}
}

func (m *CharacterData) encodeTeleports(w *response.Writer, t tenant.Model) {
	for i := 0; i < 5; i++ {
		v := _map.EmptyMapId
		if i < len(m.TeleportMaps) {
			v = m.TeleportMaps[i]
		}
		w.WriteInt(uint32(v))
	}
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		for i := 0; i < 10; i++ {
			v := _map.EmptyMapId
			if i < len(m.VipTeleportMaps) {
				v = m.VipTeleportMaps[i]
			}
			w.WriteInt(uint32(v))
		}
	}
}

func (m *CharacterData) decodeTeleports(r *request.Reader, t tenant.Model) {
	for i := 0; i < 5; i++ {
		v := _map.Id(r.ReadUint32())
		if v != _map.EmptyMapId {
			m.TeleportMaps = append(m.TeleportMaps, v)
		}
	}
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		for i := 0; i < 10; i++ {
			v := _map.Id(r.ReadUint32())
			if v != _map.EmptyMapId {
				m.VipTeleportMaps = append(m.VipTeleportMaps, v)
			}
		}
	}
}

// encodeMonsterBookCover writes the monster-book cover card id (flag 0x20000),
// introduced in the v61 revision.
func (m *CharacterData) encodeMonsterBookCover(w *response.Writer) {
	w.WriteInt(uint32(m.MonsterBook.CoverCardId)) // cover: full item id (flag 0x20000)
}

// encodeMonsterBookCards writes the owned-card list (flag 0x10000), introduced in
// the v72 revision. Always mode 0 (simple list).
func (m *CharacterData) encodeMonsterBookCards(w *response.Writer) {
	w.WriteByte(0) // mode 0: simple list (flag 0x10000)
	w.WriteShort(uint16(len(m.MonsterBook.Cards)))
	for _, c := range m.MonsterBook.Cards {
		w.WriteShort(uint16(uint32(c.CardId) - uint32(item.MonsterBookCardBase)))
		w.WriteByte(c.Level)
	}
}

func (m *CharacterData) decodeMonsterBookCover(r *request.Reader) {
	m.MonsterBook.CoverCardId = item.Id(r.ReadUint32())
}

// decodeMonsterBookCards is the symmetric reader for atlas's own mode-0 output.
// The server only ever emits mode 0, so only mode 0 is decoded (the client-side
// mode-1 bitmap form is never produced here).
func (m *CharacterData) decodeMonsterBookCards(r *request.Reader) {
	_ = r.ReadByte() // mode selector (always 0 on the wire we emit)
	count := r.ReadUint16()
	m.MonsterBook.Cards = make([]MonsterBookCard, count)
	for i := uint16(0); i < count; i++ {
		m.MonsterBook.Cards[i].CardId = item.MonsterBookCardBase + item.Id(r.ReadUint16())
		m.MonsterBook.Cards[i].Level = r.ReadByte()
	}
}

func (m *CharacterData) encodeNewYear(w *response.Writer) {
	w.WriteShort(0)
}

func (m *CharacterData) decodeNewYear(r *request.Reader) {
	_ = r.ReadUint16()
}

func (m *CharacterData) encodeArea(w *response.Writer) {
	w.WriteShort(0)
}

func (m *CharacterData) decodeArea(r *request.Reader) {
	_ = r.ReadUint16()
}
