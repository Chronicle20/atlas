package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestParcelResultArms pins the fifteen bodyless PARCEL result arms (Task 8):
// each body func resolves its mode from the tenant `operations` table (never
// a hard-coded literal) and the arm's Encode writes exactly that one byte.
// gms_v83 modes per docs/packets/dispatchers/parcel.yaml (Task 6). None of
// these fifteen keys carry a jms_v185 value in that table — see Ruling 5 in
// the task-8 brief: jms_v185's notice-slot mapping is genuinely
// underdetermined and deliberately left unset, so no jms_v185 case is
// exercised here.
func TestParcelResultArms(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)

	tests := []struct {
		name string
		key  string
		mode byte
		call func(options map[string]interface{}) []byte
	}{
		{"SEND_ENABLE_ACTIONS", ParcelOperationSendEnableActions, 0x09, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelSendEnableActionsBody()(l, ctx)(options)
		}},
		{"NOT_ENOUGH_MESOS", ParcelOperationNotEnoughMesos, 0x0A, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelNotEnoughMesosBody()(l, ctx)(options)
		}},
		{"INCORRECT_REQUEST", ParcelOperationIncorrectRequest, 0x0B, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelIncorrectRequestBody()(l, ctx)(options)
		}},
		{"NAME_DOES_NOT_EXIST", ParcelOperationNameDoesNotExist, 0x0C, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelNameDoesNotExistBody()(l, ctx)(options)
		}},
		{"SAME_ACCOUNT", ParcelOperationSameAccount, 0x0D, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelSameAccountBody()(l, ctx)(options)
		}},
		{"RECEIVER_STORAGE_FULL", ParcelOperationReceiverStorageFull, 0x0E, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelReceiverStorageFullBody()(l, ctx)(options)
		}},
		{"RECEIVER_UNABLE_TO_RECEIVE", ParcelOperationReceiverUnableToReceive, 0x0F, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelReceiverUnableToReceiveBody()(l, ctx)(options)
		}},
		{"SENDER_UNIQUE_CONFLICT", ParcelOperationSenderUniqueConflict, 0x10, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelSenderUniqueConflictBody()(l, ctx)(options)
		}},
		{"MESO_LIMIT", ParcelOperationMesoLimit, 0x11, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelMesoLimitBody()(l, ctx)(options)
		}},
		{"SUCCESSFULLY_SENT", ParcelOperationSuccessfullySent, 0x12, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelSuccessfullySentBody()(l, ctx)(options)
		}},
		{"UNKNOWN_ERROR", ParcelOperationUnknownError, 0x13, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelUnknownErrorBody()(l, ctx)(options)
		}},
		{"RECV_ENABLE_ACTIONS", ParcelOperationRecvEnableActions, 0x14, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelRecvEnableActionsBody()(l, ctx)(options)
		}},
		{"RECV_NO_FREE_SLOTS", ParcelOperationRecvNoFreeSlots, 0x15, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelRecvNoFreeSlotsBody()(l, ctx)(options)
		}},
		{"RECV_UNIQUE_CONFLICT", ParcelOperationRecvUniqueConflict, 0x16, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelRecvUniqueConflictBody()(l, ctx)(options)
		}},
		{"UNKNOWN_ERROR_2", ParcelOperationUnknownError2, 0x1C, func(options map[string]interface{}) []byte {
			l, _ := testlog.NewNullLogger()
			return ParcelUnknownError2Body()(l, ctx)(options)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			options := map[string]interface{}{
				"operations": map[string]interface{}{
					tc.key: float64(tc.mode),
				},
			}
			got := tc.call(options)
			want := []byte{tc.mode}
			if !bytes.Equal(got, want) {
				t.Errorf("%s: got % x want % x", tc.name, got, want)
			}
		})
	}

	t.Run("unresolved key falls back", func(t *testing.T) {
		l, _ := testlog.NewNullLogger()
		options := map[string]interface{}{
			"operations": map[string]interface{}{},
		}
		got := ParcelNotEnoughMesosBody()(l, ctx)(options)
		if len(got) != 1 {
			t.Fatalf("unresolved key: expected 1 byte fallback, got %d bytes: % x", len(got), got)
		}
		if got[0] != 99 {
			t.Fatalf("unresolved key: expected ResolveCode's documented 99 sentinel, got %d", got[0])
		}
	})
}
