package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/parcel"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// Parcel dispatcher operation keys — docs/packets/dispatchers/parcel.yaml.
// This file adds the two keys Task 7 wires; Task 8 and Task 9 append the
// remaining 19 keys to this same const block.
const (
	ParcelOperationOpen                    = "OPEN"
	ParcelOperationOpenQuick               = "OPEN_QUICK"
	ParcelOperationSendEnableActions       = "SEND_ENABLE_ACTIONS"
	ParcelOperationNotEnoughMesos          = "NOT_ENOUGH_MESOS"
	ParcelOperationIncorrectRequest        = "INCORRECT_REQUEST"
	ParcelOperationNameDoesNotExist        = "NAME_DOES_NOT_EXIST"
	ParcelOperationSameAccount             = "SAME_ACCOUNT"
	ParcelOperationReceiverStorageFull     = "RECEIVER_STORAGE_FULL"
	ParcelOperationReceiverUnableToReceive = "RECEIVER_UNABLE_TO_RECEIVE"
	ParcelOperationSenderUniqueConflict    = "SENDER_UNIQUE_CONFLICT"
	ParcelOperationMesoLimit               = "MESO_LIMIT"
	ParcelOperationSuccessfullySent        = "SUCCESSFULLY_SENT"
	ParcelOperationUnknownError            = "UNKNOWN_ERROR"
	ParcelOperationRecvEnableActions       = "RECV_ENABLE_ACTIONS"
	ParcelOperationRecvNoFreeSlots         = "RECV_NO_FREE_SLOTS"
	ParcelOperationRecvUniqueConflict      = "RECV_UNIQUE_CONFLICT"
	ParcelOperationUnknownError2           = "UNKNOWN_ERROR_2"
)

// ParcelOpenBody resolves the OPEN mode from the tenant operations table and
// constructs the Open arm.
func ParcelOpenBody(quickEnabled bool, mailbox []parcel.Parcel, arrived []parcel.Parcel) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationOpen, func(mode byte) packet.Encoder {
		return NewParcelOpen(mode, quickEnabled, mailbox, arrived)
	})
}

// ParcelOpenQuickBody resolves the OPEN_QUICK mode from the tenant
// operations table and constructs the OpenQuick arm.
func ParcelOpenQuickBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationOpenQuick, func(mode byte) packet.Encoder {
		return NewParcelOpenQuick(mode)
	})
}

// ParcelSendEnableActionsBody resolves the SEND_ENABLE_ACTIONS mode from the
// tenant operations table and constructs the SendEnableActions arm.
func ParcelSendEnableActionsBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationSendEnableActions, func(mode byte) packet.Encoder {
		return NewParcelSendEnableActions(mode)
	})
}

// ParcelNotEnoughMesosBody resolves the NOT_ENOUGH_MESOS mode from the
// tenant operations table and constructs the NotEnoughMesos arm.
func ParcelNotEnoughMesosBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationNotEnoughMesos, func(mode byte) packet.Encoder {
		return NewParcelNotEnoughMesos(mode)
	})
}

// ParcelIncorrectRequestBody resolves the INCORRECT_REQUEST mode from the
// tenant operations table and constructs the IncorrectRequest arm.
func ParcelIncorrectRequestBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationIncorrectRequest, func(mode byte) packet.Encoder {
		return NewParcelIncorrectRequest(mode)
	})
}

// ParcelNameDoesNotExistBody resolves the NAME_DOES_NOT_EXIST mode from the
// tenant operations table and constructs the NameDoesNotExist arm.
func ParcelNameDoesNotExistBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationNameDoesNotExist, func(mode byte) packet.Encoder {
		return NewParcelNameDoesNotExist(mode)
	})
}

// ParcelSameAccountBody resolves the SAME_ACCOUNT mode from the tenant
// operations table and constructs the SameAccount arm.
func ParcelSameAccountBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationSameAccount, func(mode byte) packet.Encoder {
		return NewParcelSameAccount(mode)
	})
}

// ParcelReceiverStorageFullBody resolves the RECEIVER_STORAGE_FULL mode
// from the tenant operations table and constructs the ReceiverStorageFull
// arm.
func ParcelReceiverStorageFullBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationReceiverStorageFull, func(mode byte) packet.Encoder {
		return NewParcelReceiverStorageFull(mode)
	})
}

// ParcelReceiverUnableToReceiveBody resolves the
// RECEIVER_UNABLE_TO_RECEIVE mode from the tenant operations table and
// constructs the ReceiverUnableToReceive arm.
func ParcelReceiverUnableToReceiveBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationReceiverUnableToReceive, func(mode byte) packet.Encoder {
		return NewParcelReceiverUnableToReceive(mode)
	})
}

// ParcelSenderUniqueConflictBody resolves the SENDER_UNIQUE_CONFLICT mode
// from the tenant operations table and constructs the SenderUniqueConflict
// arm.
func ParcelSenderUniqueConflictBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationSenderUniqueConflict, func(mode byte) packet.Encoder {
		return NewParcelSenderUniqueConflict(mode)
	})
}

// ParcelMesoLimitBody resolves the MESO_LIMIT mode from the tenant
// operations table and constructs the MesoLimit arm.
func ParcelMesoLimitBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationMesoLimit, func(mode byte) packet.Encoder {
		return NewParcelMesoLimit(mode)
	})
}

// ParcelSuccessfullySentBody resolves the SUCCESSFULLY_SENT mode from the
// tenant operations table and constructs the SuccessfullySent arm.
func ParcelSuccessfullySentBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationSuccessfullySent, func(mode byte) packet.Encoder {
		return NewParcelSuccessfullySent(mode)
	})
}

// ParcelUnknownErrorBody resolves the UNKNOWN_ERROR mode from the tenant
// operations table and constructs the UnknownError arm.
func ParcelUnknownErrorBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationUnknownError, func(mode byte) packet.Encoder {
		return NewParcelUnknownError(mode)
	})
}

// ParcelRecvEnableActionsBody resolves the RECV_ENABLE_ACTIONS mode from
// the tenant operations table and constructs the RecvEnableActions arm.
func ParcelRecvEnableActionsBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationRecvEnableActions, func(mode byte) packet.Encoder {
		return NewParcelRecvEnableActions(mode)
	})
}

// ParcelRecvNoFreeSlotsBody resolves the RECV_NO_FREE_SLOTS mode from the
// tenant operations table and constructs the RecvNoFreeSlots arm.
func ParcelRecvNoFreeSlotsBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationRecvNoFreeSlots, func(mode byte) packet.Encoder {
		return NewParcelRecvNoFreeSlots(mode)
	})
}

// ParcelRecvUniqueConflictBody resolves the RECV_UNIQUE_CONFLICT mode from
// the tenant operations table and constructs the RecvUniqueConflict arm.
func ParcelRecvUniqueConflictBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationRecvUniqueConflict, func(mode byte) packet.Encoder {
		return NewParcelRecvUniqueConflict(mode)
	})
}

// ParcelUnknownError2Body resolves the UNKNOWN_ERROR_2 mode from the
// tenant operations table and constructs the UnknownError2 arm.
func ParcelUnknownError2Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ParcelOperationUnknownError2, func(mode byte) packet.Encoder {
		return NewParcelUnknownError2(mode)
	})
}
