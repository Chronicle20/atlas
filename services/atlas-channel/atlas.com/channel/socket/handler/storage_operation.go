package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

type StorageOperationMode string

const (
	StorageOperationHandle       = "StorageOperationHandle"
	StorageOperationModeRetrieve = "RETRIEVE_ASSET" // 4
	StorageOperationModeStore    = "STORE_ASSET"    // 5
	StorageOperationModeArrange  = "ARRANGE_ASSET"  // 6
	StorageOperationModeMeso     = "MESO"           // 7
	StorageOperationModeClose    = "CLOSE"
)

func StorageOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		mode := r.ReadByte()
		if isStorageOperation(l)(readerOptions, mode, StorageOperationModeRetrieve) {
			it := inventory.Type(r.ReadByte())
			slot := r.ReadByte()
			l.Debugf("Character [%d] is attempting to retrieve an item from storage inventory type [%d] slot [%d].", s.CharacterId(), it, slot)
			return
		}
		if isStorageOperation(l)(readerOptions, mode, StorageOperationModeStore) {
			slot := r.ReadInt16()
			itemId := r.ReadUint32()
			quantity := r.ReadUint16()
			l.Debugf("Character [%d] is attempting to store [%d] of item [%d] from inventory slot [%d].", s.CharacterId(), quantity, itemId, slot)
			return
		}
		if isStorageOperation(l)(readerOptions, mode, StorageOperationModeArrange) {
			l.Debugf("Character [%d] would like to arrange their storage.", s.CharacterId())
			return
		}
		if isStorageOperation(l)(readerOptions, mode, StorageOperationModeMeso) {
			amount := r.ReadInt32()
			if amount <= 0 {
				l.Debugf("Character [%d] is attempting to store [%d] mesos.", s.CharacterId(), amount)
			} else {
				l.Debugf("Character [%d] is attempting to retrieve [%d] mesos.", s.CharacterId(), amount)
			}
			return
		}
		if isStorageOperation(l)(readerOptions, mode, StorageOperationModeClose) {
			l.Debugf("Character [%d] exited storage.", s.CharacterId())
			return
		}
	}
}

func isStorageOperation(l logrus.FieldLogger) func(options map[string]interface{}, op byte, key StorageOperationMode) bool {
	return func(options map[string]interface{}, op byte, key StorageOperationMode) bool {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		res, ok := codes[string(key)].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}
		return byte(res) == op
	}
}
