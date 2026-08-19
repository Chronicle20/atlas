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
	ParcelOperationOpen      = "OPEN"
	ParcelOperationOpenQuick = "OPEN_QUICK"
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
