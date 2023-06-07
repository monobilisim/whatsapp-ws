package main

import (
	"fmt"
	"time"
)

// insertMessageHistory inserts a message history record into the database.
//
// The message history includes details such as message ID, device JID, remote JID,
// message content, message type, timestamp, and sent status.
//
// Parameters:
//   - messageID: The ID of the message.
//   - device_jid: The JID of the device (current session owner, our JID).
//   - remote_jid: The JID of the remote user.
//   - messageContent: The content of the message.
//   - messageType: The type of the message.
//   - timestamp: The timestamp when the message was sent.
//   - sent: A boolean indicating if the message was sent.
//
// Returns:
//   - error: An error if the insertion fails, or nil if successful.
func insertMessageHistory(messageID, device_jid, remote_jid, messageContent, messageType string, timestamp time.Time, sent bool) error {
	_, err := db.Exec(`
		INSERT INTO message_history (message_id, device_jid, remote_jid, message_type, message_content, timestamp, sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, messageID, device_jid, remote_jid, messageContent, messageType, timestamp, sent)
	if err != nil {
		return fmt.Errorf("failed to insert message history: %w", err)
	}
	log.Infof("Inserted message history: %d, %s, %s, %s, %s", messageID, device_jid, remote_jid, messageContent, timestamp)
	return nil
}

// insertLastMessages inserts or updates the last message for a remote JID in the database.
//
// If the remote JID is not present in the database, it inserts a new record.
// If the remote JID is already present, it updates the existing record with the new message details.
//
// Parameters:
//   - messageID: The ID of the message.
//   - device_jid: The JID of the device.
//   - remote_jid: The JID of the remote user.
//   - messageContent: The content of the message.
//   - messageType: The type of the message.
//   - timestamp: The timestamp when the message was sent.
//   - sent: A boolean indicating if the message was sent.
//
// Returns:
//   - error: An error if the insertion/update fails, or nil if successful.
func insertLastMessages(messageID, device_jid, remote_jid, messageContent, messageType string, timestamp time.Time, sent bool) error {
	_, err := db.Exec(`
		INSERT INTO last_messages (message_id, device_jid, remote_jid, message_type, message_content, timestamp, sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (remote_jid)
		DO UPDATE SET message_id = $1, device_jid = $2, message_type = $4, message_content = $5, timestamp = $6, sent = $7
	`, messageID, device_jid, remote_jid, messageContent, messageType, timestamp, sent)
	if err != nil {
		return fmt.Errorf("failed to insert last messages: %w", err)
	}
	log.Infof("Inserted last messages: %d, %s, %s, %s, %s", messageID, device_jid, remote_jid, messageContent, timestamp)
	return nil
}
