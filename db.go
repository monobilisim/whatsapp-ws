package main

import (
	"fmt"
	"time"
)

// InsertMessageHistory inserts a message history record into the database.
func insertMessages(messageID, deviceJID, remoteJID, messageContent, messageType string, timestamp time.Time, sent bool, fileName string, userIDInteger int) error {
	var userID *int
	if userIDInteger == -1 {
		userID = nil
	} else {
		userID = &userIDInteger
	}
	_, err := db.Exec(`
		INSERT INTO messages (message_id, device_jid, remote_jid, type, content, timestamp, sent, file_name, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `, messageID, deviceJID, remoteJID, messageType, messageContent, timestamp, sent, fileName, userID)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	log.Infof("Inserted into messages: %d, %s, %s, %s, %s", messageID, deviceJID, remoteJID, messageContent, timestamp)
	return nil
}

// InsertOrUpdateLastMessage inserts or updates the last message for a remote JID in the database.
func insertLastMessages(messageID, deviceJID, remoteJID, messageContent, messageType string, timestamp time.Time, sent bool, fileName string, userIDInteger int) error {
	var userID *int
	if userIDInteger == -1 {
		userID = nil
	} else {
		userID = &userIDInteger
	}
	_, err := db.Exec(`
		INSERT INTO last_messages (message_id, device_jid, remote_jid, type, content, timestamp, sent, file_name, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (remote_jid)
		DO UPDATE SET message_id = $1, device_jid = $2, type = $4, content = $5, timestamp = $6, sent = $7, file_name = $8, user_id = $9
	`, messageID, deviceJID, remoteJID, messageType, messageContent, timestamp, sent, fileName, userID)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	log.Infof("Inserted into last_messages: %d, %s, %s, %s, %s", messageID, deviceJID, remoteJID, messageContent, timestamp)
	return nil
}

func markMessageRead(messageID, remoteJID string, timestamp time.Time) error {
	_, err := db.Exec(`
		UPDATE messages SET read_at = $1 WHERE message_id = $2 AND remote_jid = $3
	`, timestamp, messageID, remoteJID)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	log.Infof("Marked message as read: %d, %s, %s", messageID, remoteJID, timestamp)
	return nil
}
