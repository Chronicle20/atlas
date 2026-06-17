package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Discrete per-mode notice arms of the CTrunkDlg::OnPacket dispatcher (STORAGE
// op). Each of these error/notice modes consumes ONLY the leading mode byte off
// the wire — the dispatcher reads Decode1(mode), maps the code to a fixed
// StringPool message, and shows a CUtilDlg::Notice with NO further CInPacket
// reads. Each mode therefore has its OWN discrete struct that writes exactly
// that one mode byte (task-096: discrete-per-mode rule — no struct serves more
// than one mode; the former shared ErrorSimple shape is retired).
//
// The mode byte is RESOLVED per-tenant at encode time (the body function in
// storage/operation_body.go FIXES the operation KEY via WithResolvedCode and
// passes the resolved byte to the constructor). The KEY→byte mapping is
// version-shifted (gms_v83/v84/v87/v95 vs jms_v185 -1); see
// docs/packets/dispatchers/storage_operation.yaml. The wire shape (one byte) is
// identical across every version. Dispatchers: v83 0x7c8a4c, v84 0x7eec1a,
// v87 0x81c336, v95 0x76a990, jms 0x84e5a1.

// StorageErrorInventoryFull — the INVENTORY_FULL notice arm (gms mode 10 /
// jms 9). Decode1(mode) only; the client shows the storage-full StringPool
// notice with no further wire reads.
//
// packet-audit:fname CTrunkDlg::OnPacket#ErrorInventoryFull
type ErrorInventoryFull struct {
	mode byte
}

func NewStorageErrorInventoryFull(mode byte) ErrorInventoryFull {
	return ErrorInventoryFull{mode: mode}
}

func (m ErrorInventoryFull) Mode() byte        { return m.mode }
func (m ErrorInventoryFull) Operation() string { return StorageOperationWriter }
func (m ErrorInventoryFull) String() string {
	return fmt.Sprintf("storage error inventory full mode [%d]", m.mode)
}

func (m ErrorInventoryFull) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte; sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *ErrorInventoryFull) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// StorageErrorNotEnoughMesos — the NOT_ENOUGH_MESOS notice arm (gms mode 11 /
// jms 10). Decode1(mode) only; the client shows the not-enough-mesos StringPool
// notice with no further wire reads.
//
// packet-audit:fname CTrunkDlg::OnPacket#ErrorNotEnoughMesos
type ErrorNotEnoughMesos struct {
	mode byte
}

func NewStorageErrorNotEnoughMesos(mode byte) ErrorNotEnoughMesos {
	return ErrorNotEnoughMesos{mode: mode}
}

func (m ErrorNotEnoughMesos) Mode() byte        { return m.mode }
func (m ErrorNotEnoughMesos) Operation() string { return StorageOperationWriter }
func (m ErrorNotEnoughMesos) String() string {
	return fmt.Sprintf("storage error not enough mesos mode [%d]", m.mode)
}

func (m ErrorNotEnoughMesos) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte; sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *ErrorNotEnoughMesos) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// StorageErrorOneOfAKind — the ONE_OF_A_KIND notice arm (gms mode 12 / jms 11).
// Decode1(mode) only; the client shows the one-of-a-kind StringPool notice with
// no further wire reads.
//
// packet-audit:fname CTrunkDlg::OnPacket#ErrorOneOfAKind
type ErrorOneOfAKind struct {
	mode byte
}

func NewStorageErrorOneOfAKind(mode byte) ErrorOneOfAKind {
	return ErrorOneOfAKind{mode: mode}
}

func (m ErrorOneOfAKind) Mode() byte        { return m.mode }
func (m ErrorOneOfAKind) Operation() string { return StorageOperationWriter }
func (m ErrorOneOfAKind) String() string {
	return fmt.Sprintf("storage error one of a kind mode [%d]", m.mode)
}

func (m ErrorOneOfAKind) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte; sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *ErrorOneOfAKind) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
