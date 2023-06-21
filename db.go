package main

import (
	"fmt"
	"time"
)

// insertMessages inserts a messages record into the database.
//
// The messages includes details such as message ID, device JID, remote JID,
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
func insertMessages(messageID, device_jid, remote_jid, messageContent, messageType string, timestamp time.Time, sent bool) error {
	_, err := db.Exec(`
		INSERT INTO messages (message_id, device_jid, remote_jid, type, content, timestamp, sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, messageID, device_jid, remote_jid, messageContent, messageType, timestamp, sent)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	log.Infof("Inserted into messages: %d, %s, %s, %s, %s", messageID, device_jid, remote_jid, messageContent, timestamp)
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
		INSERT INTO last_messages (message_id, device_jid, remote_jid, type, content, timestamp, sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (remote_jid)
		DO UPDATE SET message_id = $1, device_jid = $2, type = $4, content = $5, timestamp = $6, sent = $7
	`, messageID, device_jid, remote_jid, messageContent, messageType, timestamp, sent)
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
