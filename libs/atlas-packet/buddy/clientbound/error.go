package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// BuddyOperationWriter mirrors the parent-package buddy.BuddyOperationWriter
// ("BuddyOperation"). Emission always passes the PARENT const to
// session.Announce(...); these structs' Operation() return is COSMETIC (debug
// logging only, never used for emission/resolution). The clientbound package
// CANNOT import the parent buddy package (import cycle), so the const is
// redeclared here — same pattern as party/clientbound.PartyOperationWriter.
const BuddyOperationWriter = "BuddyOperation"

// --- OnFriendResult error/notice arms (discrete per mode; Task-4 AP split) ----
//
// The shared buddy/clientbound.Error catch-all is gone: every enumerated
// OnFriendResult error/notice arm now has its OWN discrete struct so it maps
// 1:1 to a fixed operation key and a single dispatcher #-entry (mirrors the
// task-103 guild and task-105 party OnPartyResult splits). Mode bytes are
// resolved at emit time via the config "operations" table — NEVER literal here.
//
// The OnFriendResult switch is BYTE-IDENTICAL across ALL FIVE versions for every
// arm below (unlike OnGuildResult/OnPartyResult, the buddy mode table is NOT
// shifted in v95). v83==v84==v87==v95==jms. Per-version OnFriendResult addrs
// (IDA-verified, task-105 Task 1): gms_v83 0xa3f2e8, gms_v84 0xa8ada2,
// gms_v87 0xad7ae5, gms_v95 0xa12630, jms_v185 0xb2a873. See
// docs/packets/dispatchers/buddy.yaml.
//
// NOTE: UNKNOWN_1 (mode 10) and UNKNOWN_2 (mode 18) are NOT error arms — they
// share the CFriend::Reset list-reset handler (same shape as UPDATE) and get NO
// discrete error struct here.

// --- Mode-only error arms -----------------------------------------------------

// ListFull — BUDDY_LIST_FULL (mode 11, all 5 versions). Mode-only: the
// OnFriendResult arm shows a StringPool notice, no Decode after the mode byte.
// packet-audit:fname CWvsContext::OnFriendResult#ListFull
type ListFull struct{ mode byte }

func NewListFull(mode byte) ListFull { return ListFull{mode: mode} }
func (m ListFull) Operation() string { return BuddyOperationWriter }
func (m ListFull) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m ListFull) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *ListFull) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// OtherListFull — OTHER_BUDDY_LIST_FULL (mode 12, all 5 versions). Mode-only.
// packet-audit:fname CWvsContext::OnFriendResult#OtherListFull
type OtherListFull struct{ mode byte }

func NewOtherListFull(mode byte) OtherListFull { return OtherListFull{mode: mode} }
func (m OtherListFull) Operation() string      { return BuddyOperationWriter }
func (m OtherListFull) String() string         { return fmt.Sprintf("mode [%d]", m.mode) }
func (m OtherListFull) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *OtherListFull) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// AlreadyBuddy — ALREADY_BUDDY (mode 13, all 5 versions). Mode-only.
// packet-audit:fname CWvsContext::OnFriendResult#AlreadyBuddy
type AlreadyBuddy struct{ mode byte }

func NewAlreadyBuddy(mode byte) AlreadyBuddy { return AlreadyBuddy{mode: mode} }
func (m AlreadyBuddy) Operation() string     { return BuddyOperationWriter }
func (m AlreadyBuddy) String() string        { return fmt.Sprintf("mode [%d]", m.mode) }
func (m AlreadyBuddy) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *AlreadyBuddy) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// CannotBuddyGm — CANNOT_BUDDY_GM (mode 14, all 5 versions). Mode-only.
// packet-audit:fname CWvsContext::OnFriendResult#CannotBuddyGm
type CannotBuddyGm struct{ mode byte }

func NewCannotBuddyGm(mode byte) CannotBuddyGm { return CannotBuddyGm{mode: mode} }
func (m CannotBuddyGm) Operation() string      { return BuddyOperationWriter }
func (m CannotBuddyGm) String() string         { return fmt.Sprintf("mode [%d]", m.mode) }
func (m CannotBuddyGm) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *CannotBuddyGm) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// CharacterNotFound — CHARACTER_NOT_FOUND (mode 15, all 5 versions). Mode-only.
// packet-audit:fname CWvsContext::OnFriendResult#CharacterNotFound
type CharacterNotFound struct{ mode byte }

func NewCharacterNotFound(mode byte) CharacterNotFound { return CharacterNotFound{mode: mode} }
func (m CharacterNotFound) Operation() string          { return BuddyOperationWriter }
func (m CharacterNotFound) String() string             { return fmt.Sprintf("mode [%d]", m.mode) }
func (m CharacterNotFound) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *CharacterNotFound) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// --- Extra-byte error arms (version-gated) ------------------------------------
//
// The UNKNOWN_ERROR family (modes 16/17/19/22 — GMS cases 0x10/0x11/0x13/0x16)
// share one GMS arm that FIRST reads `if (CInPacket::Decode1())` — a trailing
// extra byte — then shows a DecodeStr'd name (extra != 0) or a fixed StringPool
// notice (extra == 0). So in GMS (v83/v84/v87/v95) ALL FOUR write the extra
// byte (0 = no-name path). IN JMS (0xb2a873) the same four cases are MODE-ONLY:
// the jms arm goes straight to StringPool 765 + Notice with NO leading Decode1.
// The extra byte is therefore gated: emitted in GMS, omitted in JMS.
// (buddy.yaml; context.md §10.)

// UnknownError — UNKNOWN_ERROR (mode 16, all 5 versions). GMS reads a trailing
// Decode1; jms is mode-only (see family note above).
// packet-audit:fname CWvsContext::OnFriendResult#UnknownError
type UnknownError struct{ mode byte }

func NewUnknownError(mode byte) UnknownError { return UnknownError{mode: mode} }
func (m UnknownError) Operation() string     { return BuddyOperationWriter }
func (m UnknownError) String() string        { return fmt.Sprintf("mode [%d]", m.mode) }
func (m UnknownError) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	gms := t.IsRegion("GMS")
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		if gms {
			w.WriteByte(0) // GMS reads a trailing Decode1; 0 = no-name path. JMS is mode-only (buddy.yaml).
		}
		return w.Bytes()
	}
}
func (m *UnknownError) Decode(_ logrus.FieldLogger, ctx context.Context) func(*request.Reader, map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	gms := t.IsRegion("GMS")
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		if gms {
			_ = r.ReadByte()
		}
	}
}

// UnknownError2 — UNKNOWN_ERROR_2 (mode 17, all 5 versions). GMS reads a
// trailing Decode1 (case 0x11); jms is mode-only.
// packet-audit:fname CWvsContext::OnFriendResult#UnknownError2
type UnknownError2 struct{ mode byte }

func NewUnknownError2(mode byte) UnknownError2 { return UnknownError2{mode: mode} }
func (m UnknownError2) Operation() string      { return BuddyOperationWriter }
func (m UnknownError2) String() string         { return fmt.Sprintf("mode [%d]", m.mode) }
func (m UnknownError2) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	gms := t.IsRegion("GMS")
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		if gms {
			w.WriteByte(0) // GMS reads a trailing Decode1; 0 = no-name path. JMS is mode-only (buddy.yaml).
		}
		return w.Bytes()
	}
}
func (m *UnknownError2) Decode(_ logrus.FieldLogger, ctx context.Context) func(*request.Reader, map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	gms := t.IsRegion("GMS")
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		if gms {
			_ = r.ReadByte()
		}
	}
}

// UnknownError3 — UNKNOWN_ERROR_3 (mode 19, all 5 versions). GMS reads a
// trailing Decode1 (case 0x13); jms is mode-only.
// packet-audit:fname CWvsContext::OnFriendResult#UnknownError3
type UnknownError3 struct{ mode byte }

func NewUnknownError3(mode byte) UnknownError3 { return UnknownError3{mode: mode} }
func (m UnknownError3) Operation() string      { return BuddyOperationWriter }
func (m UnknownError3) String() string         { return fmt.Sprintf("mode [%d]", m.mode) }
func (m UnknownError3) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	gms := t.IsRegion("GMS")
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		if gms {
			w.WriteByte(0) // GMS reads a trailing Decode1; 0 = no-name path. JMS is mode-only (buddy.yaml).
		}
		return w.Bytes()
	}
}
func (m *UnknownError3) Decode(_ logrus.FieldLogger, ctx context.Context) func(*request.Reader, map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	gms := t.IsRegion("GMS")
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		if gms {
			_ = r.ReadByte()
		}
	}
}

// UnknownError4 — UNKNOWN_ERROR_4 (mode 22, all 5 versions). GMS reads a
// trailing Decode1 (case 0x16); jms is mode-only.
// packet-audit:fname CWvsContext::OnFriendResult#UnknownError4
type UnknownError4 struct{ mode byte }

func NewUnknownError4(mode byte) UnknownError4 { return UnknownError4{mode: mode} }
func (m UnknownError4) Operation() string      { return BuddyOperationWriter }
func (m UnknownError4) String() string         { return fmt.Sprintf("mode [%d]", m.mode) }
func (m UnknownError4) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	gms := t.IsRegion("GMS")
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		if gms {
			w.WriteByte(0) // GMS reads a trailing Decode1; 0 = no-name path. JMS is mode-only (buddy.yaml).
		}
		return w.Bytes()
	}
}
func (m *UnknownError4) Decode(_ logrus.FieldLogger, ctx context.Context) func(*request.Reader, map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	gms := t.IsRegion("GMS")
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		if gms {
			_ = r.ReadByte()
		}
	}
}
