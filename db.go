package main

import (
	"fmt"
	"time"
)

// InsertMessageHistory inserts a message history record into the database.
func InsertMessageHistory(messageID, deviceJID, remoteJID, messageContent, messageType string, timestamp time.Time, sent bool) error {
	_, err := db.Exec(`
		INSERT INTO message_history (message_id, device_jid, remote_jid, message_type, message_content, timestamp, sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, messageID, deviceJID, remoteJID, messageContent, messageType, timestamp, sent)
	if err != nil {
		return fmt.Errorf("failed to insert message history: %w", err)
	}
	log.Infof("Inserted message history: %s, %s, %s, %s, %s", messageID, deviceJID, remoteJID, messageContent, timestamp)
	return nil
}

// InsertOrUpdateLastMessage inserts or updates the last message for a remote JID in the database.
func InsertOrUpdateLastMessage(messageID, deviceJID, remoteJID, messageContent, messageType string, timestamp time.Time, sent bool) error {
	_, err := db.Exec(`
		INSERT INTO last_messages (message_id, device_jid, remote_jid, message_type, message_content, timestamp, sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (remote_jid)
		DO UPDATE SET message_id = $1, device_jid = $2, message_type = $4, message_content = $5, timestamp = $6, sent = $7
	`, messageID, deviceJID, remoteJID, messageContent, messageType, timestamp, sent)
	if err != nil {
		return fmt.Errorf("failed to insert or update last message: %w", err)
	}
	log.Infof("Inserted or updated last message: %s, %s, %s, %s, %s", messageID, deviceJID, remoteJID, messageContent, timestamp)
	return nil
}
