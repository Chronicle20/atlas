package outbox

import (
	"encoding/json"
	"errors"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Message struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers map[string]string
}

func Enqueue(tx *gorm.DB, msg Message) error {
	if tx == nil {
		return errors.New("outbox: nil transaction")
	}
	if msg.Topic == "" {
		return errors.New("outbox: empty topic")
	}
	if len(msg.Key) == 0 {
		return errors.New("outbox: empty message key")
	}

	headers := datatypes.JSON([]byte("{}"))
	if len(msg.Headers) > 0 {
		b, err := json.Marshal(msg.Headers)
		if err != nil {
			return err
		}
		headers = datatypes.JSON(b)
	}

	ent := Entity{
		Topic:        msg.Topic,
		MessageKey:   msg.Key,
		MessageValue: msg.Value,
		Headers:      headers,
	}
	if err := tx.Create(&ent).Error; err != nil {
		return err
	}

	if isPostgres(tx) {
		if err := tx.Exec("SELECT pg_notify(?, ?)", notifyChannel, msg.Topic).Error; err != nil {
			return err
		}
	}
	return nil
}

const notifyChannel = "atlas_outbox_new"

func isPostgres(db *gorm.DB) bool {
	return db != nil && db.Dialector != nil && db.Dialector.Name() == "postgres"
}
