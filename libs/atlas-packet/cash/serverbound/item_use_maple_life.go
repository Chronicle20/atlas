package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseMapleLife is the sub-body of the cash ItemUse packet for the
// character-creation ("Maple Life", `Cash/0543` ids `05431000`/`05432000`)
// slot purchase — CUICharacterSaleDlg::SendCreateNewCharacter. Like its
// item_use_*.go siblings this struct carries only the fields AFTER the
// shared ItemUse header's nPOS/nItemID pair (decoded separately by that
// struct); the caller composes ItemUse's header decode with this sub-body's
// decode from the same reader, per the character_cash_item_use.go dispatch
// pattern.
//
// Field set and order — derivation.md §2, decompiled/raw-disassembly
// confirmed on gms_v83 (0x7d7960), gms_v87 (0x82e402), gms_v92 (0x758770),
// gms_v95 (0x77a240); gms_v84 is VERSION-ABSENT (derivation.md §2.0 — no
// CUICharacterSaleDlg code path exists on that binary at all):
//
//	EncodeStr  sName
//	Encode4    al[0]   // CUICharacterSaleDlg's per-slot avatar-look selection
//	Encode4    al[1]
//	Encode4    al[2]
//	Encode4    al[3]
//	Encode4    nGender
//	Encode4    nCurrentClass
//	Encode4    nSP
//	Encode4    update_time  // TRAILING — see doc note below
//
// # The doubled update_time write
//
// Every sibling item_use_*.go sub-body models its trailing update_time with
// item_use_incubator.go's `if !updateTimeFirst { write }` convention: on
// GMS <= v83 the shared ItemUse header omits the leading copy, so the
// sub-body supplies the packet's ONLY update_time; on GMS >= v87 the header
// already wrote it, so the sub-body omits it and there is exactly one
// occurrence total.
//
// SendCreateNewCharacter does NOT follow that convention. Raw disassembly
// (not decompiler pseudocode) confirms two INDEPENDENT `get_update_time()`
// call + `Encode4` sites in every version's function body:
//
//	gms_v83 (0x7d7960): ONE call site, 0x7d7a4d (`call get_update_time`) ->
//	0x7d7a56 (`call Encode4`), immediately before SendRequest. No leading
//	copy exists anywhere earlier in the function (COutPacket ctor at
//	0x7d79c1 is followed directly by Encode2 nPOS at 0x7d79d6 — no
//	intervening update_time write). One occurrence, TRAILING, matches the
//	shared header's UpdateTimeFirst()==false on v83 (single total write).
//
//	gms_v87 (0x82e402): TWO independent call sites —
//	0x82e46b (`call get_update_time`) -> 0x82e474 (`call Encode4`), directly
//	after the COutPacket ctor and BEFORE Encode2 nPOS (the LEADING copy the
//	shared ItemUse header models); and a SECOND, separate
//	0x82e4fd (`call get_update_time`) -> 0x82e506 (`call Encode4`), after
//	nSP and before SendRequest (the TRAILING copy). Two distinct call
//	instructions to the same leaf function, two distinct Encode4 call
//	instructions — not a decompiler artifact of one call being rendered
//	twice.
//
//	gms_v92 (0x758770) and gms_v95 (0x77a240): decompile-confirmed same
//	shape as v87 — two separate `get_update_time`/`Encode4` (v95) or
//	`sub_936E80`/`Encode4` (v92, the unnamed but structurally-identical
//	get_update_time twin) pairs, one immediately after the COutPacket ctor,
//	one immediately before SendRequest.
//
// So on GMS >= v87 the packet carries update_time TWICE: once via the
// shared ItemUse header's leading write (cash/serverbound/item_use.go's
// UpdateTimeFirst gate), and — independently — once via THIS sub-body's own
// trailing write. The existing single-field `!updateTimeFirst`-gated
// convention cannot model that: it assumes the header and the sub-body
// never both carry update_time. This struct therefore does NOT gate its
// trailing update_time write on updateTimeFirst at all — it always writes
// (Encode)/consumes (Decode) a trailing update_time, on every version. The
// doubling on v87+ falls out naturally from composing this struct's
// unconditional trailing write with the header's unconditional (on v87+)
// leading write; v83's single total occurrence falls out from the header
// omitting its leading write (UpdateTimeFirst()==false) while this struct
// still writes its own.
//
// updateTimeFirst is still accepted by the constructor, for signature
// parity with every sibling item_use_*.go sub-body (callers already compute
// cashsb.UpdateTimeFirst(t) once per dispatch and thread it through), but it
// is not read anywhere in Encode/Decode — kept only as call-site
// documentation of which header shape the caller is composing this
// sub-body with.
//
// Derivation: docs/tasks/task-246-maple-life-character-creation/derivation.md §2.
// packet-audit:fname CUICharacterSaleDlg::SendCreateNewCharacter
type ItemUseMapleLife struct {
	updateTimeFirst bool
	name            string
	al0             int32
	al1             int32
	al2             int32
	al3             int32
	gender          int32
	currentClass    int32
	sp              int32
	updateTime      uint32
}

func NewItemUseMapleLife(updateTimeFirst bool) *ItemUseMapleLife {
	return &ItemUseMapleLife{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseMapleLife) Name() string        { return m.name }
func (m ItemUseMapleLife) AL0() int32          { return m.al0 }
func (m ItemUseMapleLife) AL1() int32          { return m.al1 }
func (m ItemUseMapleLife) AL2() int32          { return m.al2 }
func (m ItemUseMapleLife) AL3() int32          { return m.al3 }
func (m ItemUseMapleLife) Gender() int32       { return m.gender }
func (m ItemUseMapleLife) CurrentClass() int32 { return m.currentClass }
func (m ItemUseMapleLife) SP() int32           { return m.sp }
func (m ItemUseMapleLife) UpdateTime() uint32  { return m.updateTime }
func (m ItemUseMapleLife) Operation() string   { return "ItemUseMapleLife" }

func (m ItemUseMapleLife) String() string {
	return fmt.Sprintf("name [%s], al [%d %d %d %d], gender [%d], currentClass [%d], sp [%d], updateTime [%d]",
		m.name, m.al0, m.al1, m.al2, m.al3, m.gender, m.currentClass, m.sp, m.updateTime)
}

func (m ItemUseMapleLife) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		w.WriteInt32(m.al0)
		w.WriteInt32(m.al1)
		w.WriteInt32(m.al2)
		w.WriteInt32(m.al3)
		w.WriteInt32(m.gender)
		w.WriteInt32(m.currentClass)
		w.WriteInt32(m.sp)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ItemUseMapleLife) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
		m.al0 = r.ReadInt32()
		m.al1 = r.ReadInt32()
		m.al2 = r.ReadInt32()
		m.al3 = r.ReadInt32()
		m.gender = r.ReadInt32()
		m.currentClass = r.ReadInt32()
		m.sp = r.ReadInt32()
		m.updateTime = r.ReadUint32()
	}
}
