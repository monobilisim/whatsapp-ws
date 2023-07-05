package main

import (
	"fmt"
	"time"
)

// InsertMessageHistory inserts a message history record into the database.
func insertMessages(messageID, device_jid, remote_jid, messageContent, messageType string, timestamp time.Time, sent bool, extension string, fileName string, user_id_integer int) error {
	var user_id *int
	if user_id_integer == -1 {
		user_id = nil
	} else {
		user_id = &user_id_integer
	}
	_, err := db.Exec(`
		INSERT INTO messages (message_id, device_jid, remote_jid, type, content, timestamp, sent, extension, file_name, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `, messageID, device_jid, remote_jid, messageContent, messageType, timestamp, sent, extension, fileName, user_id)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	log.Infof("Inserted into messages: %d, %s, %s, %s, %s", messageID, device_jid, remote_jid, messageContent, timestamp)
	return nil
}

// InsertOrUpdateLastMessage inserts or updates the last message for a remote JID in the database.
func insertLastMessages(messageID, device_jid, remote_jid, messageContent, messageType string, timestamp time.Time, sent bool, extension string, fileName string, user_id_integer int) error {
	var user_id *int
	if user_id_integer == -1 {
		user_id = nil
	} else {
		user_id = &user_id_integer
	}
	_, err := db.Exec(`
		INSERT INTO last_messages (message_id, device_jid, remote_jid, type, content, timestamp, sent, extension, file_name, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (remote_jid)
		DO UPDATE SET message_id = $1, device_jid = $2, type = $4, content = $5, timestamp = $6, sent = $7, extension = $8, file_name = $9, user_id = $10
	`, messageID, device_jid, remote_jid, messageContent, messageType, timestamp, sent, extension, fileName, user_id)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	log.Infof("Inserted into last_messages: %d, %s, %s, %s, %s", messageID, device_jid, remote_jid, messageContent, timestamp)
	return nil
}

func markMessageRead(messageID, remote_jid string, timestamp time.Time) error {
	_, err := db.Exec(`
		UPDATE messages SET read_at = $1 WHERE message_id = $2 AND remote_jid = $3
	`, timestamp, messageID, remote_jid)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	log.Infof("Marked message as read: %d, %s, %s", messageID, remote_jid, timestamp)
	return nil
}
